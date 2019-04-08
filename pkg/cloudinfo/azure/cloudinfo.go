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

package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-09-01/skus"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2018-03-31/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/preview/commerce/mgmt/2015-06-01-preview/commerce"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
)

const svcAks = "aks"

var (
	regionCodeMappings = map[string]string{
		"ap": "asia",
		"au": "australia",
		"br": "brazil",
		"ca": "canada",
		"eu": "europe",
		"fr": "france",
		"in": "india",
		"ja": "japan",
		"kr": "korea",
		"uk": "uk",
		"us": "us",
	}

	//mtBasic, _     = regexp.Compile("^BASIC.A\\d+[_Promo]*$")
	//mtStandardA, _ = regexp.Compile("^A\\d+[_Promo]*$")
	mtStandardB, _ = regexp.Compile(`^Standard_B\d+m?[_v\d]*[_Promo]*$`)
	mtStandardD, _ = regexp.Compile(`^Standard_D\d[_v\d]*[_Promo]*$`)
	mtStandardE, _ = regexp.Compile(`^Standard_E\d+i?[_v\d]*[_Promo]*$`)
	mtStandardF, _ = regexp.Compile(`^Standard_F\d+[_v\d]*[_Promo]*$`)
	mtStandardG, _ = regexp.Compile(`^Standard_G\d+[_v\d]*[_Promo]*$`)
	mtStandardL, _ = regexp.Compile(`^Standard_L\d+[_v\d]*[_Promo]*$`)
	mtStandardM, _ = regexp.Compile(`^Standard_M\d+[m|t|l]*s[_v\d]*[_Promo]*$`)
	mtStandardN, _ = regexp.Compile(`^Standard_N[C|D|V]\d+r?[_v\d]*[_Promo]*$`)
)

type authentication struct {
	SubscriptionID string `json:"subscriptionId,omitempty"`
}

// AzureInfoer encapsulates the data and operations needed to access external Azure resources
type AzureInfoer struct {
	subscriptionId      string
	subscriptionsClient LocationRetriever
	vmSizesClient       VmSizesRetriever
	rateCardClient      PriceRetriever
	skusClient          CategoryRetriever
	providersClient     ProviderSource
	containerSvcClient  VersionRetriever
	log                 logur.Logger
}

// VmSizesRetriever list of operations for retrieving virtual machines information
type VmSizesRetriever interface {
	List(ctx context.Context, location string) (result compute.VirtualMachineSizeListResult, err error)
}

// LocationRetriever collects regions
type LocationRetriever interface {
	ListLocations(ctx context.Context, subscriptionID string) (result subscriptions.LocationListResult, err error)
}

// ProviderSource returns the available location per provider
type ProviderSource interface {
	Get(ctx context.Context, resourceProviderNamespace string, expand string) (result resources.Provider, err error)
}

// PriceRetriever collects prices
type PriceRetriever interface {
	Get(ctx context.Context, filter string) (result commerce.ResourceRateCardInfo, err error)
}

// VersionRetriever collects k8s versions
type VersionRetriever interface {
	ListOrchestrators(ctx context.Context, location string, resourceType string) (result containerservice.OrchestratorVersionProfileListResult, err error)
}

// CategoryRetriever collects virtual machines family
type CategoryRetriever interface {
	List(ctx context.Context) (result skus.ResourceSkusResultPage, err error)
}

// newInfoer creates a new instance of the Azure infoer
func newInfoer(authLocation string, log logur.Logger) (*AzureInfoer, error) {
	err := os.Setenv("AZURE_AUTH_LOCATION", authLocation)
	if err != nil {
		return nil, err
	}

	authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadFile(authLocation)
	if err != nil {
		return nil, err
	}
	a := authentication{}
	err = json.Unmarshal(contents, &a)
	if err != nil {
		return nil, err
	}

	sClient := subscriptions.NewClient()
	sClient.Authorizer = authorizer

	vmClient := compute.NewVirtualMachineSizesClient(a.SubscriptionID)
	vmClient.Authorizer = authorizer

	rcClient := commerce.NewRateCardClient(a.SubscriptionID)
	rcClient.Authorizer = authorizer

	skusClient := skus.NewResourceSkusClient(a.SubscriptionID)
	skusClient.Authorizer = authorizer

	providersClient := resources.NewProvidersClient(a.SubscriptionID)
	providersClient.Authorizer = authorizer

	containerServiceClient := containerservice.NewContainerServicesClient(a.SubscriptionID)
	containerServiceClient.Authorizer = authorizer

	return &AzureInfoer{
		subscriptionId:      a.SubscriptionID,
		subscriptionsClient: sClient,
		vmSizesClient:       vmClient,
		skusClient:          skusClient,
		rateCardClient:      rcClient,
		providersClient:     providersClient,
		containerSvcClient:  &containerServiceClient,
		log:                 log,
	}, nil
}

