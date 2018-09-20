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

package google

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/banzaicloud/productinfo/logger"

	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"golang.org/x/oauth2/google"
	billing "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi/transport"
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
	"us-west2":                "US West (Los Angeles)",
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
func (g *GceInfoer) Initialize(ctx context.Context) (map[string]map[string]productinfo.Price, error) {
	log := logger.Extract(ctx)
	log.Debug("initializing price info")
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

	log.Debugf("google compute engine service id: %s", compEngId)

	zonesInRegions := make(map[string][]string)
	regions, err := g.GetRegions(ctx, "compute")
	if err != nil {
		return nil, err
	}
	for r := range regions {
		zones, err := g.GetZones(ctx, r)
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
						log.WithField("region", region).Debugf("price info added: [machinetype=%s, price=%v]", instanceType, price)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	log.Debug("finished initializing price info")
	return allPrices, nil
}

// GetAttributeValues gets the AttributeValues for the given attribute name
// Queries the Google Cloud Compute API's machine type list endpoint
func (g *GceInfoer) GetAttributeValues(ctx context.Context, attribute string) (productinfo.AttrValues, error) {
	log := logger.Extract(ctx)
	log.Debugf("getting %s values", attribute)

	values := make(productinfo.AttrValues, 0)
	valueSet := make(map[productinfo.AttrValue]interface{})

	err := g.computeSvc.MachineTypes.AggregatedList(g.projectId).Pages(context.TODO(), func(allMts *compute.MachineTypeAggregatedList) error {
		for _, scope := range allMts.Items {
			for _, mt := range scope.MachineTypes {
				switch attribute {
				case productinfo.Cpu:
					valueSet[productinfo.AttrValue{
						Value:    float64(mt.GuestCpus),
						StrValue: fmt.Sprintf("%v", mt.GuestCpus),
					}] = ""
				case productinfo.Memory:
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
func (g *GceInfoer) GetProducts(ctx context.Context, service, regionId string) ([]productinfo.VmInfo, error) {
	log := logger.Extract(ctx)
	log.Debugf("getting product info")
	var vmsMap = make(map[string]productinfo.VmInfo)
	var ntwPerf uint
	zones, err := g.GetZones(ctx, regionId)
	if err != nil {
		return nil, err
	}
	err = g.computeSvc.MachineTypes.List(g.projectId, zones[0]).Pages(context.TODO(), func(allMts *compute.MachineTypeList) error {
		for _, mt := range allMts.Items {
			if _, ok := vmsMap[mt.Name]; !ok {
				switch {
				case mt.GuestCpus < 1:
					// minimum 1 Gbps network performance for each virtual machine
					ntwPerf = 1
				case mt.GuestCpus > 8:
					// theoretical maximum of 16 Gbps for each virtual machine
					ntwPerf = 16
				default:
					// each vCPU has a 2 Gbps egress cap for peak performance
					ntwPerf = uint(mt.GuestCpus * 2)
				}
				vmsMap[mt.Name] = productinfo.VmInfo{
					Type:    mt.Name,
					Cpus:    float64(mt.GuestCpus),
					Mem:     float64(mt.MemoryMb) / 1024,
					NtwPerf: fmt.Sprintf("%d Gbit/s", ntwPerf),
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var vms []productinfo.VmInfo
	for _, vm := range vmsMap {
		vms = append(vms, vm)
	}
	log.Debugf("found vms: %#v", vms)
	return vms, nil
}

// GetRegions returns a map with available regions transforms the api representation into a "plain" map
func (g *GceInfoer) GetRegions(ctx context.Context, service string) (map[string]string, error) {
	log := logger.Extract(ctx)
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
func (g *GceInfoer) GetZones(ctx context.Context, region string) ([]string, error) {
	log := logger.Extract(ctx)
	log.Debug("getting zones")
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
	log.Debugf("found zones %s", zones)
	return zones, nil
}

// HasShortLivedPriceInfo - Google Cloud has static prices for preemptible instances as well
func (g *GceInfoer) HasShortLivedPriceInfo() bool {
	return false
}

// GetCurrentPrices retrieves all the spot prices in a region
func (g *GceInfoer) GetCurrentPrices(ctx context.Context, region string) (map[string]productinfo.Price, error) {
	return nil, fmt.Errorf("google prices cannot be queried on the fly")
}

// GetMemoryAttrName returns the provider representation of the memory attribute
func (g *GceInfoer) GetMemoryAttrName() string {
	return productinfo.Memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (g *GceInfoer) GetCpuAttrName() string {
	return productinfo.Cpu
}

// GetNetworkPerformanceMapper returns the network performance mapper implementation for this provider
func (g *GceInfoer) GetNetworkPerformanceMapper() (productinfo.NetworkPerfMapper, error) {
	return newGceNetworkMapper(), nil
}

// GetServices returns the available services on the  provider
func (g *GceInfoer) GetServices() ([]productinfo.ServiceDescriber, error) {
	services := []productinfo.ServiceDescriber{
		productinfo.NewService("compute"),
		productinfo.NewService("gke")}
	return services, nil
}

// GetService returns the given service details on the provider
func (g *GceInfoer) GetService(ctx context.Context, service string) (productinfo.ServiceDescriber, error) {
	svcs, err := g.GetServices()
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
func (g *GceInfoer) GetServiceImages(region, service string) ([]productinfo.ImageDescriber, error) {
	return nil, fmt.Errorf("GetServiceImages - not yet implemented")
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (g *GceInfoer) GetServiceProducts(region, service string) ([]productinfo.ProductDetails, error) {
	return nil, fmt.Errorf("GetServiceProducts - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (g *GceInfoer) GetServiceAttributes(region, service, attribute string) (productinfo.AttrValues, error) {
	return nil, fmt.Errorf("GetServiceAttributes - not yet implemented")
}
