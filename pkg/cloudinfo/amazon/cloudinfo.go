// Copyright © 2018 Banzai Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package amazon

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
)

const svcEks = "eks"

// Ec2Infoer encapsulates the data and operations needed to access external resources
type Ec2Infoer struct {
	pricingSvc   PricingSource
	prometheus   v1.API
	promQuery    string
	ec2Describer func(region string) Ec2Describer
	log          logur.Logger
}

// Ec2Describer interface for operations describing EC2 artifacts. (a subset of the Ec2 cli operations used by this app)
type Ec2Describer interface {
	DescribeAvailabilityZones(input *ec2.DescribeAvailabilityZonesInput) (*ec2.DescribeAvailabilityZonesOutput, error)
	DescribeImages(*ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error)
	DescribeSpotPriceHistoryPages(input *ec2.DescribeSpotPriceHistoryInput, fn func(*ec2.DescribeSpotPriceHistoryOutput, bool) bool) error
}

// NewAmazonInfoer builds an infoer instance based on the provided configuration
func NewAmazonInfoer(config Config, logger logur.Logger) (*Ec2Infoer, error) {
	const defaultPricingRegion = "us-east-1"

	providers := []credentials.Provider{
		&credentials.StaticProvider{Value: credentials.Value{
			AccessKeyID:     config.AccessKey,
			SecretAccessKey: config.SecretKey,
		}},
		&credentials.EnvProvider{},
		&credentials.SharedCredentialsProvider{
			Filename: config.SharedCredentialsFile,
			Profile:  config.Profile,
		},
	}

	s, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewChainCredentials(providers),
		Region:      aws.String(defaultPricingRegion),
	})
	if err != nil {
		logger.Error("failed to create AWS session")
		return nil, err
	}

	var promApi v1.API
	if config.PrometheusAddress == "" {
		logger.Warn("Prometheus API address is not set, fallback to direct API access.")
		promApi = nil
	} else {
		promClient, err := api.NewClient(api.Config{
			Address: config.PrometheusAddress,
		})
		if err != nil {
			logger.Error("failed to create Prometheus client, fallback to direct API access.")
			promApi = nil
		} else {
			promApi = v1.NewAPI(promClient)
		}
	}

	return &Ec2Infoer{
		pricingSvc: NewPricingSource(s),
		prometheus: promApi,
		promQuery:  config.PrometheusQuery,
		ec2Describer: func(region string) Ec2Describer {
			return ec2.New(s, s.Config.WithRegion(region))
		},
		log: logger,
	}, nil
}

// Initialize is not needed on EC2 because price info is changing frequently
func (e *Ec2Infoer) Initialize() (map[string]map[string]cloudinfo.Price, error) {
	return nil, nil
}

