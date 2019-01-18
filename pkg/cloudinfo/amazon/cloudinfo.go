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
	"errors"
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
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	// Cpu represents the cpu attribute for the recommender
	Cpu = "vcpu"
)

// Ec2Infoer encapsulates the data and operations needed to access external resources
type Ec2Infoer struct {
	pricingSvc   PricingSource
	prometheus   v1.API
	promQuery    string
	ec2Describer func(region string) Ec2Describer
}

// Ec2Describer interface for operations describing EC2 artifacts. (a subset of the Ec2 cli operations used by this app)
type Ec2Describer interface {
	DescribeAvailabilityZones(input *ec2.DescribeAvailabilityZonesInput) (*ec2.DescribeAvailabilityZonesOutput, error)
	DescribeImages(*ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error)
	DescribeSpotPriceHistoryPages(input *ec2.DescribeSpotPriceHistoryInput, fn func(*ec2.DescribeSpotPriceHistoryOutput, bool) bool) error
}

// NewEc2Infoer creates a new instance of the infoer
func NewEc2Infoer(ctx context.Context, promAddr, pq, awsAccessKeyId, awsSecretAccessKey string) (*Ec2Infoer, error) {
	log := logger.Extract(ctx)

	const defaultPricingRegion = "us-east-1"

	s, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			awsAccessKeyId,
			awsSecretAccessKey,
			"",
		),
		Region: aws.String(defaultPricingRegion),
	})
	if err != nil {
		log.WithError(err).Error("Error creating AWS session")
		return nil, err
	}

	var promApi v1.API
	if promAddr == "" {
		log.Warn("Prometheus API address is not set, fallback to direct API access.")
		promApi = nil
	} else {
		promClient, err := api.NewClient(api.Config{
			Address: promAddr,
		})
		if err != nil {
			log.WithError(err).Error("Error creating Prometheus client, fallback to direct API access.")
			promApi = nil
		} else {
			promApi = v1.NewAPI(promClient)
		}
	}

	return &Ec2Infoer{
		pricingSvc: NewPricingSource(s),
		prometheus: promApi,
		promQuery:  pq,
		ec2Describer: func(region string) Ec2Describer {
			return ec2.New(s, s.Config.WithRegion(region))
		},
	}, nil
}

