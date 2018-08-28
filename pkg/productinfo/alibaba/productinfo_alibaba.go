package alibaba

import (
	"errors"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/log"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"sync"
)

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
	if err != nil {
		// Handle exceptions
		panic(err)
	}

	return &AlibabaInfoer{
		client: ecsClient,
	}, nil
}

// Initialize downloads and parses the ECS price list on Alibaba Cloud
func (e *AlibabaInfoer) Initialize() (map[string]map[string]productinfo.Price, error) {
	log.Debug("initializing Alibaba price info")

	var allPrices = struct {
		sync.RWMutex
		prices map[string]map[string]productinfo.Price
	}{prices: make(map[string]map[string]productinfo.Price)}
	var waitGroup sync.WaitGroup
	zonesInRegions := make(map[string][]string)

	regions, err := e.GetRegions()
	if err != nil {
		return nil, err
	}

	req := ecs.CreateDescribeInstanceTypesRequest()
	req.RegionId = "eu-central-1"

	vmSizes, err := e.client.DescribeInstanceTypes(req)
	if err != nil {
		return nil, err
	}
	instanceTypes := vmSizes.InstanceTypes.InstanceType
	for region := range regions {
		zones, err := e.GetZones(region)
		if err != nil {
			return nil, err
		}
		zonesInRegions[region] = zones
		waitGroup.Add(1)
		go func(region string, instanceTypes []ecs.InstanceType, zonesInRegions map[string][]string) {
			defer waitGroup.Done()
			for _, instanceType := range instanceTypes {
				request := ecs.CreateDescribeSpotPriceHistoryRequest()
				request.RegionId = region
				request.NetworkType = "vpc"
				request.InstanceType = instanceType.InstanceTypeId
				request.OSType = "linux"
				allPrices.RLock()
				if allPrices.prices[region] == nil {
					allPrices.RUnlock()
					allPrices.Lock()
					allPrices.prices[region] = make(map[string]productinfo.Price)
					allPrices.Unlock()
					allPrices.RLock()
				}
				allPrices.RUnlock()

				prices, err := e.client.DescribeSpotPriceHistory(request)
				if err != nil {
					return
				}

				allPrices.RLock()
				price := allPrices.prices[region][instanceType.InstanceTypeId]
				allPrices.RUnlock()
				spotPrice := make(productinfo.SpotPriceInfo)
				priceTypes := prices.SpotPrices.SpotPriceType
				for _, priceType := range priceTypes {
					price.OnDemandPrice = priceType.OriginPrice
					for _, z := range zonesInRegions[region] {
						if z == priceType.ZoneId {
							spotPrice[z] = priceType.SpotPrice
							price.SpotPrice = spotPrice
							break
						}
					}

					allPrices.Lock()
					allPrices.prices[region][priceType.InstanceType] = price
					allPrices.Unlock()
				}
			}
		}(region, instanceTypes, zonesInRegions)
	}
	waitGroup.Wait()

	log.Debug("finished initializing Alibaba price info")
	return allPrices.prices, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
func (e *AlibabaInfoer) GetAttributeValues(attribute string) (productinfo.AttrValues, error) {
	return nil, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
func (e *AlibabaInfoer) GetProducts(regionId string) ([]productinfo.VmInfo, error) {
	log.Debugf("getting product info [region=%s]", regionId)
	var vms []productinfo.VmInfo

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.RegionId = regionId

	vmSizes, err := e.client.DescribeInstanceTypes(request)
	if err != nil {
		return nil, err
	}
	instanceTypes := vmSizes.InstanceTypes.InstanceType
	for _, instanceType := range instanceTypes {
		vms = append(vms, productinfo.VmInfo{
			Type:    instanceType.InstanceTypeId,
			Cpus:    float64(instanceType.CpuCoreCount),
			Mem:     instanceType.MemorySize,
			Gpus:    float64(instanceType.GPUAmount),
			NtwPerf: fmt.Sprintf("%.1f Gbit/s", float64(instanceType.InstanceBandwidthRx)/1024000),
		})
	}
	log.Debugf("found vms: %#v", vms)
	return vms, nil
}

// GetZones returns the availability zones in a region
func (e *AlibabaInfoer) GetZones(region string) ([]string, error) {
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
func (e *AlibabaInfoer) GetRegions() (map[string]string, error) {
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

// HasShortLivedPriceInfo - Alibaba doesn't have frequently changing prices
func (e *AlibabaInfoer) HasShortLivedPriceInfo() bool {
	return false
}

// GetCurrentPrices retrieves all the price info in a region
func (e *AlibabaInfoer) GetCurrentPrices(region string) (map[string]productinfo.Price, error) {
	return nil, errors.New("alibaba prices cannot be queried on the fly")
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