func (e *Ec2Infoer) GetVirtualMachines(region string) ([]cloudinfo.VmInfo, error) {
	logger := log.WithFields(e.log, map[string]interface{}{"region": region})
	logger.Debug("getting available instance types from AWS API")

	missingAttributes := make(map[string][]string)
	var (
		missingGpu []string
		vms        []cloudinfo.VmInfo
		priceList  []aws.JSONValue
		err        error
	)

	if priceList, err = e.pricingSvc.GetPriceList(e.newGetProductsInput(region)); err != nil {
		return nil, err
	}

	for i, price := range priceList {
		pd, err := newPriceData(price)
		if err != nil {
			logger.Warn("could not extract pricing info", map[string]interface{}{"itemindex": i})
			continue
		}

		instanceType, err := pd.getDataForKey("instanceType")
		if err != nil {
			logger.Warn("could not retrieve instance type", map[string]interface{}{"instancetype": instanceType})
			continue
		}
		instanceFamily, err := pd.getDataForKey("instanceFamily")
		if err != nil {
			logger.Warn("could not retrieve instance family", map[string]interface{}{"instanceFamily": instanceFamily})
		}
		cpusStr, err := pd.getDataForKey("vcpu")
		if err != nil {
			missingAttributes[instanceType] = append(missingAttributes[instanceType], "cpu")
		}
		memStr, err := pd.getDataForKey(cloudinfo.Memory)
		if err != nil {
			missingAttributes[instanceType] = append(missingAttributes[instanceType], "memory")
		}
		gpu, err := pd.getDataForKey("gpu")
		if err != nil {
			missingGpu = append(missingGpu, instanceType)
		}
		odPriceStr, err := pd.getOnDemandPrice()
		if err != nil {
			missingAttributes[instanceType] = append(missingAttributes[instanceType], "onDemandPrice")
		}
		ntwPerf, err := pd.getDataForKey("networkPerformance")
		if err != nil {
			missingAttributes[instanceType] = append(missingAttributes[instanceType], "networkPerformance")
		}

		var currGen = true
		if currentGenStr, err := pd.getDataForKey("currentGeneration"); err == nil {
			if strings.ToLower(currentGenStr) == "no" {
				currGen = false
			}
		}

		ntwMapper := newAmazonNetworkMapper()
		ntwPerfCat, err := ntwMapper.MapNetworkPerf(ntwPerf)
		if err != nil {
			logger.Debug(emperror.Wrap(err, "failed to get network performance category").Error(),
				map[string]interface{}{"instanceType": instanceType})
		}

		onDemandPrice, _ := strconv.ParseFloat(odPriceStr, 64)
		cpus, _ := strconv.ParseFloat(cpusStr, 64)
		mem, _ := strconv.ParseFloat(strings.Split(memStr, " ")[0], 64)
		gpus, _ := strconv.ParseFloat(gpu, 64)
		vm := cloudinfo.VmInfo{
			Category:      instanceFamily,
			Type:          instanceType,
			OnDemandPrice: onDemandPrice,
			Cpus:          cpus,
			Mem:           mem,
			Gpus:          gpus,
			NtwPerf:       ntwPerf,
			NtwPerfCat:    ntwPerfCat,
			CurrentGen:    currGen,
			Attributes:    cloudinfo.Attributes(cpusStr, strings.Split(memStr, " ")[0], ntwPerfCat, instanceFamily),
		}
		vms = append(vms, vm)
	}
	logger.Debug("instance types with missing attributes", map[string]interface{}{"missingAttrs": missingAttributes})
	logger.Debug("instance types with missing gpu", map[string]interface{}{"missingGPU": missingGpu})

	if vms == nil {
		logger.Debug("couldn't find any virtual machines to recommend")
	}

	logger.Debug("found vms", map[string]interface{}{"numberOfVms": len(vms)})
	return vms, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
// Delegates to the underlying PricingSource instance and performs transformations
func (e *Ec2Infoer) GetProducts(vms []cloudinfo.VmInfo, service, regionId string) ([]cloudinfo.VmInfo, error) {
	switch service {
	case svcEks:
		vms = append(vms, cloudinfo.VmInfo{
			Type:          "EKS Control Plane",
			OnDemandPrice: 0.2,
		})
		return vms, nil
	default:
		return nil, errors.Wrap(errors.New(service), "invalid service")
	}
}

type priceData struct {
	awsData aws.JSONValue
	attrMap map[string]interface{}
}

func newPriceData(prData aws.JSONValue) (*priceData, error) {
	pd := priceData{awsData: prData}

	// get the attributes map
	productMap, err := getMapForKey("product", pd.awsData)
	if err != nil {
		return nil, err
	}

	attrMap, err := getMapForKey("attributes", productMap)
	if err != nil {
		return nil, err
	}

	pd.attrMap = attrMap

	return &pd, nil
}
func (pd *priceData) getDataForKey(attr string) (string, error) {
	if value, ok := pd.attrMap[attr].(string); ok {
		return value, nil
	}
	return "", errors.Errorf("could not get %s or could not cast %s to string", attr, attr)
}

func (pd *priceData) getOnDemandPrice() (string, error) {
	termsMap, err := getMapForKey("terms", pd.awsData)
	if err != nil {
		return "", err
	}
	onDemandMap, err := getMapForKey("OnDemand", termsMap)
	if err != nil {
		return "", err
	}
	for _, term := range onDemandMap {
		priceDimensionsMap, err := getMapForKey("priceDimensions", term.(map[string]interface{}))
		if err != nil {
			return "", err
		}
		for _, dimension := range priceDimensionsMap {

			pricePerUnitMap, err := getMapForKey("pricePerUnit", dimension.(map[string]interface{}))
			if err != nil {
				return "", err
			}
			odPrice, ok := pricePerUnitMap["USD"].(string)
			if !ok {
				return "", errors.New("could not get on demand price or could not cast on demand price to string")
			}
			return odPrice, nil
		}
	}
	return "", nil
}

func getMapForKey(key string, srcMap map[string]interface{}) (map[string]interface{}, error) {
	rawMap, ok := srcMap[key]
	if !ok {
		return nil, errors.Wrap(errors.New(key), "could not get map for key")
	}

	remap, ok := rawMap.(map[string]interface{})
	if !ok {
		return nil, errors.Wrap(errors.New(key), "could not be cast to map[string]interface{} for key")
	}
	return remap, nil
}

// GetRegion gets the api specific region representation based on the provided id
func (e *Ec2Infoer) GetRegion(id string) *endpoints.Region {
	awsp := endpoints.AwsPartition()
	for _, r := range awsp.Regions() {
		if r.ID() == id {
			return &r
		}
	}
	return nil
}

// newAttributeValuesInput assembles a GetProductsInput instance for querying the provider
func (e *Ec2Infoer) newGetProductsInput(regionId string) *pricing.GetProductsInput {
	return &pricing.GetProductsInput{

		ServiceCode: aws.String("AmazonEC2"),
		Filters: []*pricing.Filter{
			{
				Type:  aws.String(pricing.FilterTypeTermMatch),
				Field: aws.String("operatingSystem"),
				Value: aws.String("Linux"),
			},
			{
				Type:  aws.String(pricing.FilterTypeTermMatch),
				Field: aws.String("location"),
				Value: aws.String(e.GetRegion(regionId).Description()),
			},
			{
				Type:  aws.String(pricing.FilterTypeTermMatch),
				Field: aws.String("tenancy"),
				Value: aws.String("shared"),
			},
			{
				Type:  aws.String(pricing.FilterTypeTermMatch),
				Field: aws.String("preInstalledSw"),
				Value: aws.String("NA"),
			},
			{
				Type:  aws.String(pricing.FilterTypeTermMatch),
				Field: aws.String("capacitystatus"),
				Value: aws.String("Used"),
			},
		},
	}
}

// GetRegions returns a map with available regions
// transforms the api representation into a "plain" map
func (e *Ec2Infoer) GetRegions(service string) (map[string]string, error) {
	logger := log.WithFields(e.log, map[string]interface{}{"service": service})
	logger.Debug("getting regions")

	regionIdMap := make(map[string]string)
	for key, region := range endpoints.AwsPartition().Regions() {
		regionIdMap[key] = region.Description()
	}

	switch service {
	case svcEks:
		eksRegionIdMap := make(map[string]string)

		eksRegionIdMap[endpoints.UsEast1RegionID] = "US East (N. Virginia)"
		eksRegionIdMap[endpoints.UsEast2RegionID] = "US East (Ohio)"
		eksRegionIdMap[endpoints.UsWest2RegionID] = "US West (Oregon)"
		eksRegionIdMap[endpoints.EuWest1RegionID] = "EU (Ireland)"
		eksRegionIdMap[endpoints.EuWest2RegionID] = "EU (London)"
		eksRegionIdMap[endpoints.EuWest3RegionID] = "EU (Paris)"
		eksRegionIdMap[endpoints.EuNorth1RegionID] = "EU (Stockholm)"
		eksRegionIdMap[endpoints.EuCentral1RegionID] = "EU (Frankfurt)"
		eksRegionIdMap[endpoints.ApNortheast1RegionID] = "Asia Pacific (Tokyo)"
		eksRegionIdMap[endpoints.ApNortheast2RegionID] = "Asia Pacific (Seoul)"
		eksRegionIdMap[endpoints.ApSoutheast1RegionID] = "Asia Pacific (Singapore)"
		eksRegionIdMap[endpoints.ApSoutheast2RegionID] = "Asia Pacific (Sydney)"
		eksRegionIdMap[endpoints.ApSouth1RegionID] = "Asia Pacific (Mumbai)"

		return eksRegionIdMap, nil
	case "_eks":
		input := &ec2.DescribeImagesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("name"),
					Values: []*string{aws.String("amazon-eks-node-1.10-v*")},
				},
				{
					Name:   aws.String("is-public"),
					Values: []*string{aws.String("true")},
				},
				{
					Name:   aws.String("state"),
					Values: []*string{aws.String("available")},
				},
			},
		}

		eksRegionIdMap := make(map[string]string)

		for key, value := range regionIdMap {
			images, err := e.ec2Describer(key).DescribeImages(input)
			if err != nil {
				return nil, err
			}
			if len(images.Images) != 0 {
				eksRegionIdMap[key] = value
			}
		}
		logger.Debug("found regions", map[string]interface{}{"numberOfRegions": len(eksRegionIdMap)})
		return eksRegionIdMap, nil
	default:
		logger.Debug("found regions", map[string]interface{}{"numberOfRegions": len(regionIdMap)})
		return regionIdMap, nil
	}
}