func NewAzureInfoer(cfg Config, log logur.Logger) (*AzureInfoer, error) {
	return newInfoer(cfg.AuthLocation, log)
}

type regionParts []string

func (r regionParts) String() string {
	var result string
	for _, p := range r {
		result += p
	}
	return result
}

func (a *AzureInfoer) toRegionID(meterRegion string, regions map[string]string) (string, error) {
	var rp regionParts = strings.Split(strings.ToLower(meterRegion), " ")
	regionCode := regionCodeMappings[rp[0]]
	lastPart := rp[len(rp)-1]
	var regionIds []string
	if _, err := strconv.Atoi(lastPart); err == nil {
		regionIds = []string{
			fmt.Sprintf("%s%s%s", regionCode, rp[1:len(rp)-1], lastPart),
			fmt.Sprintf("%s%s%s", rp[1:len(rp)-1], regionCode, lastPart),
		}
	} else {
		regionIds = []string{
			fmt.Sprintf("%s%s", regionCode, rp[1:]),
			fmt.Sprintf("%s%s", rp[1:], regionCode),
		}
	}
	for _, regionID := range regionIds {
		if a.checkRegionID(regionID, regions) {
			return regionID, nil
		}
	}
	return "", errors.Wrap(errors.New(meterRegion), "couldn't find region")
}

func (a *AzureInfoer) checkRegionID(regionID string, regions map[string]string) bool {
	for region := range regions {
		if regionID == region {
			return true
		}
	}
	return false
}

// Initialize downloads and parses the Rate Card API's meter list on Azure
func (a *AzureInfoer) Initialize() (map[string]map[string]cloudinfo.Price, error) {
	a.log.Debug("initializing price info")
	allPrices := make(map[string]map[string]cloudinfo.Price)

	regions, err := a.GetRegions("compute")
	if err != nil {
		return nil, err
	}

	rateCardFilter := "OfferDurableId eq 'MS-AZR-0003p' and Currency eq 'USD' and Locale eq 'en-US' and RegionInfo eq 'US'"
	result, err := a.rateCardClient.Get(context.TODO(), rateCardFilter)
	if err != nil {
		return nil, err
	}

	var missingRegions []string
	for _, v := range *result.Meters {
		if *v.MeterCategory == "Virtual Machines" && len(*v.MeterTags) == 0 && *v.MeterRegion != "" {
			if !strings.Contains(*v.MeterSubCategory, "Windows") {
				region, err := a.toRegionID(*v.MeterRegion, regions)
				if err != nil {
					missingRegions = appendIfMissing(missingRegions, *v.MeterRegion)
					continue
				}

				instanceTypes := a.machineType(*v.MeterName, *v.MeterSubCategory)

				var priceInUsd float64

				if len(v.MeterRates) < 1 {
					a.log.Debug("missing rate info", map[string]interface{}{"MeterSubCategory": *v.MeterSubCategory, "region": region})
					continue
				}
				for _, rate := range v.MeterRates {
					priceInUsd += *rate
				}
				if allPrices[region] == nil {
					allPrices[region] = make(map[string]cloudinfo.Price)
				}
				for _, instanceType := range instanceTypes {
					price := allPrices[region][instanceType]
					if !strings.Contains(*v.MeterName, "Low Priority") {
						price.OnDemandPrice = priceInUsd
					} else {
						spotPrice := make(cloudinfo.SpotPriceInfo)
						spotPrice[region] = priceInUsd
						price.SpotPrice = spotPrice
						metrics.ReportAzureSpotPrice(region, instanceType, priceInUsd)
					}

					allPrices[region][instanceType] = price

					mts := a.getMachineTypeVariants(instanceType)
					for _, mt := range mts {
						allPrices[region][mt] = price
					}
				}
			}
		}
	}
	a.log.Debug("couldn't find regions", map[string]interface{}{"missingRegions": missingRegions})

	a.log.Debug("finished initializing price info")
	return allPrices, nil
}