// Initialize is not needed on EC2 because price info is changing frequently
func (e *Ec2Infoer) Initialize(ctx context.Context) (map[string]map[string]cloudinfo.Price, error) {
	return nil, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
// Delegates to the underlying PricingSource instance and unifies (transforms) the response
func (e *Ec2Infoer) GetAttributeValues(ctx context.Context, service, attribute string) (cloudinfo.AttrValues, error) {
	log := logger.Extract(ctx)
	apiValues, err := e.pricingSvc.GetAttributeValues(e.newAttributeValuesInput(attribute))
	if err != nil {
		return nil, err
	}
	var values cloudinfo.AttrValues
	for _, v := range apiValues.AttributeValues {
		dotValue := strings.Replace(*v.Value, ",", ".", -1)
		floatValue, err := strconv.ParseFloat(strings.Split(dotValue, " ")[0], 64)
		if err != nil {
			log.WithError(err).Warnf("Couldn't parse attribute Value: [%s=%s]", attribute, dotValue)
		}
		values = append(values, cloudinfo.AttrValue{
			Value:    floatValue,
			StrValue: *v.Value,
		})
	}
	log.Debugf("found %s values: %v", attribute, values)
	return values, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
// Delegates to the underlying PricingSource instance and performs transformations
func (e *Ec2Infoer) GetProducts(ctx context.Context, service, regionId string) ([]cloudinfo.VmInfo, error) {
	log := logger.Extract(ctx)

	missingAttributes := make(map[string][]string)
	var (
		missingGpu []string
		vms        []cloudinfo.VmInfo
		priceList  []aws.JSONValue
		err        error
	)
	log.Debug("Getting available instance types from AWS API.")

	if priceList, err = e.pricingSvc.GetPriceList(e.newGetProductsInput(regionId)); err != nil {
		return nil, err
	}

	for i, price := range priceList {
		pd, err := newPriceData(price)
		if err != nil {
			log.WithError(err).Warnf("could not extract pricing info for the item with index: [ %d ]", i)
			continue
		}

		instanceType, err := pd.GetDataForKey("instanceType")
		if err != nil {
			log.WithError(err).Warnf("could not retrieve instance type [%s]", instanceType)
			continue
		}
		cpusStr, err := pd.GetDataForKey(Cpu)
		if err != nil {
			missingAttributes[instanceType] = append(missingAttributes[instanceType], "cpu")
		}
		memStr, err := pd.GetDataForKey(cloudinfo.Memory)
		if err != nil {
			missingAttributes[instanceType] = append(missingAttributes[instanceType], "memory")
		}
		gpu, err := pd.GetDataForKey("gpu")
		if err != nil {
			missingGpu = append(missingGpu, instanceType)
		}
		odPriceStr, err := pd.GetOnDemandPrice()
		if err != nil {
			missingAttributes[instanceType] = append(missingAttributes[instanceType], "onDemandPrice")
		}
		ntwPerf, err := pd.GetDataForKey("networkPerformance")
		if err != nil {
			missingAttributes[instanceType] = append(missingAttributes[instanceType], "networkPerformance")
		}

		var currGen = true
		if currentGenStr, err := pd.GetDataForKey("currentGeneration"); err == nil {
			if strings.ToLower(currentGenStr) == "no" {
				currGen = false
			}
		}

		ntwMapper := newAmazonNetworkMapper()
		ntwPerfCat, err := ntwMapper.MapNetworkPerf(ntwPerf)
		if err != nil {
			log.WithError(err).Debug("could not get network performance category")
		}

		onDemandPrice, _ := strconv.ParseFloat(odPriceStr, 64)
		cpus, _ := strconv.ParseFloat(cpusStr, 64)
		mem, _ := strconv.ParseFloat(strings.Split(memStr, " ")[0], 64)
		gpus, _ := strconv.ParseFloat(gpu, 64)
		vm := cloudinfo.VmInfo{
			Type:          instanceType,
			OnDemandPrice: onDemandPrice,
			Cpus:          cpus,
			Mem:           mem,
			Gpus:          gpus,
			NtwPerf:       ntwPerf,
			NtwPerfCat:    ntwPerfCat,
			CurrentGen:    currGen,
			Attributes:    cloudinfo.Attributes(cpusStr, strings.Split(memStr, " ")[0], ntwPerfCat),
		}
		vms = append(vms, vm)
	}
	log.Warnf("instance types with missing attributes %s", missingAttributes)
	log.Debugf("instance types with missing gpu %s", missingGpu)
	if vms == nil {
		log.Debug("couldn't find any virtual machines to recommend")
	}

	if service == "eks" {
		vms = append(vms, cloudinfo.VmInfo{
			Type:          "EKS Control Plane",
			OnDemandPrice: 0.2,
		})
	}

	log.Debugf("found vms: %#v", vms)
	return vms, nil
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
func (pd *priceData) GetDataForKey(attr string) (string, error) {
	if value, ok := pd.attrMap[attr].(string); ok {
		return value, nil
	}
	return "", fmt.Errorf("could not get %s or could not cast %s to string", attr, attr)
}

func (pd *priceData) GetOnDemandPrice() (string, error) {
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
		return nil, fmt.Errorf("could not get map for key: [ %s ]", key)
	}

	remap, ok := rawMap.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("the value for key: [ %s ] could not be cast to map[string]interface{}", key)
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

// newAttributeValuesInput assembles a GetAttributeValuesInput instance for querying the provider
func (e *Ec2Infoer) newAttributeValuesInput(attr string) *pricing.GetAttributeValuesInput {
	return &pricing.GetAttributeValuesInput{
		ServiceCode:   aws.String("AmazonEC2"),
		AttributeName: aws.String(attr),
	}
}

// newAttributeValuesInput assembles a GetAttributeValuesInput instance for querying the provider
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
func (e *Ec2Infoer) GetRegions(ctx context.Context, service string) (map[string]string, error) {
	regionIdMap := make(map[string]string)
	for key, region := range endpoints.AwsPartition().Regions() {
		regionIdMap[key] = region.Description()
	}

	switch service {
	case "eks":
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
		return eksRegionIdMap, nil
	default:
		return regionIdMap, nil
	}
}

// GetZones returns the availability zones in a region
func (e *Ec2Infoer) GetZones(ctx context.Context, region string) ([]string, error) {

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
	return zones, nil
}

// HasShortLivedPriceInfo - Spot Prices are changing continuously on EC2
func (e *Ec2Infoer) HasShortLivedPriceInfo() bool {
	return true
}

func (e *Ec2Infoer) getSpotPricesFromPrometheus(ctx context.Context, region string) (map[string]cloudinfo.SpotPriceInfo, error) {
	log := logger.Extract(ctx)
	log.Debug("getting spot price averages from Prometheus API")
	priceInfo := make(map[string]cloudinfo.SpotPriceInfo)
	query := fmt.Sprintf(e.promQuery, region)
	log.Debugf("sending prometheus query: %s", query)
	result, err := e.prometheus.Query(context.Background(), query, time.Now())
	if err != nil {
		return nil, err
	}
	if result.String() == "" {
		log.Warn("Prometheus metric is empty")
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

func (e *Ec2Infoer) getCurrentSpotPrices(ctx context.Context, region string) (map[string]cloudinfo.SpotPriceInfo, error) {
	priceInfo := make(map[string]cloudinfo.SpotPriceInfo)
	err := e.ec2Describer(region).DescribeSpotPriceHistoryPages(&ec2.DescribeSpotPriceHistoryInput{
		StartTime:           aws.Time(time.Now()),
		ProductDescriptions: []*string{aws.String("Linux/UNIX")},
	}, func(history *ec2.DescribeSpotPriceHistoryOutput, lastPage bool) bool {
		for _, pe := range history.SpotPriceHistory {
			price, err := strconv.ParseFloat(*pe.SpotPrice, 64)
			if err != nil {
				logger.Extract(ctx).WithError(err).Error("couldn't parse spot price from history")
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
func (e *Ec2Infoer) GetCurrentPrices(ctx context.Context, region string) (map[string]cloudinfo.Price, error) {
	log := logger.Extract(ctx)
	var spotPrices map[string]cloudinfo.SpotPriceInfo
	var err error
	if e.prometheus != nil {
		spotPrices, err = e.getSpotPricesFromPrometheus(ctx, region)
		if err != nil {
			log.WithError(err).Warn("Couldn't get spot price info from Prometheus API, fallback to direct AWS API access.")
		}
	}

	if len(spotPrices) == 0 {
		log.Debug("getting current spot prices directly from the AWS API")
		spotPrices, err = e.getCurrentSpotPrices(ctx, region)
		if err != nil {
			log.WithError(err).Error("could not retrieve current prices")
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

// GetMemoryAttrName returns the provider representation of the memory attribute
func (e *Ec2Infoer) GetMemoryAttrName() string {
	return cloudinfo.Memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (e *Ec2Infoer) GetCpuAttrName() string {
	return Cpu
}

// GetServices returns the available services on the provider
func (e *Ec2Infoer) GetServices() ([]cloudinfo.ServiceDescriber, error) {
	services := []cloudinfo.ServiceDescriber{
		cloudinfo.NewService("compute"),
		cloudinfo.NewService("eks")}
	return services, nil
}

// GetService returns the given service description
func (e *Ec2Infoer) GetService(ctx context.Context, service string) (cloudinfo.ServiceDescriber, error) {
	svcs, err := e.GetServices()
	if err != nil {
		return nil, err
	}
	for _, sd := range svcs {
		if service == sd.ServiceName() {
			logger.Extract(ctx).Debugf("found service: %s", service)
			return sd, nil
		}
	}
	return nil, fmt.Errorf("the service [%s] is not supported", service)

}

// HasImages - Amazon support images
func (e *Ec2Infoer) HasImages() bool {
	return true
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (e *Ec2Infoer) GetServiceImages(service, region string) ([]cloudinfo.ImageDescriber, error) {
	imageDescribers := make([]cloudinfo.ImageDescriber, 0)

	if service == "eks" {
		for _, version := range []string{"1.10", "1.11"} {
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
	return nil, fmt.Errorf("GetServiceProducts - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (e *Ec2Infoer) GetServiceAttributes(region, service, attribute string) (cloudinfo.AttrValues, error) {
	return nil, fmt.Errorf("GetServiceAttributes - not yet implemented")
}

// GetVersions retrieves the kubernetes versions supported by the given service in the given region
func (e *Ec2Infoer) GetVersions(ctx context.Context, service, region string) ([]string, error) {
	switch service {
	case "eks":
		return []string{"1.10"}, nil
	default:
		return []string{}, nil
	}
}
