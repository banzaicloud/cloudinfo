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
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/banzaicloud/productinfo/pkg/logger"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"github.com/spf13/viper"
)

// OnDemandPriceFromJson parses json file
type OnDemandPriceFromJson struct {
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
	client *ecs.Client
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
		client: ecsClient,
	}, nil
}

// Initialize is not needed on Alibaba because price info is changing frequently
func (e *AlibabaInfoer) Initialize(ctx context.Context) (map[string]map[string]productinfo.Price, error) {
	return nil, nil
}

func (e *AlibabaInfoer) getCurrentSpotPrices(ctx context.Context, region string, zones []string) (map[string]productinfo.SpotPriceInfo, error) {
	log := logger.Extract(ctx)
	log.Debug("start retrieving spot price data")
	priceInfo := make(map[string]productinfo.SpotPriceInfo)

	var (
		alibabaAccessKeyId     = "alibaba-access-key-id"
		alibabaAccessKeySecret = "alibaba-access-key-secret"
	)

	testCli, _ := ecs.NewClientWithAccessKey(
		region, viper.GetString(alibabaAccessKeyId), viper.GetString(alibabaAccessKeySecret),
	)

	request := ecs.CreateDescribeSpotPriceHistoryRequest()
	request.RegionId = region
	request.NetworkType = "vpc"
	request.OSType = "linux"

	log.Debug("created new client")

	dataFromJson, err := getJson(viper.GetString(priceInfoUrl))
	if err != nil {
		return nil, err
	}

	for key := range dataFromJson.PricingInfo {
		values := strings.Split(key, "::")
		if values[0] == region && values[3] == "linux" {
			request.InstanceType = values[1]

			prices, err := testCli.DescribeSpotPriceHistory(request)
			if err != nil {
				log.WithField("region", region).WithError(err).Errorf("failed to get spot price history for instance type [%s].", values[1])
				continue
			}

			price := priceInfo[values[1]]
			spotPrice := make(productinfo.SpotPriceInfo)
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
	log.WithField("region", region).Debug("finished retrieving spot price data")
	return priceInfo, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
func (e *AlibabaInfoer) GetAttributeValues(ctx context.Context, attribute string) (productinfo.AttrValues, error) {
	log := logger.Extract(ctx)
	log.Debugf("getting %s values", attribute)

	values := make(productinfo.AttrValues, 0)
	valueSet := make(map[productinfo.AttrValue]interface{})

	regions, err := e.GetRegions(ctx, "compute")
	if err != nil {
		return nil, err
	}

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.RegionId = "eu-central-1"

	vmSizes, err := e.client.DescribeInstanceTypes(request)
	if err != nil {
		return nil, err
	}

	dataFromJson, err := getJson(viper.GetString(priceInfoUrl))
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
					case productinfo.Cpu:
						valueSet[productinfo.AttrValue{
							Value:    float64(instanceType.CpuCoreCount),
							StrValue: fmt.Sprintf("%v", instanceType.CpuCoreCount),
						}] = ""
					case productinfo.Memory:
						valueSet[productinfo.AttrValue{
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
	log.Debugf("found %s values: %v", attribute, values)
	return values, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
func (e *AlibabaInfoer) GetProducts(ctx context.Context, service, regionId string) ([]productinfo.VmInfo, error) {
	log := logger.Extract(ctx)
	log.Debug("getting product info")
	var vms []productinfo.VmInfo

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.RegionId = regionId

	vmSizes, err := e.client.DescribeInstanceTypes(request)
	if err != nil {
		return nil, err
	}

	dataFromJson, err := getJson(viper.GetString(priceInfoUrl))
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
						onDemandPrice, err := strconv.ParseFloat(price.Price, 64)
						if err != nil {
							return nil, err
						}
						vms = append(vms, productinfo.VmInfo{
							Type:          instanceType.InstanceTypeId,
							OnDemandPrice: onDemandPrice,
							Cpus:          float64(instanceType.CpuCoreCount),
							Mem:           instanceType.MemorySize,
							Gpus:          float64(instanceType.GPUAmount),
							NtwPerf:       fmt.Sprintf("%.1f Gbit/s", float64(instanceType.InstanceBandwidthRx)/1024000),
						})
					}
				}
			}

		}
	}
	log.Debugf("found vms: %#v", vms)
	return vms, nil
}

// GetZones returns the availability zones in a region
func (e *AlibabaInfoer) GetZones(ctx context.Context, region string) ([]string, error) {
	var zones []string

	request := ecs.CreateDescribeZonesRequest()
	request.RegionId = region
	response, err := e.client.DescribeZones(request)
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
	response, err := e.client.DescribeRegions(request)
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
	return true
}

// GetCurrentPrices returns the current spot prices of every instance type in every availability zone in a given region
func (e *AlibabaInfoer) GetCurrentPrices(ctx context.Context, region string) (map[string]productinfo.Price, error) {
	log := logger.Extract(ctx)
	var spotPrices map[string]productinfo.SpotPriceInfo
	var err error

	zones, err := e.GetZones(ctx, region)
	if err != nil {
		return nil, err
	}

	log.WithField("region", region).Debug("getting current spot prices directly from the API")
	spotPrices, err = e.getCurrentSpotPrices(ctx, region, zones)

	if err != nil {
		log.WithField("region", region).WithError(err).Error("could not retrieve current prices.")
		return nil, err
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
func (e *AlibabaInfoer) GetMemoryAttrName() string {
	return productinfo.Memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (e *AlibabaInfoer) GetCpuAttrName() string {
	return productinfo.Cpu
}

// GetNetworkPerformanceMapper returns the network performance mapper implementation for this provider
func (e *AlibabaInfoer) GetNetworkPerformanceMapper() (productinfo.NetworkPerfMapper, error) {
	nm := newAlibabaNetworkMapper()
	return nm, nil
}

func getJson(url string) (OnDemandPriceFromJson, error) {
	var myClient = &http.Client{Timeout: 10 * time.Second}
	var dataFromJson OnDemandPriceFromJson
	r, err := myClient.Get(url)
	if err != nil {
		return OnDemandPriceFromJson{}, err
	}
	defer r.Body.Close()

	json.NewDecoder(r.Body).Decode(&dataFromJson)

	return dataFromJson, nil
}

// GetServices returns the available services on the provider
func (e *AlibabaInfoer) GetServices() ([]productinfo.ServiceDescriber, error) {
	services := []productinfo.ServiceDescriber{
		productinfo.NewService("compute"),
		productinfo.NewService("acsk")}
	return services, nil
}

// GetService returns the given service description
func (e *AlibabaInfoer) GetService(ctx context.Context, service string) (productinfo.ServiceDescriber, error) {
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
func (e *AlibabaInfoer) GetServiceImages(region, service string) ([]productinfo.ImageDescriber, error) {
	return nil, fmt.Errorf("GetServiceImages - not yet implemented")
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (e *AlibabaInfoer) GetServiceProducts(region, service string) ([]productinfo.ProductDetails, error) {
	return nil, fmt.Errorf("GetServiceProducts - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (e *AlibabaInfoer) GetServiceAttributes(region, service, attribute string) (productinfo.AttrValues, error) {
	return nil, fmt.Errorf("GetServiceAttributes - not yet implemented")
}
