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

package alibaba

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/goph/emperror"
	"github.com/spf13/viper"
)

// OnDemandPrice contains price data from json
type OnDemandPrice struct {
	Currency        string                   `json:"currency"`
	Version         string                   `json:"version"`
	PublicationDate string                   `json:"publicationDate"`
	PricingInfo     map[string]TimeUnitPrice `json:"pricingInfo"`
	Disclaimer      string                   `json:"disclaimer"`
	Type            string                   `json:"type"`
	Site            string                   `json:"site"`
	Description     string                   `json:"description"`
}

// TimeUnitPrice contains time data from json
type TimeUnitPrice struct {
	Hours  []Price `json:"hours"`
	Months []Price `json:"months"`
	Years  []Price `json:"years"`
}

// Price contains price data from json
type Price struct {
	Price  string `json:"price"`
	Period string `json:"period"`
}

var priceInfoUrl = "alibaba-price-info-url"

// AlibabaInfoer encapsulates the data and operations needed to access external Alibaba resources
type AlibabaInfoer struct {
	ecsClient      EcsSource
	priceRetriever PriceRetriever
	spotClient     func(region string) EcsSource
}

// EcsSource list of operations for retrieving ecs information
type EcsSource interface {
	DescribeInstanceTypes(request *ecs.DescribeInstanceTypesRequest) (response *ecs.DescribeInstanceTypesResponse, err error)
	DescribeSpotPriceHistory(request *ecs.DescribeSpotPriceHistoryRequest) (response *ecs.DescribeSpotPriceHistoryResponse, err error)
	DescribeZones(request *ecs.DescribeZonesRequest) (response *ecs.DescribeZonesResponse, err error)
	DescribeRegions(request *ecs.DescribeRegionsRequest) (response *ecs.DescribeRegionsResponse, err error)
}

type onDemandPrice struct{}

const (
	svcCompute = "compute"
	svcAck     = "ack"
)

// PriceRetriever collects on demand prices from a json file
// TODO revisit this later when API starts supporting DescribePrice(request *DescribePriceRequest) (response *DescribePriceResponse, err error) method
type PriceRetriever interface {
	getOnDemandPrice(url string) (OnDemandPrice, error)
}

// NewAlibabaInfoer creates a new instance of the Alibaba infoer
func NewAlibabaInfoer(regionId, accessKeyId, accessKeySecret string) (*AlibabaInfoer, error) {

	// Create an ECS client
	ecsClient, err := ecs.NewClientWithAccessKey(
		regionId,
		accessKeyId,
		accessKeySecret,
	)

	//ecsClient.GetConfig().WithAutoRetry(true)
	ecsClient.GetConfig().WithGoRoutinePoolSize(100)
	ecsClient.GetConfig().WithEnableAsync(true)
	ecsClient.GetConfig().WithDebug(true)

	if err != nil {
		// Handle exceptions
		panic(err)
	}

	return &AlibabaInfoer{
		ecsClient:      ecsClient,
		priceRetriever: &onDemandPrice{},
		spotClient: func(region string) EcsSource {
			return ecsClient
		},
	}, nil
}

func NewAliInfoer(ctx context.Context, cfg Config) (*AlibabaInfoer, error) {
	return NewAlibabaInfoer(cfg.RegionId, cfg.AccessKeyId, cfg.AccessKeySecret)
}

// Initialize is not needed on Alibaba because price info is changing frequently
func (e *AlibabaInfoer) Initialize(ctx context.Context) (map[string]map[string]cloudinfo.Price, error) {
	return nil, nil
}

