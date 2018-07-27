package gce

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/banzaicloud/productinfo/pkg/productinfo"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2/google"
	billing "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi/transport"
)

const (
	cpu    = "cpu"
	memory = "memory"
)

var regionNames = map[string]string{
	"asia-east1":              "Asia Pacific (Taiwan)",
	"asia-northeast1":         "Asia Pacific (Tokyo)",
	"asia-south1":             "Asia Pacific (Mumbai)",
	"asia-southeast1":         "Asia Pacific (Singapore)",
	"australia-southeast1":    "Asia Pacific (Sydney)",
	"europe-north1":           "EU (Finland)",
	"europe-west1":            "EU (Belgium)",
	"europe-west2":            "EU (London)",
	"europe-west3":            "EU (Frankfurt)",
	"europe-west4":            "EU (Netherlands)",
	"northamerica-northeast1": "Canada (Montréal)",
	"southamerica-east1":      "South America (São Paulo)",
	"us-central1":             "US Central (Iowa)",
	"us-east1":                "US East (South Carolina)",
	"us-east4":                "US East (Northern Virginia)",
	"us-west1":                "US West (Oregon)",
}

// GceInfoer encapsulates the data and operations needed to access external resources
type GceInfoer struct {
	cbSvc              *billing.APIService
	computeSvc         *compute.Service
	projectId          string
	cpuRegex           *regexp.Regexp
	resourceGroupRegex *regexp.Regexp
}

