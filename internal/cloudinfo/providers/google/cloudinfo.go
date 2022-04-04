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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
)

const svcGke = "gke"

var regionNames = map[string]string{
	"asia-east1":              "Asia Pacific (Taiwan)",
	"asia-east2":              "Asia Pacific (Hong Kong)",
	"asia-northeast1":         "Asia Pacific (Tokyo)",
	"asia-northeast2":         "Asia Pacific (Osaka)",
	"asia-northeast3":         "Asia Pacific (Seoul)",
	"asia-south1":             "Asia Pacific (Mumbai)",
	"asia-south2":             "Asia Pacific (Delhi)",
	"asia-southeast1":         "Asia Pacific (Singapore)",
	"asia-southeast2":         "Asia Pacific (Jakarta)",
	"australia-southeast1":    "Asia Pacific (Sydney)",
	"australia-southeast2":    "Asia Pacific (Melbourne)",
	"europe-north1":           "EU (Finland)",
	"europe-central2":         "EU (Warsaw)",
	"europe-west1":            "EU (Belgium)",
	"europe-west2":            "EU (London)",
	"europe-west3":            "EU (Frankfurt)",
	"europe-west4":            "EU (Netherlands)",
	"europe-west6":            "EU (Zurich)",
	"northamerica-northeast1": "Canada (Montréal)",
	"northamerica-northeast2": "Canada (Toronto)",
	"southamerica-east1":      "South America (São Paulo)",
	"southamerica-west1":      "South America (Santiago)",
	"us-central1":             "US Central (Iowa)",
	"us-east1":                "US East (South Carolina)",
	"us-east4":                "US East (Northern Virginia)",
	"us-west1":                "US West (Oregon)",
	"us-west2":                "US West (Los Angeles)",
	"us-west3":                "US West (Salt Lake City)",
	"us-west4":                "US West (Las Vegas)",
}

// GceInfoer encapsulates the data and operations needed to access external resources
type GceInfoer struct {
	cbSvc        *cloudbilling.APIService
	computeSvc   *compute.Service
	containerSvc *container.Service
	projectId    string
	log          cloudinfo.Logger
}

// NewGoogleInfoer creates a new instance of the Google infoer.
func NewGoogleInfoer(config Config, logger cloudinfo.Logger) (*GceInfoer, error) {
	clientOpts := []option.ClientOption{
		option.WithCredentialsFile(config.CredentialsFile),
		option.WithScopes(compute.ComputeReadonlyScope, container.CloudPlatformScope),
	}

	if config.Credentials != "" {
		decoded, err := base64.StdEncoding.DecodeString(config.Credentials)
		if err != nil {
			return nil, emperror.WrapWith(err, "failed to decode credentials")
		}

		clientOpts = append(clientOpts, option.WithCredentialsJSON(decoded))
	}

	computeSvc, err := compute.NewService(context.Background(), clientOpts...)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create the compute service client")
	}

	billingSvc, err := cloudbilling.NewService(context.Background(), clientOpts...)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create the billing service client")
	}

	containerSvc, err := container.NewService(context.Background(), clientOpts...)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create the container service client")
	}

	project, err := getProject(config)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to determine the project")
	}

	return &GceInfoer{
		cbSvc:        billingSvc,
		computeSvc:   computeSvc,
		containerSvc: containerSvc,
		projectId:    project,
		log:          logger,
	}, nil
}

func getProject(config Config) (string, error) {
	if config.Project != "" {
		return config.Project, nil
	}

	if config.Credentials != "" {
		decoded, err := base64.StdEncoding.DecodeString(config.Credentials)
		if err != nil {
			return "", emperror.Wrap(err, "failed to decode credentials")
		}

		credentials, err := google.CredentialsFromJSON(context.Background(), decoded)
		if err != nil {
			return "", emperror.Wrap(err, "failed to parse credentials")
		}

		if credentials.ProjectID != "" {
			return credentials.ProjectID, nil
		}
	}

	if config.CredentialsFile != "" {
		file, err := ioutil.ReadFile(config.CredentialsFile)
		if err != nil {
			return "", emperror.Wrap(err, "failed to read credentials file")
		}

		credentials, err := google.CredentialsFromJSON(context.Background(), file)
		if err != nil {
			return "", emperror.Wrap(err, "failed to parse credentials")
		}

		if credentials.ProjectID != "" {
			return credentials.ProjectID, nil
		}
	}

	credentials, err := google.FindDefaultCredentials(
		context.Background(),
		compute.ComputeReadonlyScope,
		container.CloudPlatformScope,
	)
	if err != nil {
		return "", emperror.Wrap(err, "failed to find credentials")
	}
	if credentials.ProjectID != "" {
		return credentials.ProjectID, nil
	}

	return "", errors.New("non of the credentials contained the project id")
}