func (e *AlibabaInfoer) getCurrentSpotPrices(ctx context.Context, region string, zones []string) (map[string]cloudinfo.SpotPriceInfo, error) {
	log := logger.Extract(ctx)
	log.Debug("start retrieving spot price data")
	priceInfo := make(map[string]cloudinfo.SpotPriceInfo)

	request := ecs.CreateDescribeSpotPriceHistoryRequest()
	request.RegionId = region
	request.NetworkType = "vpc"
	request.OSType = "linux"

	log.Debug("created new client")

	dataFromJson, err := e.priceRetriever.getOnDemandPrice(viper.GetString(priceInfoUrl))
	if err != nil {
		return nil, err
	}

	for key := range dataFromJson.PricingInfo {
		values := strings.Split(key, "::")
		if values[0] == region && values[3] == "linux" {
			request.InstanceType = values[1]

			prices, err := e.spotClient(region).DescribeSpotPriceHistory(request)
			if err != nil {
				log.Error("failed to get spot price history", map[string]interface{}{"instancetype": values[1]})
				continue
			}

			price := priceInfo[values[1]]
			spotPrice := make(cloudinfo.SpotPriceInfo)
			priceTypes := prices.SpotPrices.SpotPriceType
			for _, priceType := range priceTypes {
				for _, z := range zones {
					if z == priceType.ZoneId {
						spotPrice[z] = priceType.SpotPrice
						price = spotPrice
						break
					}
				}
				priceInfo[values[1]] = price
			}
		}
	}
	log.Debug("retrieved spot price data", map[string]interface{}{"region": region})
	return priceInfo, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
func (e *AlibabaInfoer) GetAttributeValues(ctx context.Context, service, attribute string) (cloudinfo.AttrValues, error) {
	log := logger.Extract(ctx)
	log.Debug("retrieving attribute values", map[string]interface{}{"attribute": attribute})

	values := make(cloudinfo.AttrValues, 0)
	valueSet := make(map[cloudinfo.AttrValue]interface{})

	regions, err := e.GetRegions(ctx, service)
	if err != nil {
		return nil, err
	}

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.RegionId = "eu-central-1"

	vmSizes, err := e.ecsClient.DescribeInstanceTypes(request)
	if err != nil {
		return nil, err
	}

	dataFromJson, err := e.priceRetriever.getOnDemandPrice(viper.GetString(priceInfoUrl))
	if err != nil {
		return nil, err
	}

	instanceTypes := vmSizes.InstanceTypes.InstanceType
	for region := range regions {
		for _, instanceType := range instanceTypes {
			for key := range dataFromJson.PricingInfo {
				// The key structure is 'RegionId::InstanceType::NetworkType::OSType::IoOptimized'"
				values := strings.Split(key, "::")
				if values[0] == region && values[1] == instanceType.InstanceTypeId {
					switch attribute {
					case cloudinfo.Cpu:
						valueSet[cloudinfo.AttrValue{
							Value:    float64(instanceType.CpuCoreCount),
							StrValue: fmt.Sprintf("%v", instanceType.CpuCoreCount),
						}] = ""
					case cloudinfo.Memory:
						valueSet[cloudinfo.AttrValue{
							Value:    instanceType.MemorySize,
							StrValue: fmt.Sprintf("%v", instanceType.MemorySize),
						}] = ""
					}
				}
			}
		}
	}

	for attr := range valueSet {
		values = append(values, attr)
	}

	log.Debug("found attribute values", map[string]interface{}{"attribute": attribute, "values": fmt.Sprintf("%v", values)})
	return values, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
func (e *AlibabaInfoer) GetProducts(ctx context.Context, service, regionId string) ([]cloudinfo.VmInfo, error) {
	log := logger.Extract(ctx)
	log.Debug("getting product info")
	var vms []cloudinfo.VmInfo

	request := ecs.CreateDescribeInstanceTypesRequest()

	vmSizes, err := e.ecsClient.DescribeInstanceTypes(request)
	if err != nil {
		return nil, err
	}

	dataFromJson, err := e.priceRetriever.getOnDemandPrice(viper.GetString(priceInfoUrl))
	if err != nil {
		return nil, err
	}

	zoneRequest := ecs.CreateDescribeZonesRequest()
	request.RegionId = regionId

	zoneResponse, err := e.ecsClient.DescribeZones(zoneRequest)
	if err != nil {
		return nil, err
	}

	instanceTypes := vmSizes.InstanceTypes.InstanceType
	for _, instanceType := range instanceTypes {
		for key, prices := range dataFromJson.PricingInfo {
			for _, price := range prices.Hours {
				if price.Period == "1" {
					// The key structure is 'RegionId::InstanceType::NetworkType::OSType::IoOptimized'"
					values := strings.Split(key, "::")
					if values[0] == regionId && values[1] == instanceType.InstanceTypeId && values[3] == "linux" {
						var zones []string
						for _, zone := range zoneResponse.Zones.Zone {
							for _, availableVm := range zone.AvailableInstanceTypes.InstanceTypes {
								if instanceType.InstanceTypeId == availableVm {
									zones = append(zones, zone.ZoneId)
								}
							}
						}
						ntwMapper := newAlibabaNetworkMapper()
						ntwPerf := fmt.Sprintf("%.1f Gbit/s", float64(instanceType.InstanceBandwidthRx)/1024000)
						ntwPerfCat, err := ntwMapper.MapNetworkPerf(ntwPerf)
						if err != nil {
							log.Debug("could not get network performance category")
						}

						onDemandPrice, err := strconv.ParseFloat(price.Price, 64)
						if err != nil {
							return nil, err
						}
						vms = append(vms, cloudinfo.VmInfo{
							Type:          instanceType.InstanceTypeId,
							OnDemandPrice: onDemandPrice,
							Cpus:          float64(instanceType.CpuCoreCount),
							Mem:           instanceType.MemorySize,
							Gpus:          float64(instanceType.GPUAmount),
							NtwPerf:       ntwPerf,
							NtwPerfCat:    ntwPerfCat,
							Zones:         zones,
							Attributes:    cloudinfo.Attributes(fmt.Sprint(instanceType.CpuCoreCount), fmt.Sprint(instanceType.MemorySize), ntwPerfCat),
						})
					}
				}
			}
		}
	}

	log.Debug("found vms", map[string]interface{}{"vms": fmt.Sprintf("%v", vms)})
	return vms, nil
}

// GetZones returns the availability zones in a region
func (e *AlibabaInfoer) GetZones(ctx context.Context, region string) ([]string, error) {
	var zones []string

	request := ecs.CreateDescribeZonesRequest()
	request.RegionId = region
	response, err := e.ecsClient.DescribeZones(request)
	if err != nil {
		return nil, err
	}
	zonesResponse := response.Zones.Zone

	for _, zone := range zonesResponse {
		zones = append(zones, zone.ZoneId)
	}
	return zones, nil
}

// GetRegions returns a map with available regions
func (e *AlibabaInfoer) GetRegions(ctx context.Context, service string) (map[string]string, error) {
	var RegionIdMap = make(map[string]string)
	request := ecs.CreateDescribeRegionsRequest()
	request.AcceptLanguage = "en-US"
	response, err := e.ecsClient.DescribeRegions(request)
	if err != nil {
		return nil, err
	}
	regions := response.Regions.Region
	for _, region := range regions {
		RegionIdMap[region.RegionId] = region.LocalName
	}
	return RegionIdMap, nil
}

// HasShortLivedPriceInfo - Spot Prices are changing continuously on Alibaba
func (e *AlibabaInfoer) HasShortLivedPriceInfo() bool {
	return false
}

// GetCurrentPrices returns the current spot prices of every instance type in every availability zone in a given region
func (e *AlibabaInfoer) GetCurrentPrices(ctx context.Context, region string) (map[string]cloudinfo.Price, error) {
	log := logger.Extract(ctx)
	var spotPrices map[string]cloudinfo.SpotPriceInfo
	var err error

	zones, err := e.GetZones(ctx, region)
	if err != nil {
		return nil, err
	}

	log.Debug("getting current spot prices directly from the API", map[string]interface{}{"region": region})
	spotPrices, err = e.getCurrentSpotPrices(ctx, region, zones)
	if err != nil {
		log.Error("could not retrieve current prices.", map[string]interface{}{"region": region})
		return nil, err
	}

	prices := make(map[string]cloudinfo.Price)
	for instanceType, sp := range spotPrices {
		prices[instanceType] = cloudinfo.Price{
			SpotPrice:     sp,
			OnDemandPrice: -1,
		}
		for zone, price := range sp {
			metrics.ReportAlibabaSpotPrice(region, zone, instanceType, price)
		}
	}

	return prices, nil
}

// GetMemoryAttrName returns the provider representation of the memory attribute
func (e *AlibabaInfoer) GetMemoryAttrName() string {
	return cloudinfo.Memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (e *AlibabaInfoer) GetCpuAttrName() string {
	return cloudinfo.Cpu
}

func (p *onDemandPrice) getOnDemandPrice(url string) (OnDemandPrice, error) {
	var myClient = &http.Client{Timeout: 10 * time.Second}
	var dataFromJson OnDemandPrice
	r, err := myClient.Get(url)
	if err != nil {
		return OnDemandPrice{}, err
	}
	defer r.Body.Close()

	err = json.NewDecoder(r.Body).Decode(&dataFromJson)
	if err != nil {
		return OnDemandPrice{}, err
	}

	return dataFromJson, nil
}

// GetServices returns the available services on the provider
func (e *AlibabaInfoer) GetServices() ([]cloudinfo.ServiceDescriber, error) {
	services := []cloudinfo.ServiceDescriber{
		cloudinfo.NewService(svcCompute),
		cloudinfo.NewService(svcAck)}
	return services, nil
}

// GetService returns the given service description
func (e *AlibabaInfoer) GetService(ctx context.Context, service string) (cloudinfo.ServiceDescriber, error) {
	svcs, err := e.GetServices()
	if err != nil {
		return nil, err
	}
	for _, sd := range svcs {
		if service == sd.ServiceName() {
			return sd, nil
		}
	}
	return nil, emperror.With(errors.New("the service is not supported"), "service", service)
}

// HasImages - Alibaba doesn't support images
func (e *AlibabaInfoer) HasImages() bool {
	return false
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (e *AlibabaInfoer) GetServiceImages(service, region string) ([]cloudinfo.ImageDescriber, error) {
	return nil, errors.New("GetServiceImages - not yet implemented")
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (e *AlibabaInfoer) GetServiceProducts(region, service string) ([]cloudinfo.ProductDetails, error) {
	return nil, errors.New("GetServiceImages - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (e *AlibabaInfoer) GetServiceAttributes(region, service, attribute string) (cloudinfo.AttrValues, error) {
	return nil, errors.New("GetServiceImages - not yet implemented")
}

// GetVersions retrieves the kubernetes versions supported by the given service in the given region
func (e *AlibabaInfoer) GetVersions(ctx context.Context, service, region string) ([]string, error) {
	switch service {
	case svcAck:
		return []string{"1.11.2"}, nil
	default:
		return []string{}, nil
	}
}