func (a *AzureInfoer) machineType(meterName string, subCategory string) []string {
	var instanceTypes []string
	name := strings.TrimSuffix(meterName, " Low Priority")
	instanceType := strings.Split(name, "/")
	for _, it := range instanceType {
		instanceTypes = append(instanceTypes, strings.Replace(it, " ", "_", 1))
	}

	// not available now
	if strings.Contains(subCategory, "Promo") {
		instanceTypes[0] = cloudinfo.CreateString(instanceTypes[0], "_Promo")
	}
	instanceTypes = a.transformMachineType(subCategory, instanceTypes)
	if strings.Contains(name, "Expired") {
		instanceTypes = []string{}
	}
	return instanceTypes
}

func (a *AzureInfoer) transformMachineType(subCategory string, mt []string) []string {
	switch {
	case strings.Contains(subCategory, "Basic"):
		return []string{cloudinfo.CreateString("Basic_", mt[0])}
	case len(mt) == 2:
		return []string{cloudinfo.CreateString("Standard_", mt[0]), cloudinfo.CreateString("Standard_", mt[1])}
	default:
		return []string{cloudinfo.CreateString("Standard_", mt[0])}
	}
}

func (a *AzureInfoer) getMachineTypeVariants(mt string) []string {
	switch {
	case mtStandardB.MatchString(mt):
		return []string{cloudinfo.CreateString(mt, "s")}
	case mtStandardD.MatchString(mt):
		var result []string
		result = append(result, a.addSuffix(mt, "s")[0])
		dsType := strings.Replace(mt, "Standard_D", "Standard_DS", -1)
		result = append(result, dsType)
		result = append(result, a.addSuffix(dsType, "-1", "-2", "-4", "-8")...)
		return result
	case mtStandardE.MatchString(mt):
		return a.addSuffix(mt, "s", "-2s", "-4s", "-8s", "-16s", "-32s")
	case mtStandardF.MatchString(mt):
		return a.addSuffix(mt, "s")
	case mtStandardG.MatchString(mt):
		var result []string
		gsType := strings.Replace(mt, "Standard_G", "Standard_GS", -1)
		result = append(result, gsType)
		return append(result, a.addSuffix(gsType, "-4", "-8", "-16")...)
	case mtStandardL.MatchString(mt):
		return a.addSuffix(mt, "s")
	case mtStandardM.MatchString(mt) && strings.HasSuffix(mt, "ms"):
		base := strings.TrimSuffix(mt, "ms")
		return a.addSuffix(base, "-2ms", "-4ms", "-8ms", "-16ms", "-32ms", "-64ms")
	case mtStandardM.MatchString(mt) && (strings.HasSuffix(mt, "ls") || strings.HasSuffix(mt, "ts")):
		return []string{}
	case mtStandardM.MatchString(mt) && strings.HasSuffix(mt, "s"):
		base := strings.TrimSuffix(mt, "s")
		return a.addSuffix(base, "", "m")
	case mtStandardN.MatchString(mt):
		return a.addSuffix(mt, "s")
	}

	return []string{}
}

func (a *AzureInfoer) addSuffix(mt string, suffixes ...string) []string {
	result := make([]string, len(suffixes))
	var suffix string
	parts := strings.Split(mt, "_")
	if len(parts) > 2 {
		for _, p := range parts[2:] {
			suffix = cloudinfo.CreateString(suffix, "_", p)
		}
	}
	for i, s := range suffixes {
		result[i] = cloudinfo.CreateString(parts[0], "_", parts[1], s, suffix)
	}
	return result
}