// Initialize downloads and parses the SKU list of the Compute Engine service
func (g *GceInfoer) Initialize() (map[string]map[string]types.Price, error) {
	g.log.Debug("initializing price info")
	allPrices := make(map[string]map[string]types.Price)
	unsupportedInstanceTypes := []string{"n1-ultramem-40", "n1-ultramem-80", "n1-megamem-96", "n1-ultramem-160"}

	zonesInRegions := make(map[string][]string)
	regions, err := g.GetRegions("compute")
	if err != nil {
		return nil, err
	}

	pricePerRegion, err := g.getPrice()
	if err != nil {
		return nil, err
	}
	for r := range regions {
		zones, err := g.GetZones(r)
		if err != nil {
			return nil, err
		}
		zonesInRegions[r] = zones
		err = g.computeSvc.MachineTypes.List(g.projectId, zones[0]).Pages(context.TODO(), func(allMts *compute.MachineTypeList) error {
			for region, price := range pricePerRegion {
				for _, mt := range allMts.Items {
					if !cloudinfo.Contains(unsupportedInstanceTypes, mt.Name) {
						if allPrices[region] == nil {
							allPrices[region] = make(map[string]types.Price)
						}
						prices := allPrices[region][mt.Name]

						if mt.Name == "f1-micro" || mt.Name == "g1-small" {
							prices.OnDemandPrice = price[mt.Name]["OnDemand"]
						} else {
							prices.OnDemandPrice = price[types.CPU]["OnDemand"]*float64(mt.GuestCpus) + price[types.Memory]["OnDemand"]*float64(mt.MemoryMb)/1024
						}
						spotPrice := make(types.SpotPriceInfo)
						for _, z := range zonesInRegions[region] {
							if mt.Name == "f1-micro" || mt.Name == "g1-small" {
								spotPrice[z] = price[mt.Name]["Preemptible"]
								metrics.ReportGoogleSpotPrice(region, z, mt.Name, spotPrice[z])
							} else {
								spotPrice[z] = price[types.CPU]["Preemptible"]*float64(mt.GuestCpus) + price[types.Memory]["Preemptible"]*float64(mt.MemoryMb)/1024
							}

							metrics.ReportGoogleSpotPrice(region, z, mt.Name, spotPrice[z])
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

	g.log.Debug("finished initializing price info")
	return allPrices, nil
}

func (g *GceInfoer) getPrice() (map[string]map[string]map[string]float64, error) {
	svcList, err := g.cbSvc.Services.List().Fields("services/displayName", "services/name").Do()
	if err != nil {
		return nil, err
	}

	var compEngId string
	for _, svc := range svcList.Services {
		if svc.DisplayName == "Compute Engine" {
			compEngId = svc.Name
		}
	}

	price := make(map[string]map[string]map[string]float64)
	err = g.cbSvc.Services.Skus.List(compEngId).Pages(context.Background(), func(response *cloudbilling.ListSkusResponse) error {
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
							price[region][types.Memory] = g.priceFromSku(price, region, types.Memory, sku.Category.UsageType, priceInUsd)
						} else {
							price[region][types.CPU] = g.priceFromSku(price, region, types.CPU, sku.Category.UsageType, priceInUsd)
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

func (g *GceInfoer) priceInUsd(pricingInfos []*cloudbilling.PricingInfo) (float64, error) {
	if len(pricingInfos) != 1 {
		return 0, emperror.With(errors.New("pricing info not parsable"), "numberOfPricingInfos", len(pricingInfos))
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

func (g *GceInfoer) GetVirtualMachines(region string) ([]types.VMInfo, error) {
	logger := log.WithFields(g.log, map[string]interface{}{"region": region})
	logger.Debug("retrieving product information")
	var vmsMap = make(map[string]types.VMInfo)
	var ntwPerf uint

	zones, err := g.GetZones(region)
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
					logger.Debug(emperror.Wrap(err, "failed to get network performance category").Error(),
						map[string]interface{}{"instanceType": mt.Name})
				}
				vmsMap[mt.Name] = types.VMInfo{
					Category:   g.getCategory(mt.Name),
					Type:       mt.Name,
					Cpus:       float64(mt.GuestCpus),
					Mem:        float64(mt.MemoryMb) / 1024,
					NtwPerf:    fmt.Sprintf("%d Gbit/s", ntwPerf),
					NtwPerfCat: ntwPerfCat,
					Zones:      zones,
					Attributes: cloudinfo.Attributes(fmt.Sprint(mt.GuestCpus), fmt.Sprint(float64(mt.MemoryMb)/1024), ntwPerfCat, g.getCategory(mt.Name)),
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var vms []types.VMInfo
	for _, vm := range vmsMap {
		vms = append(vms, vm)
	}
	logger.Debug("found virtual machines", map[string]interface{}{"vms": len(vms)})
	return vms, nil
}

func (g *GceInfoer) getCategory(name string) string {
	switch {
	case strings.Contains(name, "highmem"):
		return types.CategoryMemory
	case strings.Contains(name, "highcpu"):
		return types.CategoryCompute
	default:
		return types.CategoryGeneral
	}
}

// GetProducts retrieves the available virtual machines based on the arguments provided
// Queries the Google Cloud Compute API's machine type list endpoint and CloudBilling's sku list endpoint
func (g *GceInfoer) GetProducts(vms []types.VMInfo, service, regionId string) ([]types.VMInfo, error) {
	var vmList = vms
	if len(vmList) == 0 {
		var err error
		vmList, err = g.GetVirtualMachines(regionId)
		if err != nil {
			g.log.Warn("could not get machine types for region", map[string]interface{}{"regionId": regionId})
			return nil, emperror.Wrap(err, "failed to get products")
		}
	}

	switch service {
	case svcGke, "compute":
		return vmList, nil
	default:
		return nil, errors.Wrap(errors.New(service), "invalid service")
	}
}

// GetRegions returns a map with available regions transforms the api representation into a "plain" map
func (g *GceInfoer) GetRegions(service string) (map[string]string, error) {
	logger := log.WithFields(g.log, map[string]interface{}{"service": service})
	logger.Debug("getting regions")

	regionList, err := g.computeSvc.Regions.List(g.projectId).Do()
	if err != nil {
		return nil, err
	}

	regionIdMap := make(map[string]string)
	for _, region := range regionList.Items {
		if displayName, ok := regionNames[region.Name]; ok {
			regionIdMap[region.Name] = displayName
		}
	}

	logger.Debug("found regions", map[string]interface{}{"numberOfRegions": len(regionIdMap)})
	return regionIdMap, nil
}

// GetZones returns the availability zones in a region
func (g *GceInfoer) GetZones(region string) ([]string, error) {
	logger := log.WithFields(g.log, map[string]interface{}{"region": region})
	logger.Debug("getting zones")

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

	logger.Debug("found zones", map[string]interface{}{"numberOfZones": len(zones)})
	return zones, nil
}

// HasShortLivedPriceInfo - Google Cloud has static prices for preemptible instances as well
func (g *GceInfoer) HasShortLivedPriceInfo() bool {
	return false
}

// GetCurrentPrices retrieves all the spot prices in a region
func (g *GceInfoer) GetCurrentPrices(region string) (map[string]types.Price, error) {
	return nil, errors.New("google prices cannot be queried on the fly")
}

// HasImages - Google doesn't support images
func (g *GceInfoer) HasImages() bool {
	return false
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (g *GceInfoer) GetServiceImages(service, region string) ([]types.Image, error) {
	return nil, errors.New("GetServiceImages - not yet implemented")
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (g *GceInfoer) GetServiceProducts(region, service string) ([]types.ProductDetails, error) {
	return nil, errors.New("GetServiceProducts - not yet implemented")
}

// GetVersions retrieves the kubernetes versions supported by the given service in the given region
func (g *GceInfoer) GetVersions(service, region string) ([]types.LocationVersion, error) {
	switch service {
	case svcGke:
		var zoneVersions []types.LocationVersion
		zones, err := g.GetZones(region)
		if err != nil {
			return nil, err
		}

		for _, zone := range zones {
			var versions []string

			serverConf, err := g.containerSvc.Projects.Zones.GetServerconfig(g.projectId, zone).Context(context.Background()).Do()
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
			zoneVersions = append(zoneVersions, types.NewLocationVersion(zone, versions, serverConf.DefaultClusterVersion))
		}

		return zoneVersions, nil
	default:
		return []types.LocationVersion{}, nil
	}
}
