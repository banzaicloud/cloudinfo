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
	"encoding/json"
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
)

// AlibabaInfoer encapsulates the data and operations needed to access external Alibaba resources
type AlibabaInfoer struct {
	client CommonDescriber
	log    logur.Logger
}

const (
	svcCompute = "compute"
	svcAck     = "ack"
)

// newInfoer creates a new instance of the Alibaba infoer
func newInfoer(regionId, accessKeyId, accessKeySecret string, log logur.Logger) (*AlibabaInfoer, error) {

	// Create client
	client, err := sdk.NewClientWithAccessKey(
		regionId,
		accessKeyId,
		accessKeySecret,
	)
	if err != nil {
		// Handle exceptions
		panic(err)
	}

	//client.GetConfig().WithAutoRetry(true)
	client.GetConfig().WithGoRoutinePoolSize(100)
	client.GetConfig().WithEnableAsync(true)
	client.GetConfig().WithDebug(true)
	client.GetConfig().WithMaxRetryTime(10)

	return &AlibabaInfoer{
		client: client,
		log:    log,
	}, nil
}

func NewAliInfoer(cfg Config, log logur.Logger) (*AlibabaInfoer, error) {
	return newInfoer(cfg.RegionId, cfg.AccessKeyId, cfg.AccessKeySecret, log)
}

// Initialize is not needed on Alibaba because price info is changing frequently
func (a *AlibabaInfoer) Initialize() (map[string]map[string]cloudinfo.Price, error) {
	return nil, nil
}

func (a *AlibabaInfoer) getCurrentSpotPrices(region string) (map[string]cloudinfo.SpotPriceInfo, error) {
	log := log.WithFields(a.log, map[string]interface{}{"region": region})
	log.Debug("start retrieving spot price data")
	priceInfo := make(map[string]cloudinfo.SpotPriceInfo)

	zones, err := a.getZones(region)
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		for _, instanceType := range zone.AvailableInstanceTypes.InstanceTypes {
			if priceInfo[instanceType] == nil {

				describeSpotPriceHistory, err := a.client.ProcessCommonRequest(a.describeSpotPriceHistoryRequest(region, instanceType))
				if err != nil {
					log.Error("failed to get spot price history", map[string]interface{}{"instancetype": instanceType})
					continue
				}

				response := &ecs.DescribeSpotPriceHistoryResponse{}

				err = json.Unmarshal(describeSpotPriceHistory.BaseResponse.GetHttpContentBytes(), response)
				if err != nil {
					return nil, err
				}

				spotPrice := make(cloudinfo.SpotPriceInfo, 0)

				priceTypes := response.SpotPrices.SpotPriceType
				for _, priceType := range priceTypes {
					if zone.ZoneId == priceType.ZoneId {
						spotPrice[zone.ZoneId] = priceType.SpotPrice
						break
					}
					priceInfo[instanceType] = spotPrice
				}
			}
		}
	}
	log.Debug("retrieved spot price data")
	return priceInfo, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
