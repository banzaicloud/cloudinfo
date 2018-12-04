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
	"github.com/prometheus/client_golang/prometheus"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/banzaicloud/cloudinfo/pkg/logger"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"golang.org/x/oauth2/google"
	billing "google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/googleapi/transport"
)

// SpotPriceGauge collects metrics for the prometheus
var SpotPriceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "cloudinfo",
	Name:      "google_spot_price",
	Help:      "spot price for each instance type",
},
	[]string{"region", "zone", "instanceType"},
)

var regionNames = map[string]string{
	"asia-east1":              "Asia Pacific (Taiwan)",
	"asia-east2":              "Asia Pacific (Hong Kong)",
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
	containerSvc       *container.Service
	projectId          string
	cpuRegex           *regexp.Regexp
	resourceGroupRegex *regexp.Regexp
}

// NewGceInfoer creates a new instance of the infoer
func NewGceInfoer(appCredentials, apiKey string) (*GceInfoer, error) {
	if appCredentials == "" {
		return nil, fmt.Errorf("environment variable GOOGLE_APPLICATION_CREDENTIALS is not set")
	}
	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", appCredentials)

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
	containerSvc, err := container.New(client)
	if err != nil {
		return nil, err
	}

	cpuReg, _ := regexp.Compile(`\d+ VCPU`)
	rgReg, _ := regexp.Compile(`^[a-z]+\d+`)

	return &GceInfoer{
		cbSvc:              billingSvc,
		computeSvc:         computeSvc,
		containerSvc:       containerSvc,
		projectId:          defaultCredential.ProjectID,
		cpuRegex:           cpuReg,
		resourceGroupRegex: rgReg,
	}, nil
}