// GetZones returns the availability zones in a region
func (e *Ec2Infoer) GetZones(region string) ([]string, error) {
	logger := log.WithFields(e.log, map[string]interface{}{"region": region})
	logger.Debug("getting zones")

	var zones []string
	azs, err := e.ec2Describer(region).DescribeAvailabilityZones(&ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		return nil, err
	}
	for _, az := range azs.AvailabilityZones {
		if *az.State == ec2.AvailabilityZoneStateAvailable {
			zones = append(zones, *az.ZoneName)
		}
	}

	logger.Debug("found zones", map[string]interface{}{"numberOfZones": len(zones)})
	return zones, nil
}

// HasShortLivedPriceInfo - Spot Prices are changing continuously on EC2
func (e *Ec2Infoer) HasShortLivedPriceInfo() bool {
	return true
}

func (e *Ec2Infoer) getSpotPricesFromPrometheus(region string) (map[string]cloudinfo.SpotPriceInfo, error) {
	logger := log.WithFields(e.log, map[string]interface{}{"region": region})
	logger.Debug("getting spot price averages from Prometheus API")
	priceInfo := make(map[string]cloudinfo.SpotPriceInfo)
	query := fmt.Sprintf(e.promQuery, region)
	logger.Debug("sending prometheus query", map[string]interface{}{"query": query})
	result, err := e.prometheus.Query(context.Background(), query, time.Now())
	if err != nil {
		return nil, err
	}
	if result.String() == "" {
		logger.Warn("Prometheus metric is empty")
	} else {
		r := result.(model.Vector)
		for _, value := range r {
			instanceType := string(value.Metric["instance_type"])
			az := string(value.Metric["availability_zone"])
			price, err := strconv.ParseFloat(value.Value.String(), 64)
			if err != nil {
				return nil, err
			}
			if priceInfo[instanceType] == nil {
				priceInfo[instanceType] = make(cloudinfo.SpotPriceInfo)
			}
			priceInfo[instanceType][az] = price
		}
	}
	return priceInfo, nil
}