// NewGceInfoer creates a new instance of the infoer
func NewGceInfoer(apiKey string) (*GceInfoer, error) {
	defaultCredential, err := google.FindDefaultCredentials(context.Background(), compute.ComputeScope, billing.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	client, err := google.DefaultClient(context.Background(), compute.ComputeScope, billing.CloudPlatformScope)
	if err != nil {
		return nil, err
	}
	computeSvc, err := compute.New(client)
	if err != nil {
		return nil, err
	}

	billingSvc, err := billing.New(&http.Client{
		Transport: &transport.APIKey{Key: apiKey},
	})
	if err != nil {
		return nil, err
	}

	cpuReg, _ := regexp.Compile("\\d+ VCPU")
	rgReg, _ := regexp.Compile("^[a-z]+\\d+")

	return &GceInfoer{
		cbSvc:              billingSvc,
		computeSvc:         computeSvc,
		projectId:          defaultCredential.ProjectID,
		cpuRegex:           cpuReg,
		resourceGroupRegex: rgReg,
	}, nil
}

// Initialize downloads and parses the SKU list of the Compute Engine service
func (g *GceInfoer) Initialize() (map[string]map[string]productinfo.Price, error) {

	log.Debug("initializing GCE price info")
	allPrices := make(map[string]map[string]productinfo.Price)

	svcList, err := g.cbSvc.Services.List().Do()
	if err != nil {
		return nil, err
	}

	var compEngId string
	for _, svc := range svcList.Services {
		if svc.DisplayName == "Compute Engine" {
			compEngId = svc.Name
		}
	}

	log.Debugf("gce compute engine service id: %s", compEngId)

	zonesInRegions := make(map[string][]string)
	regions, err := g.GetRegions()
	if err != nil {
		return nil, err
	}
	for r := range regions {
		zones, err := g.GetZones(r)
		if err != nil {
			return nil, err
		}
		zonesInRegions[r] = zones
	}

	log.Debugf("queried zones and regions: %v", zonesInRegions)

	err = g.cbSvc.Services.Skus.List(compEngId).Pages(context.Background(), func(response *billing.ListSkusResponse) error {
		for _, sku := range response.Skus {
			if sku.Category.ResourceFamily != "Compute" {
				continue
			} else {
				if strings.Contains(sku.Description, "CPU") {

					vcpus := strings.Split(g.cpuRegex.FindString(sku.Description), " ")[0]
					resourceGroup := strings.ToLower(sku.Category.ResourceGroup)
					prefix := g.resourceGroupRegex.FindString(resourceGroup)
					instanceType := fmt.Sprintf("%s-%s", strings.Join(strings.SplitAfter(resourceGroup, prefix), "-"), vcpus)

					switch instanceType {
					case "f1-micro-":
						instanceType = "f1-micro"
					case "g1-small-1":
						instanceType = "g1-small"
					}
					if len(sku.PricingInfo) != 1 {
						return fmt.Errorf("pricing info not parsable, %d pricing info entries are returned", len(sku.PricingInfo))
					}
					pricingInfo := sku.PricingInfo[0]
					var priceInUsd float64
					for _, tr := range pricingInfo.PricingExpression.TieredRates {
						priceInUsd += float64(tr.UnitPrice.Units) + float64(tr.UnitPrice.Nanos)*1e-9
					}

					for _, region := range sku.ServiceRegions {
						if allPrices[region] == nil {
							allPrices[region] = make(map[string]productinfo.Price)
						}
						price := allPrices[region][instanceType]
						if sku.Category.UsageType == "OnDemand" {
							price.OnDemandPrice = priceInUsd
						} else {
							spotPrice := make(productinfo.SpotPriceInfo)
							for _, z := range zonesInRegions[region] {
								spotPrice[z] = priceInUsd
							}
							price.SpotPrice = spotPrice
						}
						allPrices[region][instanceType] = price
						log.Debugf("price info added: [region=%s, machinetype=%s, price=%v]", region, instanceType, price)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Debug("finished initializing GCE price info")
	return allPrices, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
// Queries the Google Cloud Compute API's machine type list endpoint
func (g *GceInfoer) GetAttributeValues(attribute string) (productinfo.AttrValues, error) {

	log.Debugf("getting %s values", attribute)

	values := make(productinfo.AttrValues, 0)
	valueSet := make(map[productinfo.AttrValue]interface{})

	err := g.computeSvc.MachineTypes.AggregatedList(g.projectId).Pages(context.TODO(), func(allMts *compute.MachineTypeAggregatedList) error {
		for _, scope := range allMts.Items {
			for _, mt := range scope.MachineTypes {
				switch attribute {
				case cpu:
					valueSet[productinfo.AttrValue{
						Value:    float64(mt.GuestCpus),
						StrValue: fmt.Sprintf("%v", mt.GuestCpus),
					}] = ""
				case memory:
					valueSet[productinfo.AttrValue{
						Value:    float64(mt.MemoryMb) / 1024,
						StrValue: fmt.Sprintf("%v", mt.MemoryMb),
					}] = ""
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list machine types: %v", err.Error())
	}

	for attr := range valueSet {
		values = append(values, attr)
	}

	log.Debugf("found %s values: %v", attribute, values)
	return values, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
// Queries the Google Cloud Compute API's machine type list endpoint and CloudBilling's sku list endpoint
func (g *GceInfoer) GetProducts(regionId string) ([]productinfo.VmInfo, error) {
	log.Debugf("getting product info [region=%s]", regionId)
	var vms []productinfo.VmInfo
	var ntwPerf string
	zones, err := g.GetZones(regionId)
	if err != nil {
		return nil, err
	}
	// TODO: check if all machine types are available in every regions??
	err = g.computeSvc.MachineTypes.List(g.projectId, zones[0]).Pages(context.TODO(), func(allMts *compute.MachineTypeList) error {
		for _, mt := range allMts.Items {
			if mt.GuestCpus < 1 {
				// minimum 1 Gbps network performance for each virtual machine
				ntwPerf = strconv.Itoa(1)
			} else if mt.GuestCpus > 8 {
				// theoretical maximum of 16 Gbps for each virtual machine
				ntwPerf = strconv.Itoa(16)
			} else {
				// each vCPU has a 2 Gbps egress cap for peak performance
				ntwPerf = strconv.Itoa(int(mt.GuestCpus * 2))
			}
			vms = append(vms, productinfo.VmInfo{
				Type:    mt.Name,
				Cpus:    float64(mt.GuestCpus),
				Mem:     float64(mt.MemoryMb) / 1024,
				NtwPerf: ntwPerf,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	log.Debugf("found vms: %#v", vms)
	return vms, nil
}

// GetRegions returns a map with available regions transforms the api representation into a "plain" map
func (g *GceInfoer) GetRegions() (map[string]string, error) {
	log.Debugf("getting regions")
	regionIdMap := make(map[string]string)
	regionList, err := g.computeSvc.Regions.List(g.projectId).Do()
	if err != nil {
		return nil, err
	}
	for _, region := range regionList.Items {
		description := region.Description
		if displayName, ok := regionNames[region.Name]; ok {
			description = displayName
		}
		regionIdMap[region.Name] = description

	}
	log.Debugf("regions found: %v", regionIdMap)
	return regionIdMap, nil
}

// GetZones returns the availability zones in a region
func (g *GceInfoer) GetZones(region string) ([]string, error) {
	log.Debugf("getting zones in region %s", region)
	zones := make([]string, 0)
	err := g.computeSvc.Zones.List(g.projectId).Pages(context.TODO(), func(zoneList *compute.ZoneList) error {
		for _, z := range zoneList.Items {
			s := strings.Split(z.Region, "/")
			if s[len(s)-1] == region && z.Name != "" {
				zones = append(zones, z.Name)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	log.Debugf("found zones in region %s", zones)
	return zones, nil
}

// HasShortLivedPriceInfo - Google Cloud has static prices for preemptible instances as well
func (g *GceInfoer) HasShortLivedPriceInfo() bool {
	return false
}

// GetCurrentPrices retrieves all the spot prices in a region
func (g *GceInfoer) GetCurrentPrices(region string) (map[string]productinfo.Price, error) {
	log.Debugf("getting current prices in region %s", region)
	allPrices, err := g.Initialize()
	if err != nil {
		return nil, err
	}
	log.Debugf("found prices in region %s", region)
	return allPrices[region], nil
}

// GetMemoryAttrName returns the provider representation of the memory attribute
func (g *GceInfoer) GetMemoryAttrName() string {
	return memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (g *GceInfoer) GetCpuAttrName() string {
	return cpu
}

// GetNetworkPerformanceMapper returns the network performance mappier implementation for this provider
func (g *GceInfoer) GetNetworkPerformanceMapper() (productinfo.NetworkPerfMapper, error) {
	return newGceNetworkMapper(), nil
}