// Initialize downloads and parses the SKU list of the Compute Engine service
func (g *GceInfoer) Initialize(ctx context.Context) (map[string]map[string]cloudinfo.Price, error) {
	log := logger.Extract(ctx)
	log.Debug("initializing price info")
	allPrices := make(map[string]map[string]cloudinfo.Price)
	unsupportedInstanceTypes := []string{"n1-ultramem-40", "n1-ultramem-80", "n1-megamem-96", "n1-ultramem-160"}

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

	pricePerRegion, err := g.getPrice(compEngId)
	if err != nil {
		return nil, err
	}
	for r := range regions {
		zones, err := g.GetZones(ctx, r)
		if err != nil {
			return nil, err
		}
		zonesInRegions[r] = zones
		err = g.computeSvc.MachineTypes.List(g.projectId, zones[0]).Pages(context.TODO(), func(allMts *compute.MachineTypeList) error {
			for region, price := range pricePerRegion {
				for _, mt := range allMts.Items {
					if !cloudinfo.Contains(unsupportedInstanceTypes, mt.Name) {
						if allPrices[region] == nil {
							allPrices[region] = make(map[string]cloudinfo.Price)
						}
						prices := allPrices[region][mt.Name]

						if mt.Name == "f1-micro" || mt.Name == "g1-small" {
							prices.OnDemandPrice = price[mt.Name]["OnDemand"]
						} else {
							prices.OnDemandPrice = price[cloudinfo.Cpu]["OnDemand"]*float64(mt.GuestCpus) + price[cloudinfo.Memory]["OnDemand"]*float64(mt.MemoryMb)/1024
						}
						spotPrice := make(cloudinfo.SpotPriceInfo)
						for _, z := range zonesInRegions[region] {
							if mt.Name == "f1-micro" || mt.Name == "g1-small" {
								spotPrice[z] = price[mt.Name]["Preemptible"]
								SpotPriceGauge.WithLabelValues(region, z, mt.Name).Set(spotPrice[z])
							} else {
								spotPrice[z] = price[cloudinfo.Cpu]["Preemptible"]*float64(mt.GuestCpus) + price[cloudinfo.Memory]["Preemptible"]*float64(mt.MemoryMb)/1024
								SpotPriceGauge.WithLabelValues(region, z, mt.Name).Set(spotPrice[z])
							}
						}
						prices.SpotPrice = spotPrice

						allPrices[region][mt.Name] = prices
					}
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	log.Debug("finished initializing price info")
	return allPrices, nil
}

func (g *GceInfoer) getPrice(parent string) (map[string]map[string]map[string]float64, error) {
	price := make(map[string]map[string]map[string]float64)
	err := g.cbSvc.Services.Skus.List(parent).Pages(context.Background(), func(response *billing.ListSkusResponse) error {
		for _, sku := range response.Skus {
			if sku.Category.ResourceGroup == "G1Small" || sku.Category.ResourceGroup == "F1Micro" {
				priceInUsd, err := g.priceInUsd(sku.PricingInfo)
				if err != nil {
					return err
				}

				for _, region := range sku.ServiceRegions {
					if price[region] == nil {
						price[region] = make(map[string]map[string]float64)
					}
					if sku.Category.ResourceGroup == "G1Small" {
						price[region]["g1-small"] = g.priceFromSku(price, region, "g1-small", sku.Category.UsageType, priceInUsd)
					} else {
						price[region]["f1-micro"] = g.priceFromSku(price, region, "f1-micro", sku.Category.UsageType, priceInUsd)
					}
				}
			}
			if sku.Category.ResourceGroup == "N1Standard" {
				if !strings.Contains(sku.Description, "Upgrade Premium") {
					priceInUsd, err := g.priceInUsd(sku.PricingInfo)
					if err != nil {
						return err
					}

					for _, region := range sku.ServiceRegions {
						if price[region] == nil {
							price[region] = make(map[string]map[string]float64)
						}
						if strings.Contains(sku.Description, "Instance Ram") {
							price[region][cloudinfo.Memory] = g.priceFromSku(price, region, cloudinfo.Memory, sku.Category.UsageType, priceInUsd)

						} else {
							price[region][cloudinfo.Cpu] = g.priceFromSku(price, region, cloudinfo.Cpu, sku.Category.UsageType, priceInUsd)
						}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return price, nil
}

func (g *GceInfoer) priceInUsd(pricingInfos []*billing.PricingInfo) (float64, error) {
	if len(pricingInfos) != 1 {
		return 0, fmt.Errorf("pricing info not parsable, %d pricing info entries are returned", len(pricingInfos))
	}
	pricingInfo := pricingInfos[0]
	var priceInUsd float64
	for _, tr := range pricingInfo.PricingExpression.TieredRates {
		priceInUsd += float64(tr.UnitPrice.Units) + float64(tr.UnitPrice.Nanos)*1e-9
	}
	return priceInUsd, nil
}

func (g *GceInfoer) priceFromSku(price map[string]map[string]map[string]float64, region, device, priceType string, priceInUsd float64) map[string]float64 {
	pr := price[region][device]
	if pr == nil {
		pr = make(map[string]float64)
	}
	pr[priceType] = priceInUsd

	return pr
}

// GetAttributeValues gets the AttributeValues for the given attribute name
// Queries the Google Cloud Compute API's machine type list endpoint
func (g *GceInfoer) GetAttributeValues(ctx context.Context, service, attribute string) (cloudinfo.AttrValues, error) {
	log := logger.Extract(ctx)
	log.Debugf("getting %s values", attribute)

	values := make(cloudinfo.AttrValues, 0)
	valueSet := make(map[cloudinfo.AttrValue]interface{})

	err := g.computeSvc.MachineTypes.AggregatedList(g.projectId).Pages(context.TODO(), func(allMts *compute.MachineTypeAggregatedList) error {
		for _, scope := range allMts.Items {
			for _, mt := range scope.MachineTypes {
				switch attribute {
				case cloudinfo.Cpu:
					valueSet[cloudinfo.AttrValue{
						Value:    float64(mt.GuestCpus),
						StrValue: fmt.Sprintf("%v", mt.GuestCpus),
					}] = ""
				case cloudinfo.Memory:
					valueSet[cloudinfo.AttrValue{
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
func (g *GceInfoer) GetProducts(ctx context.Context, service, regionId string) ([]cloudinfo.VmInfo, error) {
	log := logger.Extract(ctx)
	log.Debugf("getting product info")
	var vmsMap = make(map[string]cloudinfo.VmInfo)
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
				ntwMapper := newGceNetworkMapper()
				ntwPerfCat, err := ntwMapper.MapNetworkPerf(fmt.Sprint(ntwPerf, " Gbit/s"))
				if err != nil {
					log.WithError(err).Debug("could not get network performance category")
				}
				vmsMap[mt.Name] = cloudinfo.VmInfo{
					Type:       mt.Name,
					Cpus:       float64(mt.GuestCpus),
					Mem:        float64(mt.MemoryMb) / 1024,
					NtwPerf:    fmt.Sprintf("%d Gbit/s", ntwPerf),
					NtwPerfCat: ntwPerfCat,
					Zones:      zones,
					Attributes: cloudinfo.Attributes(fmt.Sprint(mt.GuestCpus), fmt.Sprint(float64(mt.MemoryMb)/1024), ntwPerfCat),
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var vms []cloudinfo.VmInfo
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
func (g *GceInfoer) GetCurrentPrices(ctx context.Context, region string) (map[string]cloudinfo.Price, error) {
	return nil, fmt.Errorf("google prices cannot be queried on the fly")
}

// GetMemoryAttrName returns the provider representation of the memory attribute
func (g *GceInfoer) GetMemoryAttrName() string {
	return cloudinfo.Memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (g *GceInfoer) GetCpuAttrName() string {
	return cloudinfo.Cpu
}

// GetServices returns the available services on the  provider
func (g *GceInfoer) GetServices() ([]cloudinfo.ServiceDescriber, error) {
	services := []cloudinfo.ServiceDescriber{
		cloudinfo.NewService("compute"),
		cloudinfo.NewService("gke")}
	return services, nil
}

// GetService returns the given service details on the provider
func (g *GceInfoer) GetService(ctx context.Context, service string) (cloudinfo.ServiceDescriber, error) {
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

// HasImages - Google doesn't support images
func (g *GceInfoer) HasImages() bool {
	return false
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (g *GceInfoer) GetServiceImages(region, service string) ([]cloudinfo.ImageDescriber, error) {
	return nil, fmt.Errorf("GetServiceImages - not yet implemented")
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (g *GceInfoer) GetServiceProducts(region, service string) ([]cloudinfo.ProductDetails, error) {
	return nil, fmt.Errorf("GetServiceProducts - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (g *GceInfoer) GetServiceAttributes(region, service, attribute string) (cloudinfo.AttrValues, error) {
	return nil, fmt.Errorf("GetServiceAttributes - not yet implemented")
}

// GetVersions retrieves the kubernetes versions supported by the given service in the given region
func (g *GceInfoer) GetVersions(ctx context.Context, service, region string) ([]string, error) {
	switch service {
	case "gke":
		var versions []string
		zones, err := g.GetZones(ctx, region)
		if err != nil {
			return nil, err
		}
		serverConf, err := g.containerSvc.Projects.Zones.GetServerconfig(g.projectId, zones[0]).Context(context.Background()).Do()
		if err != nil {
			return nil, err
		}
		for _, masterVersion := range serverConf.ValidMasterVersions {
			for _, nodeVersion := range serverConf.ValidNodeVersions {
				if masterVersion == nodeVersion {
					versions = append(versions, masterVersion)
					break
				}
			}
		}
		return versions, nil
	default:
		return []string{}, nil
	}
}