func (a *AzureInfoer) getCategory(vms []cloudinfo.VmInfo, log logur.Logger) ([]cloudinfo.VmInfo, error) {
	skusResultPage, err := a.skusClient.List(context.Background())
	if err != nil {
		return nil, err
	}

	var virtualMachines []cloudinfo.VmInfo
	for _, vm := range vms {
		for _, sku := range skusResultPage.Values() {
			if *sku.ResourceType == "virtualMachines" {
				if *sku.Name == vm.Type {

					category, err := a.mapCategory(*sku.Family)
					if err != nil {
						log.Debug(emperror.Wrap(err, "failed to get virtual machine category").Error(),
							map[string]interface{}{"instanceType": vm.Type})
					}

					virtualMachines = append(virtualMachines, cloudinfo.VmInfo{
						Category:   category,
						Type:       vm.Type,
						Mem:        vm.Mem,
						Cpus:       vm.Cpus,
						NtwPerf:    "1 Gbit/s",
						NtwPerfCat: cloudinfo.NtwLow,
						Attributes: cloudinfo.Attributes(fmt.Sprint(vm.Mem), fmt.Sprint(vm.Cpus), "unknown", category),
					})
					break
				}
			}
		}
	}
	return virtualMachines, nil
}

func (a *AzureInfoer) GetVirtualMachines(region string) ([]cloudinfo.VmInfo, error) {
	logger := log.WithFields(a.log, map[string]interface{}{"region": region})
	logger.Debug("getting product info")
	var vms []cloudinfo.VmInfo
	vmSizes, err := a.vmSizesClient.List(context.TODO(), region)
	if err != nil {
		return nil, err
	}
	for _, v := range *vmSizes.Value {
		vms = append(vms, cloudinfo.VmInfo{
			Type:       *v.Name,
			Cpus:       float64(*v.NumberOfCores),
			Mem:        float64(*v.MemoryInMB) / 1024,
			NtwPerf:    "1 Gbit/s",
			NtwPerfCat: cloudinfo.NtwLow,
		})
	}

	virtualMachines, err := a.getCategory(vms, logger)
	if err != nil {
		return nil, err
	}

	logger.Debug("found virtual machines", map[string]interface{}{"numberOfVms": len(vms)})
	return virtualMachines, nil
}

// GetProducts retrieves the available virtual machines based on the arguments provided
func (a *AzureInfoer) GetProducts(vms []cloudinfo.VmInfo, service, regionId string) ([]cloudinfo.VmInfo, error) {
	switch service {
	case svcAks:
		var virtualMachines []cloudinfo.VmInfo
		possibleVmTypes := containerservice.PossibleVMSizeTypesValues()
		for _, vm := range possibleVmTypes {
			for _, virtualMachine := range vms {
				if string(vm) == virtualMachine.Type {
					virtualMachines = append(virtualMachines, virtualMachine)
					break
				}
			}
		}
		return virtualMachines, nil
	default:
		return nil, errors.Wrap(errors.New(service), "invalid service")
	}
}

// GetZones returns the availability zones in a region
func (a *AzureInfoer) GetZones(region string) ([]string, error) {
	return []string{region}, nil
}

