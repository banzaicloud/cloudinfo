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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2018-03-31/containerservice"

	"github.com/banzaicloud/productinfo/pkg/logger"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-04-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/preview/commerce/mgmt/2015-06-01-preview/commerce"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

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

	mtBasic, _     = regexp.Compile("^BASIC.A\\d+[_Promo]*$")
	mtStandardA, _ = regexp.Compile("^A\\d+[_Promo]*$")
	mtStandardB, _ = regexp.Compile("^Standard_B\\d+m?[_v\\d]*[_Promo]*$")
	mtStandardD, _ = regexp.Compile("^Standard_D\\d[_v\\d]*[_Promo]*$")
	mtStandardE, _ = regexp.Compile("^Standard_E\\d+i?[_v\\d]*[_Promo]*$")
	mtStandardF, _ = regexp.Compile("^Standard_F\\d+[_v\\d]*[_Promo]*$")
	mtStandardG, _ = regexp.Compile("^Standard_G\\d+[_v\\d]*[_Promo]*$")
	mtStandardL, _ = regexp.Compile("^Standard_L\\d+[_v\\d]*[_Promo]*$")
	mtStandardM, _ = regexp.Compile("^Standard_M\\d+[m|t|l]*s[_v\\d]*[_Promo]*$")
	mtStandardN, _ = regexp.Compile("^Standard_N[C|D|V]\\d+r?[_v\\d]*[_Promo]*$")
)

type authentication struct {
	ClientID                string `json:"clientId,omitempty"`
	ClientSecret            string `json:"clientSecret,omitempty"`
	SubscriptionID          string `json:"subscriptionId,omitempty"`
	TenantID                string `json:"tenantId,omitempty"`
	ActiveDirectoryEndpoint string `json:"activeDirectoryEndpointUrl,omitempty"`
	ResourceManagerEndpoint string `json:"resourceManagerEndpointUrl,omitempty"`
	GraphResourceID         string `json:"activeDirectoryGraphResourceId,omitempty"`
	SQLManagementEndpoint   string `json:"sqlManagementEndpointUrl,omitempty"`
	GalleryEndpoint         string `json:"galleryEndpointUrl,omitempty"`
	ManagementEndpoint      string `json:"managementEndpointUrl,omitempty"`
}

// AzureInfoer encapsulates the data and operations needed to access external Azure resources
type AzureInfoer struct {
	subscriptionId      string
	subscriptionsClient LocationRetriever
	vmSizesClient       VmSizesRetriever
	rateCardClient      PriceRetriever
	providersClient     ProviderSource
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

// NewAzureInfoer creates a new instance of the Azure infoer
func NewAzureInfoer() (*AzureInfoer, error) {
	authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		return nil, err
	}

	contents, err := ioutil.ReadFile(os.Getenv("AZURE_AUTH_LOCATION"))
	if err != nil {
		return nil, err
	}
	auth := authentication{}
	err = json.Unmarshal(contents, &auth)
	if err != nil {
		return nil, err
	}

	sClient := subscriptions.NewClient()
	sClient.Authorizer = authorizer

	vmClient := compute.NewVirtualMachineSizesClient(auth.SubscriptionID)
	vmClient.Authorizer = authorizer

	rcClient := commerce.NewRateCardClient(auth.SubscriptionID)
	rcClient.Authorizer = authorizer

	providersClient := resources.NewProvidersClient(auth.SubscriptionID)
	providersClient.Authorizer = authorizer

	return &AzureInfoer{
		subscriptionId:      auth.SubscriptionID,
		subscriptionsClient: sClient,
		vmSizesClient:       vmClient,
		rateCardClient:      rcClient,
		providersClient:     providersClient,
	}, nil
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
	return "", fmt.Errorf("couldn't find region: %s", meterRegion)
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
func (a *AzureInfoer) Initialize(ctx context.Context) (map[string]map[string]productinfo.Price, error) {
	log := logger.Extract(ctx)
	log.Debug("initializing price info")
	allPrices := make(map[string]map[string]productinfo.Price)

	regions, err := a.GetRegions(ctx, "compute")
	if err != nil {
		return nil, err
	}

	log.Debugf("queried regions: %v", regions)

	rateCardFilter := "OfferDurableId eq 'MS-AZR-0003p' and Currency eq 'USD' and Locale eq 'en-US' and RegionInfo eq 'US'"
	result, err := a.rateCardClient.Get(context.TODO(), rateCardFilter)
	if err != nil {
		return nil, err
	}
	for _, v := range *result.Meters {
		if *v.MeterCategory == "Virtual Machines" && len(*v.MeterTags) == 0 && *v.MeterRegion != "" {
			if !strings.Contains(*v.MeterSubCategory, "Windows") {
				region, err := a.toRegionID(*v.MeterRegion, regions)
				if err != nil {
					log.WithError(err).Debug()
					continue
				}
				var instanceTypes []string

				instanceTypes = a.machineType(*v.MeterName, *v.MeterSubCategory)

				var priceInUsd float64

				if len(v.MeterRates) < 1 {
					log.WithField("region", *v.MeterRegion).Debugf("%s doesn't have rate info in region %s", *v.MeterSubCategory, *v.MeterRegion)
					continue
				}
				for _, rate := range v.MeterRates {
					priceInUsd += *rate
				}
				if allPrices[region] == nil {
					allPrices[region] = make(map[string]productinfo.Price)
				}
				for _, instanceType := range instanceTypes {
					price := allPrices[region][instanceType]
					if !strings.Contains(*v.MeterName, "Low Priority") {
						price.OnDemandPrice = priceInUsd
					} else {
						spotPrice := make(productinfo.SpotPriceInfo)
						spotPrice[region] = priceInUsd
						price.SpotPrice = spotPrice
					}

					allPrices[region][instanceType] = price
					log.WithField("region", region).Debugf("price info added: [machinetype=%s, price=%v]", instanceType, price)
					mts := a.getMachineTypeVariants(instanceType)
					for _, mt := range mts {
						allPrices[region][mt] = price
						log.WithField("region", region).Debugf("price info added: [machinetype=%s, price=%v]", mt, price)
					}
				}
			}
		}
	}

	log.Debug("finished initializing price info")
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
		instanceTypes[0] = instanceTypes[0] + "_Promo"
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
		return []string{"Basic_" + mt[0]}
	case len(mt) == 2:
		return []string{"Standard_" + mt[0], "Standard_" + mt[1]}
	default:
		return []string{"Standard_" + mt[0]}
	}
}