func (e *Ec2Infoer) getCurrentSpotPrices(region string) (map[string]cloudinfo.SpotPriceInfo, error) {
	logger := log.WithFields(e.log, map[string]interface{}{"region": region})
	priceInfo := make(map[string]cloudinfo.SpotPriceInfo)
	err := e.ec2Describer(region).DescribeSpotPriceHistoryPages(&ec2.DescribeSpotPriceHistoryInput{
		StartTime:           aws.Time(time.Now()),
		ProductDescriptions: []*string{aws.String("Linux/UNIX")},
	}, func(history *ec2.DescribeSpotPriceHistoryOutput, lastPage bool) bool {
		for _, pe := range history.SpotPriceHistory {
			price, err := strconv.ParseFloat(*pe.SpotPrice, 64)
			if err != nil {
				logger.Error("couldn't parse spot price from history")
				continue
			}
			if priceInfo[*pe.InstanceType] == nil {
				priceInfo[*pe.InstanceType] = make(cloudinfo.SpotPriceInfo)
			}
			priceInfo[*pe.InstanceType][*pe.AvailabilityZone] = price
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return priceInfo, nil
}

// GetCurrentPrices returns the current spot prices of every instance type in every availability zone in a given region
func (e *Ec2Infoer) GetCurrentPrices(region string) (map[string]cloudinfo.Price, error) {
	logger := log.WithFields(e.log, map[string]interface{}{"region": region})
	var spotPrices map[string]cloudinfo.SpotPriceInfo
	var err error
	if e.prometheus != nil {
		spotPrices, err = e.getSpotPricesFromPrometheus(region)
		if err != nil {
			logger.Warn("could not get spot price info from Prometheus API, fallback to direct AWS API access.")
		}
	}

	if len(spotPrices) == 0 {
		logger.Debug("getting current spot prices directly from the AWS API")
		spotPrices, err = e.getCurrentSpotPrices(region)
		if err != nil {
			logger.Error("failed to retrieve current spot prices")
			return nil, err
		}
	}

	prices := make(map[string]cloudinfo.Price)
	for instanceType, sp := range spotPrices {
		prices[instanceType] = cloudinfo.Price{
			SpotPrice:     sp,
			OnDemandPrice: -1,
		}
		for zone, price := range sp {
			metrics.ReportAmazonSpotPrice(region, zone, instanceType, price)
		}
	}
	return prices, nil
}

// HasImages - Amazon support images
func (e *Ec2Infoer) HasImages() bool {
	return true
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (e *Ec2Infoer) GetServiceImages(service, region string) ([]cloudinfo.Image, error) {
	imageDescribers := make([]cloudinfo.Image, 0)

	if service == "_eks" {
		for _, version := range []string{"1.12", "1.13"} {
			input := &ec2.DescribeImagesInput{
				Filters: []*ec2.Filter{
					{
						Name:   aws.String("name"),
						Values: []*string{aws.String("amazon-eks-gpu-node-" + version + "-v*")},
					},
					{
						Name:   aws.String("is-public"),
						Values: []*string{aws.String("true")},
					},
					{
						Name:   aws.String("state"),
						Values: []*string{aws.String("available")},
					},
				},
			}

			gpuImages, err := e.ec2Describer(region).DescribeImages(input)
			if err != nil {
				return nil, err
			}

			for _, image := range gpuImages.Images {
				imageDescribers = append(imageDescribers, cloudinfo.NewImage(*image.ImageId, version, true))
			}

			input = &ec2.DescribeImagesInput{
				Filters: []*ec2.Filter{
					{
						Name:   aws.String("name"),
						Values: []*string{aws.String("amazon-eks-node-" + version + "-v*")},
					},
					{
						Name:   aws.String("is-public"),
						Values: []*string{aws.String("true")},
					},
					{
						Name:   aws.String("state"),
						Values: []*string{aws.String("available")},
					},
				},
			}

			images, err := e.ec2Describer(region).DescribeImages(input)
			if err != nil {
				return nil, err
			}

			for _, image := range images.Images {
				imageDescribers = append(imageDescribers, cloudinfo.NewImage(*image.ImageId, version, false))
			}
		}
	}

	return imageDescribers, nil
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (e *Ec2Infoer) GetServiceProducts(region, service string) ([]cloudinfo.ProductDetails, error) {
	return nil, errors.New("GetServiceProducts - not yet implemented")
}

// GetVersions retrieves the kubernetes versions supported by the given service in the given region
func (e *Ec2Infoer) GetVersions(service, region string) ([]cloudinfo.LocationVersion, error) {
	switch service {
	case svcEks:
		return []cloudinfo.LocationVersion{cloudinfo.NewLocationVersion(region, []string{"1.10", "1.11", "1.12", "1.13"}, "1.13")}, nil
	default:
		return []cloudinfo.LocationVersion{}, nil
	}
}
