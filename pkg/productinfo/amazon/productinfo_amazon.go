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
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/banzaicloud/productinfo/pkg/logger"
	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const (
	// Cpu represents the cpu attribute for the recommender
	Cpu = "vcpu"
)

var (
	eksRegionIds = []string{"us-west-2", "us-east-1", "eu-west-1"}
)

// PricingSource list of operations for retrieving pricing information
// Decouples the pricing logic from the amazon api
type PricingSource interface {
	GetAttributeValues(input *pricing.GetAttributeValuesInput) (*pricing.GetAttributeValuesOutput, error)
	GetProducts(input *pricing.GetProductsInput) (*pricing.GetProductsOutput, error)
}

// Ec2Infoer encapsulates the data and operations needed to access external resources
type Ec2Infoer struct {
	pricingSvc   PricingSource
	session      *session.Session
	prometheus   v1.API
	promQuery    string
	ec2Describer func(region string) Ec2Describer
}

// Ec2Describer interface for operations describing EC2 artifacts. (a subset of the Ec2 cli operations iused by this app)
type Ec2Describer interface {
	DescribeAvailabilityZones(input *ec2.DescribeAvailabilityZonesInput) (*ec2.DescribeAvailabilityZonesOutput, error)
	DescribeSpotPriceHistoryPages(input *ec2.DescribeSpotPriceHistoryInput, fn func(*ec2.DescribeSpotPriceHistoryOutput, bool) bool) error
}

// NewEc2Infoer creates a new instance of the infoer
func NewEc2Infoer(ctx context.Context, promAddr string, pq string) (*Ec2Infoer, error) {
	log := logger.Extract(ctx)
	s, err := session.NewSession()
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

	const defaultPricingRegion = "us-east-1"
	return &Ec2Infoer{
		pricingSvc: pricing.New(s, aws.NewConfig().WithRegion(defaultPricingRegion)),
		session:    s,
		prometheus: promApi,
		promQuery:  pq,
		ec2Describer: func(region string) Ec2Describer {
			return ec2.New(s, aws.NewConfig().WithRegion(region))
		},
	}, nil
}

// NewConfig creates a new  Config instance and returns a pointer to it
func NewConfig() *aws.Config {
	return aws.NewConfig()
}

// Initialize is not needed on EC2 because price info is changing frequently
func (e *Ec2Infoer) Initialize(ctx context.Context) (map[string]map[string]productinfo.Price, error) {
	return nil, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
// Delegates to the underlying PricingSource instance and unifies (transforms) the response
func (e *Ec2Infoer) GetAttributeValues(ctx context.Context, service, attribute string) (productinfo.AttrValues, error) {
	log := logger.Extract(ctx)
	apiValues, err := e.pricingSvc.GetAttributeValues(e.newAttributeValuesInput(attribute))
	if err != nil {
		return nil, err
	}
	var values productinfo.AttrValues
	for _, v := range apiValues.AttributeValues {
		dotValue := strings.Replace(*v.Value, ",", ".", -1)
		floatValue, err := strconv.ParseFloat(strings.Split(dotValue, " ")[0], 64)
		if err != nil {
			log.WithError(err).Warnf("Couldn't parse attribute Value: [%s=%s]", attribute, dotValue)
		}
		values = append(values, productinfo.AttrValue{
			Value:    floatValue,
			StrValue: *v.Value,
		})
	}
	log.Debugf("found %s values: %v", attribute, values)
	return values, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
// Delegates to the underlying PricingSource instance and performs transformations
func (e *Ec2Infoer) GetProducts(ctx context.Context, service, regionId string) ([]productinfo.VmInfo, error) {
	log := logger.Extract(ctx)

	var vms []productinfo.VmInfo
	log.Debug("Getting available instance types from AWS API.")

	products, err := e.pricingSvc.GetProducts(e.newGetProductsInput(regionId))

	if err != nil {
		return nil, err
	}
	for i, price := range products.PriceList {
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
			log.WithError(err).Warnf("could not retrieve vcpu [%s]", cpusStr)
			continue
		}
		memStr, err := pd.GetDataForKey(productinfo.Memory)
		if err != nil {
			log.WithError(err).Warnf("could not retrieve memory [%s]", memStr)
			continue
		}
		gpu, err := pd.GetDataForKey("gpu")
		if err != nil {
			log.WithError(err).Warnf("could not retrieve gpu [%s]", gpu)
		}
		odPriceStr, err := pd.GetOnDemandPrice()
		if err != nil {
			log.WithError(err).Warnf("could not retrieve on demand price [%s]", odPriceStr)
			continue
		}
		ntwPerf, err := pd.GetDataForKey("networkPerformance")
		if err != nil {
			log.WithError(err).Warnf("could not parse network performance [%s]", ntwPerf)
			continue
		}

		var currGen = true
		if currentGenStr, err := pd.GetDataForKey("currentGeneration"); err == nil {
			if strings.ToLower(currentGenStr) == "no" {
				currGen = false
			}
		}

		onDemandPrice, _ := strconv.ParseFloat(odPriceStr, 64)
		cpus, _ := strconv.ParseFloat(cpusStr, 64)
		mem, _ := strconv.ParseFloat(strings.Split(memStr, " ")[0], 64)
		gpus, _ := strconv.ParseFloat(gpu, 64)
		vm := productinfo.VmInfo{
			Type:          instanceType,
			OnDemandPrice: onDemandPrice,
			Cpus:          cpus,
			Mem:           mem,
			Gpus:          gpus,
			NtwPerf:       ntwPerf,
			CurrentGen:    currGen,
		}
		vms = append(vms, vm)
	}
	if vms == nil {
		log.Debug("couldn't find any virtual machines to recommend")
	}

	if service == "eks" {
		vm := productinfo.VmInfo{
			Type:          "EKS Control Plane",
			OnDemandPrice: 0.2,
		}
		vms = append(vms, vm)
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
		},
	}
}