func (a *AzureInfoer) getMachineTypeVariants(mt string) []string {
	switch {
	case mtStandardB.MatchString(mt):
		return []string{mt + "s"}
	case mtStandardD.MatchString(mt):
		result := make([]string, 6)
		result[0] = a.addSuffix(mt, "s")[0]
		dsType := strings.Replace(mt, "Standard_D", "Standard_DS", -1)
		result[1] = dsType
		for i, s := range a.addSuffix(dsType, "-1", "-2", "-4", "-8") {
			result[i+2] = s
		}
		return result
	case mtStandardE.MatchString(mt):
		result := make([]string, 6)
		for i, s := range a.addSuffix(mt, "s", "-2s", "-4s", "-8s", "-16s", "-32s") {
			result[i] = s
		}
		return result
	case mtStandardF.MatchString(mt):
		return a.addSuffix(mt, "s")
	case mtStandardG.MatchString(mt):
		result := make([]string, 4)
		gsType := strings.Replace(mt, "Standard_G", "Standard_GS", -1)
		result[0] = gsType
		for i, s := range a.addSuffix(gsType, "-4", "-8", "-16") {
			result[i+1] = s
		}
		return result
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
			suffix += "_" + p
		}
	}
	for i, s := range suffixes {
		result[i] = parts[0] + "_" + parts[1] + s + suffix
	}
	return result
}

