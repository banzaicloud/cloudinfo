package alibaba

import (
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/log"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"github.com/spf13/viper"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DataFromJson parses json file
type DataFromJson struct {
	Currency        string          `json:"currency"`
	Version         string          `json:"version"`
	PublicationDate string          `json:"publicationDate"`
	PricingInfo     map[string]Time `json:"pricingInfo"`
	Disclaimer      string          `json:"disclaimer"`
	Type            string          `json:"type"`
	Site            string          `json:"site"`
	Description     string          `json:"description"`
}

// Time contains time data from json
type Time struct {
	Hours  []Price `json:"hours"`
	Months []Price `json:"months"`
	Years  []Price `json:"years"`
}

// Price contains price data from json
type Price struct {
	Price  string `json:"price"`
	Period string `json:"period"`
}

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
func (e *AlibabaInfoer) Initialize() (map[string]map[string]productinfo.Price, error) {
	return nil, nil
}

func (e *AlibabaInfoer) getCurrentSpotPrices(region string, zones []string) (map[string]productinfo.SpotPriceInfo, error) {
	log.Debugf("start retrieving alibaba spot price data for region [%s]", region)
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

	log.Debugf("created new client for %s, %v", region, testCli)

	dataFromJson, err := getJson()
	if err != nil {
		return nil, err
	}

	for key := range dataFromJson.PricingInfo {
		values := strings.Split(key, "::")
		if values[0] == region && values[3] == "linux" {
			request.InstanceType = values[1]

			prices, err := testCli.DescribeSpotPriceHistory(request)
			if err != nil {
				log.Errorf("failed to get spot price history for provider [%s], region [%s], instance type [%s]. error: [%s]", "alibaba", region, values[1], err.Error())
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
	log.Debugf("finished retrieving alibaba spot price data for region [%s]", region)
	return priceInfo, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
func (e *AlibabaInfoer) GetAttributeValues(attribute string) (productinfo.AttrValues, error) {
	log.Debugf("getting %s values", attribute)

	values := make(productinfo.AttrValues, 0)
	valueSet := make(map[productinfo.AttrValue]interface{})

	regions, err := e.GetRegions()
	if err != nil {
		return nil, err
	}

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.RegionId = "eu-central-1"

	vmSizes, err := e.client.DescribeInstanceTypes(request)
	if err != nil {
		return nil, err
	}

	dataFromJson, err := getJson()
	if err != nil {
		return nil, err
	}

	instanceTypes := vmSizes.InstanceTypes.InstanceType
	for region := range regions {
		for _, v := range instanceTypes {
			for key := range dataFromJson.PricingInfo {
				values := strings.Split(key, "::")
				if values[0] == region && values[1] == v.InstanceTypeId {
					switch attribute {
					case productinfo.Cpu:
						valueSet[productinfo.AttrValue{
							Value:    float64(v.CpuCoreCount),
							StrValue: fmt.Sprintf("%v", v.CpuCoreCount),
						}] = ""
					case productinfo.Memory:
						valueSet[productinfo.AttrValue{
							Value:    v.MemorySize,
							StrValue: fmt.Sprintf("%v", v.MemorySize),
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
func (e *AlibabaInfoer) GetProducts(regionId string) ([]productinfo.VmInfo, error) {
	log.Debugf("getting product info [region=%s]", regionId)
	var vms []productinfo.VmInfo

	request := ecs.CreateDescribeInstanceTypesRequest()
	request.RegionId = "eu-central-1"

	vmSizes, err := e.client.DescribeInstanceTypes(request)
	if err != nil {
		return nil, err
	}

	dataFromJson, err := getJson()
	if err != nil {
		return nil, err
	}

	instanceTypes := vmSizes.InstanceTypes.InstanceType
	for _, instanceType := range instanceTypes {
		for key, prices := range dataFromJson.PricingInfo {
			for _, price := range prices.Hours {
				values := strings.Split(key, "::")
				if values[0] == regionId && values[1] == instanceType.InstanceTypeId {
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

// HasShortLivedPriceInfo - Spot Prices are changing continuously on Alibaba
func (e *AlibabaInfoer) HasShortLivedPriceInfo() bool {
	return true
}

// GetCurrentPrices returns the current spot prices of every instance type in every availability zone in a given region
func (e *AlibabaInfoer) GetCurrentPrices(region string) (map[string]productinfo.Price, error) {
	var spotPrices map[string]productinfo.SpotPriceInfo
	var err error

	zones, err := e.GetZones(region)
	if err != nil {
		return nil, err
	}

	log.Debug("getting current spot prices directly from the ECS API")
	spotPrices, err = e.getCurrentSpotPrices(region, zones)

	if err != nil {
		log.Errorf("could not retrieve current prices. region %s, error: %s", region, err.Error())
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

func getJson() (DataFromJson, error) {
	var myClient = &http.Client{Timeout: 10 * time.Second}
	var dataFromJson DataFromJson
	r, err := myClient.Get("https://g.alicdn.com/aliyun/ecs-price-info-intl/2.0.1/price/download/instancePrice.json?spm=0.6883001.price.1.741827my27myFB&file=instancePrice.json")
	if err != nil {
		return DataFromJson{}, err
	}
	defer r.Body.Close()

	json.NewDecoder(r.Body).Decode(&dataFromJson)

	return dataFromJson, nil
}