// GetRegions returns a map with available regions
// transforms the api representation into a "plain" map
func (e *Ec2Infoer) GetRegions(ctx context.Context, service string) (map[string]string, error) {
	switch service {
	case "eks":
		eksRegionMap := make(map[string]string)
		awsRegions := endpoints.AwsPartition().Regions()
		for _, regId := range eksRegionIds {
			eksRegionMap[regId] = awsRegions[regId].Description()
		}
		return eksRegionMap, nil
	default:
		regionIdMap := make(map[string]string)
		for key, region := range endpoints.AwsPartition().Regions() {
			regionIdMap[key] = region.Description()
		}
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

func (e *Ec2Infoer) getSpotPricesFromPrometheus(ctx context.Context, region string) (map[string]productinfo.SpotPriceInfo, error) {
	log := logger.Extract(ctx)
	log.Debug("getting spot price averages from Prometheus API")
	priceInfo := make(map[string]productinfo.SpotPriceInfo)
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
				priceInfo[instanceType] = make(productinfo.SpotPriceInfo)
			}
			priceInfo[instanceType][az] = price
		}
	}
	return priceInfo, nil
}

func (e *Ec2Infoer) getCurrentSpotPrices(ctx context.Context, region string) (map[string]productinfo.SpotPriceInfo, error) {
	priceInfo := make(map[string]productinfo.SpotPriceInfo)
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
				priceInfo[*pe.InstanceType] = make(productinfo.SpotPriceInfo)
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
func (e *Ec2Infoer) GetCurrentPrices(ctx context.Context, region string) (map[string]productinfo.Price, error) {
	log := logger.Extract(ctx)
	var spotPrices map[string]productinfo.SpotPriceInfo
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

	prices := make(map[string]productinfo.Price)
	for region, sp := range spotPrices {
		prices[region] = productinfo.Price{
			SpotPrice:     sp,
			OnDemandPrice: -1,
		}
	}
	return prices, nil
}

// GetMemoryAttrName returns the provider representation of the memory attribute
func (e *Ec2Infoer) GetMemoryAttrName() string {
	return productinfo.Memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (e *Ec2Infoer) GetCpuAttrName() string {
	return Cpu
}

// GetNetworkPerformanceMapper gets the ec2 specific network performance mapper implementation
func (e *Ec2Infoer) GetNetworkPerformanceMapper() (productinfo.NetworkPerfMapper, error) {
	nm := newEc2NetworkMapper()
	return &nm, nil
}

// GetServices returns the available services on the provider
func (e *Ec2Infoer) GetServices() ([]productinfo.ServiceDescriber, error) {
	services := []productinfo.ServiceDescriber{
		productinfo.NewService("compute"),
		productinfo.NewService("eks")}
	return services, nil
}

// GetService returns the given service description
func (e *Ec2Infoer) GetService(ctx context.Context, service string) (productinfo.ServiceDescriber, error) {
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

// GetServiceImages retrieves the images supported by the given service in the given region
func (e *Ec2Infoer) GetServiceImages(region, service string) ([]productinfo.ImageDescriber, error) {
	return nil, fmt.Errorf("GetServiceImages - not yet implemented")
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (e *Ec2Infoer) GetServiceProducts(region, service string) ([]productinfo.ProductDetails, error) {
	return nil, fmt.Errorf("GetServiceProducts - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (e *Ec2Infoer) GetServiceAttributes(region, service, attribute string) (productinfo.AttrValues, error) {
	return nil, fmt.Errorf("GetServiceAttributes - not yet implemented")
}