// GetRegions returns a map with available regions transforms the api representation into a "plain" map
func (a *AzureInfoer) GetRegions(service string) (map[string]string, error) {
	logger := log.WithFields(a.log, map[string]interface{}{"service": service})
	logger.Debug("getting locations")

	allLocations := make(map[string]string)
	supLocations := make(map[string]string)

	// retrieve all locations for the subscription id (some of them may not be supported by the required provider)
	if locations, err := a.subscriptionsClient.ListLocations(context.TODO(), a.subscriptionId); err == nil {
		// fill up the map: DisplayName - > Name
		for _, loc := range *locations.Value {
			allLocations[*loc.DisplayName] = *loc.Name
		}
	} else {
		logger.Error("error while retrieving azure locations")
		return nil, err
	}

	// identify supported locations for the namespace and resource type
	const (
		providerNamespaceForCompute = "Microsoft.Compute"
		resourceTypeForCompute      = "locations/vmSizes"
		providerNamespaceForAks     = "Microsoft.ContainerService"
		resourceTypeForAks          = "managedClusters"
	)

	switch service {
	case "aks":
		if providers, err := a.providersClient.Get(context.TODO(), providerNamespaceForAks, ""); err == nil {
			for _, pr := range *providers.ResourceTypes {
				if *pr.ResourceType == resourceTypeForAks {
					for _, displName := range *pr.Locations {
						if loc, ok := allLocations[displName]; ok {
							supLocations[loc] = displName
						} else {
							logger.Debug("unsupported location", map[string]interface{}{"name": loc, "displayname": displName})
						}
					}
					break
				}
			}
		} else {
			logger.Error("failed to retrieve supported locations", map[string]interface{}{"resource": resourceTypeForAks})
			return nil, err
		}

		logger.Debug("found supported locations", map[string]interface{}{"numberOfLocations": len(supLocations)})
		return supLocations, nil
	default:
		if providers, err := a.providersClient.Get(context.TODO(), providerNamespaceForCompute, ""); err == nil {
			for _, pr := range *providers.ResourceTypes {
				if *pr.ResourceType == resourceTypeForCompute {
					for _, displName := range *pr.Locations {
						if loc, ok := allLocations[displName]; ok {
							supLocations[loc] = displName
						} else {
							logger.Debug("unsupported location", map[string]interface{}{"name": loc, "displayname": displName})
						}
					}
					break
				}
			}
		} else {
			logger.Error("failed to retrieve supported locations", map[string]interface{}{"resource": resourceTypeForCompute})
			return nil, err
		}

		logger.Debug("found supported locations", map[string]interface{}{"numberOfLocations": len(supLocations)})
		return supLocations, nil
	}
}

// HasShortLivedPriceInfo - Azure doesn't have frequently changing prices
func (a *AzureInfoer) HasShortLivedPriceInfo() bool {
	return false
}

// GetCurrentPrices retrieves all the price info in a region
func (a *AzureInfoer) GetCurrentPrices(region string) (map[string]cloudinfo.Price, error) {
	return nil, errors.New("azure prices cannot be queried on the fly")
}

// HasImages - Azure doesn't support images
func (a *AzureInfoer) HasImages() bool {
	return false
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (a *AzureInfoer) GetServiceImages(service, region string) ([]cloudinfo.Image, error) {
	return nil, errors.New("GetServiceImages - not yet implemented")
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (a *AzureInfoer) GetServiceProducts(region, service string) ([]cloudinfo.ProductDetails, error) {
	return nil, errors.New("GetServiceProducts - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (a *AzureInfoer) GetServiceAttributes(region, service, attribute string) (cloudinfo.AttrValues, error) {
	return nil, errors.New("GetServiceAttributes - not yet implemented")
}

// GetVersions retrieves the kubernetes versions supported by the given service in the given region
func (a *AzureInfoer) GetVersions(service, region string) ([]cloudinfo.LocationVersion, error) {
	switch service {
	case svcAks:
		const resourceTypeForAks = "managedClusters"
		var versions []string
		resp, err := a.containerSvcClient.ListOrchestrators(context.TODO(), region, resourceTypeForAks)
		if err != nil {
			return nil, err
		}
		if resp.OrchestratorVersionProfileProperties != nil && resp.OrchestratorVersionProfileProperties.Orchestrators != nil {
			for _, v := range *resp.OrchestratorVersionProfileProperties.Orchestrators {
				if v.OrchestratorType != nil && *v.OrchestratorType == string(containerservice.Kubernetes) {
					versions = appendIfMissing(versions, *v.OrchestratorVersion)
					if v.Upgrades != nil {
						for _, up := range *v.Upgrades {
							versions = appendIfMissing(versions, *up.OrchestratorVersion)
						}
					}
				}
			}
		}
		return []cloudinfo.LocationVersion{cloudinfo.NewLocationVersion(region, versions)}, nil
	default:
		return []cloudinfo.LocationVersion{}, nil
	}
}

// appendIfMissing appends string to a slice if it's not contains it
func appendIfMissing(slice []string, s string) []string {
	for _, e := range slice {
		if e == s {
			return slice
		}
	}
	return append(slice, s)
}