// GetAttributeValues gets the AttributeValues for the given attribute name
func (a *AzureInfoer) GetAttributeValues(ctx context.Context, service, attribute string) (productinfo.AttrValues, error) {
	log := logger.Extract(ctx)

	log.Debugf("getting %s values", attribute)

	values := make(productinfo.AttrValues, 0)
	valueSet := make(map[productinfo.AttrValue]interface{})

	regions, err := a.GetRegions(ctx, service)
	if err != nil {
		return nil, err
	}

	for region := range regions {
		vmSizes, err := a.vmSizesClient.List(context.TODO(), region)
		if err != nil {
			log.WithField("region", region).WithError(err).Warn("couldn't get VM sizes")
			continue
		}
		switch service {
		case "aks":
			possibleVmTypes := containerservice.PossibleVMSizeTypesValues()
			for _, v := range *vmSizes.Value {
				for _, vm := range possibleVmTypes {
					if string(vm) == *v.Name {
						switch attribute {
						case productinfo.Cpu:
							valueSet[productinfo.AttrValue{
								Value:    float64(*v.NumberOfCores),
								StrValue: fmt.Sprintf("%v", *v.NumberOfCores),
							}] = ""
						case productinfo.Memory:
							valueSet[productinfo.AttrValue{
								Value:    float64(*v.MemoryInMB) / 1024,
								StrValue: fmt.Sprintf("%v", *v.MemoryInMB),
							}] = ""
						}
					}
				}
			}
		default:
			for _, v := range *vmSizes.Value {
				switch attribute {
				case productinfo.Cpu:
					valueSet[productinfo.AttrValue{
						Value:    float64(*v.NumberOfCores),
						StrValue: fmt.Sprintf("%v", *v.NumberOfCores),
					}] = ""
				case productinfo.Memory:
					valueSet[productinfo.AttrValue{
						Value:    float64(*v.MemoryInMB) / 1024,
						StrValue: fmt.Sprintf("%v", *v.MemoryInMB),
					}] = ""
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
func (a *AzureInfoer) GetProducts(ctx context.Context, service, regionId string) ([]productinfo.VmInfo, error) {
	log := logger.Extract(ctx)
	log.Debug("getting product info")
	var vms []productinfo.VmInfo
	vmSizes, err := a.vmSizesClient.List(context.TODO(), regionId)
	if err != nil {
		return nil, err
	}
	switch service {
	case "aks":
		possibleVmTypes := containerservice.PossibleVMSizeTypesValues()
		for _, v := range *vmSizes.Value {
			for _, vm := range possibleVmTypes {
				if string(vm) == *v.Name {
					vms = append(vms, productinfo.VmInfo{
						Type: *v.Name,
						Cpus: float64(*v.NumberOfCores),
						Mem:  float64(*v.MemoryInMB) / 1024,
						// TODO: netw perf
					})
				}
			}
		}
	default:
		for _, v := range *vmSizes.Value {
			vms = append(vms, productinfo.VmInfo{
				Type: *v.Name,
				Cpus: float64(*v.NumberOfCores),
				Mem:  float64(*v.MemoryInMB) / 1024,
				// TODO: netw perf
			})
		}
	}

	log.Debugf("found vms: %#v", vms)
	return vms, nil
}

// GetZones returns the availability zones in a region
func (a *AzureInfoer) GetZones(ctx context.Context, region string) ([]string, error) {
	return []string{region}, nil
}

// GetRegions returns a map with available regions transforms the api representation into a "plain" map
func (a *AzureInfoer) GetRegions(ctx context.Context, service string) (map[string]string, error) {
	log := logger.Extract(ctx)

	allLocations := make(map[string]string)
	supLocations := make(map[string]string)

	// retrieve all locations for the subscription id (some of them may not be supported by the required provider)
	if locations, err := a.subscriptionsClient.ListLocations(context.TODO(), a.subscriptionId); err == nil {
		// fill up the map: DisplayName - > Name
		for _, loc := range *locations.Value {
			allLocations[*loc.DisplayName] = *loc.Name
		}
	} else {
		log.WithError(err).Error("error while retrieving azure locations")
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
							log.WithField("region", loc).Debugf("found supported location. [name, display name] = [%s, %s]", loc, displName)
							supLocations[loc] = displName
						} else {
							log.Debugf("unsupported location. [name, display name] = [%s, %s]", loc, displName)
						}
					}
					break
				}
			}
		} else {
			log.WithError(err).Errorf("error while retrieving supported locations for provider: %s.", resourceTypeForAks)
			return nil, err
		}

		return supLocations, nil
	default:
		if providers, err := a.providersClient.Get(context.TODO(), providerNamespaceForCompute, ""); err == nil {
			for _, pr := range *providers.ResourceTypes {
				if *pr.ResourceType == resourceTypeForCompute {
					for _, displName := range *pr.Locations {
						if loc, ok := allLocations[displName]; ok {
							log.WithField("region", loc).Debugf("found supported location. [name, display name] = [%s, %s]", loc, displName)
							supLocations[loc] = displName
						} else {
							log.Debugf("unsupported location. [name, display name] = [%s, %s]", loc, displName)
						}
					}
					break
				}
			}
		} else {
			log.WithError(err).Errorf("error while retrieving supported locations for provider: %s.", resourceTypeForCompute)
			return nil, err
		}

		return supLocations, nil
	}
}

// HasShortLivedPriceInfo - Azure doesn't have frequently changing prices
func (a *AzureInfoer) HasShortLivedPriceInfo() bool {
	return false
}

// GetCurrentPrices retrieves all the price info in a region
func (a *AzureInfoer) GetCurrentPrices(ctx context.Context, region string) (map[string]productinfo.Price, error) {
	return nil, errors.New("azure prices cannot be queried on the fly")
}

// GetMemoryAttrName returns the provider representation of the memory attribute
func (a *AzureInfoer) GetMemoryAttrName() string {
	return productinfo.Memory
}

// GetCpuAttrName returns the provider representation of the cpu attribute
func (a *AzureInfoer) GetCpuAttrName() string {
	return productinfo.Cpu
}

// GetNetworkPerformanceMapper returns the network performance mapper implementation for this provider
func (a *AzureInfoer) GetNetworkPerformanceMapper() (productinfo.NetworkPerfMapper, error) {
	return newAzureNetworkMapper(), nil
}

// GetServices returns the available services on the  provider
func (a *AzureInfoer) GetServices() ([]productinfo.ServiceDescriber, error) {
	services := []productinfo.ServiceDescriber{
		productinfo.NewService("compute"),
		productinfo.NewService("aks")}
	return services, nil
}

// GetService returns the service on the provider
func (a *AzureInfoer) GetService(ctx context.Context, service string) (productinfo.ServiceDescriber, error) {
	svcs, err := a.GetServices()
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

// HasImages - Azure doesn't support images
func (a *AzureInfoer) HasImages() bool {
	return false
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (a *AzureInfoer) GetServiceImages(region, service string) ([]productinfo.ImageDescriber, error) {
	return nil, fmt.Errorf("GetServiceImages - not yet implemented")
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (a *AzureInfoer) GetServiceProducts(region, service string) ([]productinfo.ProductDetails, error) {
	return nil, fmt.Errorf("GetServiceProducts - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (a *AzureInfoer) GetServiceAttributes(region, service, attribute string) (productinfo.AttrValues, error) {
	return nil, fmt.Errorf("GetServiceAttributes - not yet implemented")
}
