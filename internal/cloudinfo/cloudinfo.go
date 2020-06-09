// Copyright © 2019 Banzai Cloud
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

package cloudinfo

import (
	"strings"

	"emperror.dev/errors"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

// cloudInfo is the module struct, holds configuration and cache
// It's the entry point for the product info retrieval and management subsystem
// It's also responsible for delegating to the cloud provider specific implementations
type cloudInfo struct {
	log            Logger
	providers      []string
	cloudInfoStore CloudInfoStore
}

// NewCloudInfo creates a new cloudInfo instance
func NewCloudInfo(providers []string, ciStore CloudInfoStore, logger Logger) (*cloudInfo, error) {
	if providers == nil || ciStore == nil {
		return nil, errors.New("could not create product infoer")
	}

	pi := cloudInfo{
		providers:      providers,
		cloudInfoStore: ciStore,
		log:            logger.WithFields(map[string]interface{}{"component": "cloudInfo"}),
	}
	return &pi, nil
}

// GetProviders returns the supported providers
func (cpi *cloudInfo) GetProviders() ([]types.Provider, error) {
	var (
		providers []types.Provider
		provider  types.Provider
		err       error
	)

	// iterate over supported provider names only
	for _, pn := range cpi.providers {
		if provider, err = cpi.GetProvider(pn); err != nil {
			return nil, err
		}

		providers = append(providers, provider)
	}

	return providers, nil
}

// GetProvider returns the supported provider
func (cpi *cloudInfo) GetProvider(provider string) (types.Provider, error) {
	var (
		srvcs []types.Service
		err   error
	)

	if !cpi.providerEnabled(provider) {
		return types.Provider{}, errors.NewWithDetails("unsupported provider", "provider", provider)
	}

	if srvcs, err = cpi.GetServices(provider); err != nil {
		return types.Provider{}, errors.WithDetails(err, "failed to get services", "provider", provider)
	}

	// decorate the provider with service information
	p := types.NewProvider(provider)
	p.Services = srvcs

	return p, nil
}

func (cpi *cloudInfo) providerEnabled(provider string) bool {
	var enabled = false

	for _, p := range cpi.providers {
		if p == provider {
			enabled = true
			break
		}
	}

	return enabled
}

// GetZones returns the availability zones in a region
func (cpi *cloudInfo) GetZones(provider, service, region string) ([]string, error) {
	if cachedVal, ok := cpi.cloudInfoStore.GetZones(provider, service, region); ok {
		return cachedVal, nil
	}

	return nil, errors.NewWithDetails("zones not yet cached", "provider", provider, "region", region)
}

// GetRegions gets the regions for the provided provider
func (cpi *cloudInfo) GetRegions(provider, service string) (map[string]string, error) {
	if cachedVal, ok := cpi.cloudInfoStore.GetRegions(provider, service); ok {
		return cachedVal, nil
	}

	return nil, errors.NewWithDetails("regions not yet cached", "provider", provider, "services", service)
}

func (cpi *cloudInfo) GetServices(provider string) ([]types.Service, error) {
	if cachedVal, ok := cpi.cloudInfoStore.GetServices(provider); ok {
		return cachedVal, nil
	}

	return nil, errors.NewWithDetails("services not yet cached", "provider", provider)
}

// GetProductDetails retrieves product details form the given provider and region
func (cpi *cloudInfo) GetProductDetails(provider, service, region string) ([]types.ProductDetails, error) {
	vms, ok := cpi.cloudInfoStore.GetVm(provider, service, region)
	if !ok {
		cpi.log.Debug("VMs not yet cached")
		return nil, errors.NewWithDetails("VMs not yet cached", "provider", provider, "service", service, "region", region)
	}

	details := make([]types.ProductDetails, 0, len(vms))
	for _, vm := range vms {
		pd := types.NewProductDetails(vm)
		cachedVal, ok := cpi.cloudInfoStore.GetPrice(provider, region, vm.Type)
		if !ok {
			cpi.log.Debug("price info not yet cached", map[string]interface{}{"instanceType": vm.Type})
		}

		for zone, price := range cachedVal.SpotPrice {
			pd.SpotPrice = append(pd.SpotPrice, *types.NewZonePrice(zone, price))
		}

		details = append(details, *pd)
	}

	return details, nil
}

// GetStatus retrieves status form the given provider
func (cpi *cloudInfo) GetStatus(provider string) (string, error) {
	if cachedStatus, ok := cpi.cloudInfoStore.GetStatus(provider); ok {
		return cachedStatus, nil
	}
	return "", errors.NewWithDetails("status not yet cached", "provider", provider)
}

// GetServiceImages retrieves available images for the given provider, service and region
func (cpi *cloudInfo) GetServiceImages(provider, service, region string) ([]types.Image, error) {
	if cachedImages, ok := cpi.cloudInfoStore.GetImage(provider, service, region); ok {
		return cachedImages, nil
	}

	return nil, errors.NewWithDetails("images not yet cached", "provider", provider,
		"service", service, "region", region)
}

// GetVersions retrieves available versions for the given provider, service and region
func (cpi *cloudInfo) GetVersions(provider, service, region string) ([]types.LocationVersion, error) {
	if cachedVersions, ok := cpi.cloudInfoStore.GetVersion(provider, service, region); ok {
		return cachedVersions, nil
	}
	return nil, errors.NewWithDetails("versions not yet cached", "provider", provider,
		"service", service, "region", region)
}

// GetContinents retrieves available continents
func (cpi *cloudInfo) GetContinents() []string {
	return []string{types.ContinentAsia, types.ContinentAustralia, types.ContinentEurope, types.ContinentNorthAmerica, types.ContinentSouthAmerica}
}

// GetContinents gets the continents and regions for the provided provider
func (cpi *cloudInfo) GetContinentsData(provider, service string) (map[string][]types.Region, error) {
	if cachedVal, ok := cpi.cloudInfoStore.GetRegions(provider, service); ok {
		continents := make(map[string][]types.Region)
		for id, name := range cachedVal {
			continent := getContinent(id)
			continents[continent] = append(continents[continent], types.Region{
				ID:   id,
				Name: name,
			})
		}
		return continents, nil
	}

	return nil, errors.NewWithDetails("regions not yet cached", "provider", provider, "services", service)
}

// getContinent categorizes regions by continents
func getContinent(region string) string {
	switch {
	case checkContinent(region, []string{"ap-southeast-2", "australia"}):
		return types.ContinentAustralia
	case checkContinent(region, []string{"cn-", "ap-", "me-", "asia", "japan", "india", "korea"}),
		strings.HasPrefix(region, "sgp"),
		strings.HasPrefix(region, "blr"):
		return types.ContinentAsia
	case checkContinent(region, []string{"eu", "uk", "france"}),
		strings.HasPrefix(region, "ams"),
		strings.HasPrefix(region, "lon"),
		strings.HasPrefix(region, "fra"):
		return types.ContinentEurope
	case checkContinent(region, []string{"us", "ca-central-1", "canada", "northamerica"}),
		strings.HasPrefix(region, "nyc"),
		strings.HasPrefix(region, "sfo"),
		strings.HasPrefix(region, "tor"):
		return types.ContinentNorthAmerica
	case checkContinent(region, []string{"southamerica", "brazil", "sa-"}):
		return types.ContinentSouthAmerica
	case checkContinent(region, []string{"africa"}):
		return types.ContinentAfrica
	default:
		return "unknown"
	}
}

func checkContinent(region string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(region, substr) {
			return true
		}
	}
	return false
}

// Contains is a helper function to check if a slice contains a string
func Contains(slice []string, s string) bool {
	for _, e := range slice {
		if e == s {
			return true
		}
	}
	return false
}

// Attributes create a map with the specified parameters
func Attributes(cpu, memory, ntwPerfCat, vmCategory string) map[string]string {
	var attributes = make(map[string]string)

	attributes[types.CPU] = cpu
	attributes[types.Memory] = memory
	attributes["networkPerfCategory"] = ntwPerfCat
	attributes["instanceTypeCategory"] = vmCategory

	return attributes
}

// CreateString builds strings effectively
func CreateString(keys ...string) string {
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
	}
	return b.String()
}