func (a *AlibabaInfoer) GetAttributeValues(service, attribute string) (cloudinfo.AttrValues, error) {
	log := log.WithFields(a.log, map[string]interface{}{"service": service, "attribute": attribute})
	log.Debug("retrieving attribute values")

	values := make(cloudinfo.AttrValues, 0)
	valueSet := make(map[cloudinfo.AttrValue]interface{})

	regions, err := a.GetRegions(service)
	if err != nil {
		return nil, err
	}

	instanceTypes, err := a.getInstanceTypes()
	if err != nil {
		return nil, err
	}

	for region := range regions {
		zones, err := a.getZones(region)
		if err != nil {
			return nil, err
		}
		for _, zone := range zones {
			for _, instanceType := range instanceTypes {
				for _, resourcesInfo := range zone.AvailableResources.ResourcesInfo {
					for _, availableInstanceType := range resourcesInfo.InstanceTypes.SupportedInstanceType {
						if availableInstanceType == instanceType.InstanceTypeId {
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
		}
	}

	for attr := range valueSet {
		values = append(values, attr)
	}

	log.Debug("found attribute values", map[string]interface{}{"numberOfValues": len(values)})
	return values, nil
}

func (a *AlibabaInfoer) getZones(region string) ([]ecs.Zone, error) {
	describeZones, err := a.client.ProcessCommonRequest(a.describeZonesRequest(region))
	if err != nil {
		return nil, emperror.Wrap(err, "DescribeZones API call problem")
	}

	response := &ecs.DescribeZonesResponse{}

	err = json.Unmarshal(describeZones.BaseResponse.GetHttpContentBytes(), response)
	if err != nil {
		return nil, err
	}

	return response.Zones.Zone, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
func (a *AlibabaInfoer) GetProducts(service, regionId string) ([]cloudinfo.VmInfo, error) {
	log := log.WithFields(a.log, map[string]interface{}{"service": service, "region": regionId})
	log.Debug("getting product info")
	vms := make([]cloudinfo.VmInfo, 0)

	instanceTypes, err := a.getInstanceTypes()
	if err != nil {
		return nil, err
	}

	availableZones, err := a.getZones(regionId)
	if err != nil {
		return nil, err
	}

	for _, instanceType := range instanceTypes {
		zones := make([]string, 0)
		for _, zone := range availableZones {
			for _, resourcesInfo := range zone.AvailableResources.ResourcesInfo {
				for _, availableInstanceType := range resourcesInfo.InstanceTypes.SupportedInstanceType {
					if availableInstanceType == instanceType.InstanceTypeId {
						zones = append(zones, zone.ZoneId)
						break
					}
				}
			}
		}
		if len(zones) > 0 {

			ntwMapper := newAlibabaNetworkMapper()
			ntwPerf := fmt.Sprintf("%.1f Gbit/s", float64(instanceType.InstanceBandwidthRx)/1024000)
			ntwPerfCat, err := ntwMapper.MapNetworkPerf(ntwPerf)
			if err != nil {
				log.Debug(emperror.Wrap(err, "failed to get network performance category").Error(),
					map[string]interface{}{"instanceType": instanceType.InstanceTypeId})
			}

			vms = append(vms, cloudinfo.VmInfo{
				Type:       instanceType.InstanceTypeId,
				Cpus:       float64(instanceType.CpuCoreCount),
				Mem:        instanceType.MemorySize,
				Gpus:       float64(instanceType.GPUAmount),
				NtwPerf:    ntwPerf,
				NtwPerfCat: ntwPerfCat,
				Zones:      zones,
				Attributes: cloudinfo.Attributes(fmt.Sprint(instanceType.CpuCoreCount), fmt.Sprint(instanceType.MemorySize), ntwPerfCat),
			})
		}
	}

	virtualMachines, err := a.getOnDemandPrice(vms, regionId)
	if err != nil {
		return nil, err
	}

	log.Debug("found vms", map[string]interface{}{"numberOfVms": len(virtualMachines)})
	return virtualMachines, nil
}

func (a *AlibabaInfoer) getInstanceTypes() ([]ecs.InstanceType, error) {
	describeInstanceTypes, err := a.client.ProcessCommonRequest(a.describeInstanceTypesRequest())
	if err != nil {
		return nil, emperror.Wrap(err, "DescribeInstanceTypes API call problem")
	}

	response := &ecs.DescribeInstanceTypesResponse{}

	err = json.Unmarshal(describeInstanceTypes.BaseResponse.GetHttpContentBytes(), response)
	if err != nil {
		return nil, err
	}

	return response.InstanceTypes.InstanceType, nil
}

func (a *AlibabaInfoer) getOnDemandPrice(vms []cloudinfo.VmInfo, region string) ([]cloudinfo.VmInfo, error) {
	price := make(map[string]float64, 0)
	vmsWithPrice := make([]cloudinfo.VmInfo, 0)

	instanceTypes := make([]string, 0)

	for _, vm := range vms {

		instanceTypes = append(instanceTypes, vm.Type)

		if len(instanceTypes) == 50 {
			resp50vm, err := a.getPrice(instanceTypes, region)
			if err != nil {
				return nil, err
			}

			switch resp50vm.Code {
			case "Success":
				for i, moduleDetail := range resp50vm.Data.ModuleDetails.ModuleDetail {
					price[instanceTypes[i]] = moduleDetail.OriginalCost
				}
			case "InternalError":
				for i := 0; i < 5; i++ {
					resp10vm, err := a.getPrice(instanceTypes[10*i:10*(i+1)], region)
					if err != nil {
						return nil, err
					}

					switch resp10vm.Code {
					case "Success":
						for n, moduleDetail := range resp10vm.Data.ModuleDetails.ModuleDetail {
							price[instanceTypes[10*i+n]] = moduleDetail.OriginalCost
						}
					case "InternalError":
						for n := 0; n < 10; n++ {
							resp1vm, err := a.getPrice([]string{instanceTypes[10*i+n]}, region)
							if err != nil {
								return nil, err
							}
							if resp1vm.Code == "Success" {
								for n, moduleDetail := range resp1vm.Data.ModuleDetails.ModuleDetail {
									price[instanceTypes[10*i+n]] = moduleDetail.OriginalCost
								}
							}
						}
					}
				}
			case "NotAuthorized":
				return nil, errors.New("user needs AliyunBSSReadOnlyAccess permission")
			default:
				return nil, errors.Errorf("unknown error code: %s", resp50vm.Code)
			}

			instanceTypes = make([]string, 0)
		}
	}

	if len(instanceTypes) != 0 {
		resp, err := a.getPrice(instanceTypes, region)
		if err != nil {
			return nil, err
		}

		switch resp.Code {
		case "Success":
			for i, moduleDetail := range resp.Data.ModuleDetails.ModuleDetail {
				price[instanceTypes[i]] = moduleDetail.OriginalCost
			}
		case "InternalError":
			for i := 0; i < len(instanceTypes); i++ {
				resp1vm, err := a.getPrice([]string{instanceTypes[i]}, region)
				if err != nil {
					return nil, err
				}
				if resp1vm.Code == "Success" {
					for n, moduleDetail := range resp1vm.Data.ModuleDetails.ModuleDetail {
						price[instanceTypes[n]] = moduleDetail.OriginalCost
					}
				}
			}
		case "NotAuthorized":
			return nil, errors.New("user needs AliyunBSSReadOnlyAccess permission")
		default:
			return nil, errors.Errorf("unknown error code: %s", resp.Code)
		}
	}

	for _, vm := range vms {
		vmsWithPrice = append(vmsWithPrice, cloudinfo.VmInfo{
			Type:          vm.Type,
			OnDemandPrice: price[vm.Type],
			Cpus:          vm.Cpus,
			Mem:           vm.Mem,
			Gpus:          vm.Gpus,
			NtwPerf:       vm.NtwPerf,
			NtwPerfCat:    vm.NtwPerfCat,
			Zones:         vm.Zones,
			Attributes:    vm.Attributes,
		})
	}

	return vmsWithPrice, nil
}

func (a *AlibabaInfoer) getPrice(instanceTypes []string, region string) (bssopenapi.GetPayAsYouGoPriceResponse, error) {
	response := &bssopenapi.GetPayAsYouGoPriceResponse{}

	getPayAsYouGoPrice, err := a.client.ProcessCommonRequest(a.getPayAsYouGoPriceRequest(region, instanceTypes))
	if err != nil {
		return bssopenapi.GetPayAsYouGoPriceResponse{}, err
	}

	err = json.Unmarshal(getPayAsYouGoPrice.BaseResponse.GetHttpContentBytes(), response)
	if err != nil {
		return bssopenapi.GetPayAsYouGoPriceResponse{}, err
	}

	return *response, nil
}

// GetZones returns the availability zones in a region
func (a *AlibabaInfoer) GetZones(region string) ([]string, error) {
	log := log.WithFields(a.log, map[string]interface{}{"region": region})
	log.Debug("getting zones")

	var zones []string

	availableZones, err := a.getZones(region)
	if err != nil {
		return nil, err
	}

	for _, zone := range availableZones {
		zones = append(zones, zone.ZoneId)
	}

	log.Debug("found zones", map[string]interface{}{"numberOfZones": len(zones)})
	return zones, nil
}

// GetRegions returns a map with available regions
func (a *AlibabaInfoer) GetRegions(service string) (map[string]string, error) {
	log := log.WithFields(a.log, map[string]interface{}{"service": service})
	log.Debug("getting regions")

	describeRegions, err := a.client.ProcessCommonRequest(a.describeRegionsRequest())
	if err != nil {
		return nil, emperror.Wrap(err, "DescribeRegions API call problem")
	}

	response := &ecs.DescribeRegionsResponse{}

	err = json.Unmarshal(describeRegions.BaseResponse.GetHttpContentBytes(), response)
	if err != nil {
		return nil, err
	}

	var regionIdMap = make(map[string]string)
	for _, region := range response.Regions.Region {
		regionIdMap[region.RegionId] = region.LocalName
	}

	log.Debug("found regions", map[string]interface{}{"numberOfRegions": len(regionIdMap)})
	return regionIdMap, nil
}

// HasShortLivedPriceInfo - Spot Prices are changing continuously on Alibaba
func (a *AlibabaInfoer) HasShortLivedPriceInfo() bool {
	return false
}

// GetCurrentPrices returns the current spot prices of every instance type in every availability zone in a given region
func (a *AlibabaInfoer) GetCurrentPrices(region string) (map[string]cloudinfo.Price, error) {
	var spotPrices map[string]cloudinfo.SpotPriceInfo
	var err error

	spotPrices, err = a.getCurrentSpotPrices(region)
	if err != nil {
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
func (a *AlibabaInfoer) GetMemoryAttrName() string {
	return cloudinfo.Memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (a *AlibabaInfoer) GetCpuAttrName() string {
	return cloudinfo.Cpu
}

// GetServices returns the available services on the provider
func (a *AlibabaInfoer) GetServices() ([]cloudinfo.Service, error) {
	services := []cloudinfo.Service{
		cloudinfo.NewService(svcCompute),
		cloudinfo.NewService(svcAck)}
	return services, nil
}

// HasImages - Alibaba doesn't support images
func (a *AlibabaInfoer) HasImages() bool {
	return false
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (a *AlibabaInfoer) GetServiceImages(service, region string) ([]cloudinfo.Image, error) {
	return nil, errors.New("GetServiceImages - not yet implemented")
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (a *AlibabaInfoer) GetServiceProducts(region, service string) ([]cloudinfo.ProductDetails, error) {
	return nil, errors.New("GetServiceImages - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (a *AlibabaInfoer) GetServiceAttributes(region, service, attribute string) (cloudinfo.AttrValues, error) {
	return nil, errors.New("GetServiceImages - not yet implemented")
}

// GetVersions retrieves the kubernetes versions supported by the given service in the given region
func (a *AlibabaInfoer) GetVersions(service, region string) ([]string, error) {
	switch service {
	case svcAck:
		return []string{"1.11.2"}, nil
	default:
		return []string{}, nil
	}
}
