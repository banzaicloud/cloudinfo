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
	"testing"

	"emperror.dev/errors"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-09-01/skus"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2018-03-31/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/preview/commerce/mgmt/2015-06-01-preview/commerce"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2016-06-01/subscriptions"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-05-01/resources"
	"github.com/stretchr/testify/assert"
	"logur.dev/logur"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/cloudinfoadapter"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

// testStruct helps to mock external calls
type testStruct struct {
	TcId string
}

// test helps to mock external calls
type test struct {
	TcId string
}

const (
	GetVmsError      = "could not get virtual machines"
	GetRegionsError  = "could not get regions"
	GetLocationError = "could not get location"
	GetPriceError    = "could not get prices"
)

func (dps *testStruct) ListLocations(ctx context.Context, subscriptionID string) (result subscriptions.LocationListResult, err error) {
	switch dps.TcId {
	case GetRegionsError:
		return subscriptions.LocationListResult{}, errors.New(GetRegionsError)
	default:
		return subscriptions.LocationListResult{
			Value: &[]subscriptions.Location{
				{
					Name:        strPointer("eastasia"),
					DisplayName: strPointer("East Asia"),
				},
				{
					Name:        strPointer("centralus"),
					DisplayName: strPointer("Central US"),
				},
				{
					Name:        strPointer("westeurope"),
					DisplayName: strPointer("West Europe"),
				},
			},
		}, nil
	}
}

func (dps *testStruct) Get(ctx context.Context, resourceProviderNamespace string, expand string) (result resources.Provider, err error) {
	switch dps.TcId {
	case GetLocationError:
		return resources.Provider{}, errors.New(GetLocationError)
	default:
		return resources.Provider{
			ResourceTypes: &[]resources.ProviderResourceType{
				{
					ResourceType: strPointer("locations/vmSizes"),
					Locations:    &[]string{"West Europe", "Central US", "East Asia"},
				},
				{
					ResourceType: strPointer("managedClusters"),
					Locations:    &[]string{"West Europe", "East Asia"},
				},
			},
		}, nil
	}
}

func (dps *test) Get(ctx context.Context, filter string) (result commerce.ResourceRateCardInfo, err error) {
	switch dps.TcId {
	case GetPriceError:
		return commerce.ResourceRateCardInfo{}, errors.New(GetPriceError)
	default:
		return commerce.ResourceRateCardInfo{
			Meters: &[]commerce.MeterInfo{
				{
					MeterName:        strPointer("F2/F2s Low Priority"),
					MeterCategory:    strPointer("Virtual Machines"),
					MeterSubCategory: strPointer("F/FS Series"),
					MeterRegion:      strPointer("EU West"),
					MeterRates: map[string]*float64{
						"0": floatPointer(0.077),
					},
					MeterTags: &[]string{},
				},
				{
					MeterName:        strPointer("F2/F2s"),
					MeterCategory:    strPointer("Virtual Machines"),
					MeterSubCategory: strPointer("F/FS Series"),
					MeterRegion:      strPointer("EU West"),
					MeterRates: map[string]*float64{
						"0": floatPointer(0.332),
					},
					MeterTags: &[]string{},
				},
				{
					MeterName:        strPointer("F2/F2s"),
					MeterCategory:    strPointer("Virtual Machines"),
					MeterSubCategory: strPointer("F/FS Series"),
					MeterRegion:      strPointer("US Central"),
					MeterRates: map[string]*float64{
						"0": floatPointer(0.132),
					},
					MeterTags: &[]string{},
				},
			},
		}, nil
	}
}

func (dps *testStruct) ListOrchestrators(ctx context.Context, location string, resourceType string) (result containerservice.OrchestratorVersionProfileListResult, err error) {
	return containerservice.OrchestratorVersionProfileListResult{}, nil
}

func (dps *testStruct) List(ctx context.Context) (result skus.ResourceSkusResultPage, err error) {
	return skus.ResourceSkusResultPage{}, nil
}

// strPointer gets the pointer to the passed string
func strPointer(str string) *string {
	return &str
}

// floatPointer gets the pointer to the passed float64
func floatPointer(i float64) *float64 {
	return &i
}

func TestAzureInfoer_toRegionID(t *testing.T) {

	regionMap := map[string]string{
		"japanwest":          "Japan West",
		"centralindia":       "Central India",
		"francesouth":        "France South",
		"northcentralus":     "North Central US",
		"japaneast":          "Japan East",
		"australiaeast":      "Australia East",
		"southindia":         "South India",
		"canadaeast":         "Canada East",
		"westus2":            "West US 2",
		"westus":             "West US",
		"northeurope":        "North Europe",
		"westeurope":         "West Europe",
		"uksouth":            "UK South",
		"centralus":          "Central US",
		"australiasoutheast": "Australia Southeast",
		"ukwest":             "UK West",
		"koreacentral":       "Korea Central",
		"koreanorthcentral":  "Korea North Central",
		"koreanorthcentral2": "Korea North Central 2",
		"francecentral":      "France Central",
		"eastasia":           "East Asia",
		"canadacentral":      "Canada Central",
		"eastus":             "East US",
		"eastus2":            "East US 2",
		"southcentralus":     "South Central US",
		"southcentralus2":    "South Central US 2",
		"australiacentral":   "Australia Central",
		"westindia":          "West India",
		"koreasouth":         "Korea South",
		"australiacentral2":  "Australia Central 2",
		"southeastasia":      "Southeast Asia",
		"brazilsouth":        "Brazil South",
		"westcentralus":      "West Central US",
	}

	tests := []struct {
		name         string
		sourceRegion string
		check        func(regionId string, err error)
	}{
		{
			name:         "successful check without postfix, len = 2",
			sourceRegion: "JA West",
			check: func(regionId string, err error) {
				assert.Equal(t, "japanwest", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check without postfix, len = 2, inverted",
			sourceRegion: "EU North",
			check: func(regionId string, err error) {
				assert.Equal(t, "northeurope", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check without postfix, len = 3",
			sourceRegion: "KR North Central",
			check: func(regionId string, err error) {
				assert.Equal(t, "koreanorthcentral", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check without postfix, len = 3, inverted",
			sourceRegion: "US North Central",
			check: func(regionId string, err error) {
				assert.Equal(t, "northcentralus", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check with postfix, len = 2",
			sourceRegion: "AU Central 2",
			check: func(regionId string, err error) {
				assert.Equal(t, "australiacentral2", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check with postfix, len = 2, inverted",
			sourceRegion: "US West 2",
			check: func(regionId string, err error) {
				assert.Equal(t, "westus2", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check with postfix, len = 3",
			sourceRegion: "KR North Central 2",
			check: func(regionId string, err error) {
				assert.Equal(t, "koreanorthcentral2", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check with postfix, len = 3, inverted",
			sourceRegion: "US South Central 2",
			check: func(regionId string, err error) {
				assert.Equal(t, "southcentralus2", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check india",
			sourceRegion: "IN Central",
			check: func(regionId string, err error) {
				assert.Equal(t, "centralindia", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check brazil",
			sourceRegion: "BR South",
			check: func(regionId string, err error) {
				assert.Equal(t, "brazilsouth", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check france",
			sourceRegion: "FR South",
			check: func(regionId string, err error) {
				assert.Equal(t, "francesouth", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check canada",
			sourceRegion: "CA Central",
			check: func(regionId string, err error) {
				assert.Equal(t, "canadacentral", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check asia",
			sourceRegion: "AP East",
			check: func(regionId string, err error) {
				assert.Equal(t, "eastasia", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "successful check uk",
			sourceRegion: "UK West",
			check: func(regionId string, err error) {
				assert.Equal(t, "ukwest", regionId, "invalid region ID returned")
				assert.Nil(t, err, "error should be nil")
			},
		},
		{
			name:         "check not supported region",
			sourceRegion: "US Gov TX",
			check: func(regionId string, err error) {
				assert.Empty(t, regionId, "empty region ID should be returned")
				assert.Equal(t, "couldn't find region: US Gov TX", err.Error(), "error should be ")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			azureInfoer := AzureInfoer{log: cloudinfoadapter.NewLogger(logur.NewTestLogger())}
			test.check(azureInfoer.toRegionID(test.sourceRegion, regionMap))
		})
	}
}

func TestAzureInfoer_transformMachineType(t *testing.T) {
	tests := []struct {
		name        string
		subCategory string
		sourceMt    []string
		check       func(mt []string)
	}{
		{
			name:     "return source if transformation not needed",
			sourceMt: []string{"A1_v2"},
			check: func(mt []string) {
				assert.Equal(t, []string([]string{"Standard_A1_v2"}), mt, "invalid machine type returned")
			},
		},
		{
			name:        "successful check for Basic.A",
			subCategory: "A Series Basic",
			sourceMt:    []string{"A8"},
			check: func(mt []string) {
				assert.Equal(t, []string([]string{"Basic_A8"}), mt, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_A",
			sourceMt: []string{"A6"},
			check: func(mt []string) {
				assert.Equal(t, []string([]string{"Standard_A6"}), mt, "invalid machine type returned")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			azureInfoer := AzureInfoer{log: cloudinfoadapter.NewLogger(logur.NewTestLogger())}
			test.check(azureInfoer.transformMachineType(test.subCategory, test.sourceMt))
		})
	}
}

func TestAzureInfoer_getMachineTypeVariants(t *testing.T) {

	tests := []struct {
		name     string
		sourceMt string
		check    func(mts []string)
	}{
		{
			name:     "return source if transformation not needed",
			sourceMt: "Basic_A3",
			check: func(mts []string) {
				assert.Equal(t, []string{}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_B",
			sourceMt: "Standard_B2",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_B2s"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_B..m",
			sourceMt: "Standard_B8m",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_B8ms"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_D",
			sourceMt: "Standard_D16",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_D16s", "Standard_DS16", "Standard_DS16-1", "Standard_DS16-2", "Standard_DS16-4", "Standard_DS16-8"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_D.._v3",
			sourceMt: "Standard_D16_v3",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_D16s_v3", "Standard_DS16_v3", "Standard_DS16-1_v3", "Standard_DS16-2_v3", "Standard_DS16-4_v3", "Standard_DS16-8_v3"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_D.._v2_Promo",
			sourceMt: "Standard_D12_v2_Promo",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_D12s_v2_Promo", "Standard_DS12_v2_Promo", "Standard_DS12-1_v2_Promo", "Standard_DS12-2_v2_Promo", "Standard_DS12-4_v2_Promo", "Standard_DS12-8_v2_Promo"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_E.._v3",
			sourceMt: "Standard_E16_v3",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_E16s_v3", "Standard_E16-2s_v3", "Standard_E16-4s_v3", "Standard_E16-8s_v3", "Standard_E16-16s_v3", "Standard_E16-32s_v3"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_E64i_v3",
			sourceMt: "Standard_E64i_v3",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_E64is_v3", "Standard_E64i-2s_v3", "Standard_E64i-4s_v3", "Standard_E64i-8s_v3", "Standard_E64i-16s_v3", "Standard_E64i-32s_v3"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_F4",
			sourceMt: "Standard_F4",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_F4s"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_F4_v2",
			sourceMt: "Standard_F4_v2",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_F4s_v2"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_G1",
			sourceMt: "Standard_G1",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_GS1", "Standard_GS1-4", "Standard_GS1-8", "Standard_GS1-16"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_H16mr",
			sourceMt: "Standard_H16mr",
			check: func(mts []string) {
				assert.Equal(t, []string{}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_L4",
			sourceMt: "Standard_L4",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_L4s"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_L16_v2",
			sourceMt: "Standard_L16_v2",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_L16s_v2"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_ND12",
			sourceMt: "Standard_ND12",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_ND12s"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_ND24r",
			sourceMt: "Standard_ND24r",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_ND24rs"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_M16ms",
			sourceMt: "Standard_M16ms",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_M16-2ms", "Standard_M16-4ms", "Standard_M16-8ms", "Standard_M16-16ms", "Standard_M16-32ms", "Standard_M16-64ms"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_M32s",
			sourceMt: "Standard_M32s",
			check: func(mts []string) {
				assert.Equal(t, []string{"Standard_M32", "Standard_M32m"}, mts, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_M32ls",
			sourceMt: "Standard_M32ls",
			check: func(mts []string) {
				assert.Equal(t, []string{}, mts, "invalid machine type returned")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			azureInfoer := AzureInfoer{log: cloudinfoadapter.NewLogger(logur.NewTestLogger())}
			test.check(azureInfoer.getMachineTypeVariants(test.sourceMt))
		})
	}
}

func TestAzureInfoer_GetProducts(t *testing.T) {
	vms := []types.VMInfo{
		{
			Type: "Standard_A1",
			Mem:  32,
			Cpus: 4,
		},
		{
			Type: "Standard_A10",
			Cpus: 8,
			Mem:  32,
		},
		{
			Type: "dummy",
			Gpus: 1,
			Mem:  5,
			Cpus: 2,
		},
	}
	tests := []struct {
		name    string
		service string
		vmSizes ResourceSkuRetriever
		check   func(vms []types.VMInfo, err error)
	}{
		{
			name:    "retrieve the available virtual machines for aks service",
			service: "aks",
			vmSizes: &testStruct{},
			check: func(vms []types.VMInfo, err error) {
				assert.Nil(t, err, "the error should be nil")
				var cpus []float64
				var mems []float64

				for _, vm := range vms {
					cpus = append(cpus, vm.Cpus)
					mems = append(mems, vm.Mem)
				}
				assert.ElementsMatch(t, cpus, []float64{4, 8})
				assert.ElementsMatch(t, mems, []float64{32, 32})
			},
		},
		{
			name:    "could not retrieve virtual machines",
			service: "dummy",
			vmSizes: &testStruct{GetVmsError},
			check: func(vms []types.VMInfo, err error) {
				assert.Nil(t, vms, "the vms should be nil")
				assert.EqualError(t, err, "invalid service: dummy")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			azureInfoer := AzureInfoer{log: cloudinfoadapter.NewLogger(logur.NewTestLogger())}

			test.check(azureInfoer.GetProducts(vms, test.service, "dummyRegion"))
		})
	}
}

func TestAzureInfoer_GetRegions(t *testing.T) {
	tests := []struct {
		name      string
		service   string
		location  LocationRetriever
		providers ProviderSource
		check     func(regions map[string]string, err error)
	}{
		{
			name:      "receive all regions for compute service",
			service:   "compute",
			location:  &testStruct{},
			providers: &testStruct{},
			check: func(regions map[string]string, err error) {
				assert.Equal(t, map[string]string{"westeurope": "West Europe", "centralus": "Central US", "eastasia": "East Asia"}, regions)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:      "receive all regions for aks service",
			service:   "aks",
			location:  &testStruct{},
			providers: &testStruct{},
			check: func(regions map[string]string, err error) {
				assert.Equal(t, map[string]string{"westeurope": "West Europe", "eastasia": "East Asia"}, regions)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:      "could not retrieve regions",
			service:   "compute",
			location:  &testStruct{GetRegionsError},
			providers: &testStruct{},
			check: func(regions map[string]string, err error) {
				assert.Nil(t, regions, "the regions should be nil")
				assert.EqualError(t, err, GetRegionsError)
			},
		},
		{
			name:      "invalid location for compute service",
			service:   "compute",
			location:  &testStruct{},
			providers: &testStruct{GetLocationError},
			check: func(regions map[string]string, err error) {
				assert.Nil(t, regions, "the regions should be nil")
				assert.EqualError(t, err, GetLocationError)
			},
		},
		{
			name:      "invalid location for aks service",
			service:   "aks",
			location:  &testStruct{},
			providers: &testStruct{GetLocationError},
			check: func(regions map[string]string, err error) {
				assert.Nil(t, regions, "the regions should be nil")
				assert.EqualError(t, err, GetLocationError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			azureInfoer := AzureInfoer{log: cloudinfoadapter.NewLogger(logur.NewTestLogger())}

			azureInfoer.subscriptionsClient = test.location
			azureInfoer.providersClient = test.providers
			test.check(azureInfoer.GetRegions(test.service))
		})
	}
}

func TestAzureInfoer_Initialize(t *testing.T) {
	tests := []struct {
		name      string
		location  LocationRetriever
		providers ProviderSource
		price     PriceRetriever
		check     func(prices map[string]map[string]types.Price, err error)
	}{
		{
			name:      "success",
			location:  &testStruct{},
			providers: &testStruct{},
			price:     &test{},
			check: func(prices map[string]map[string]types.Price, err error) {
				var onDemandPrice []float64
				var spotPrice []float64
				for _, allPrices := range prices {
					for _, price := range allPrices {
						onDemandPrice = append(onDemandPrice, price.OnDemandPrice)
						for _, spot := range price.SpotPrice {
							spotPrice = append(spotPrice, spot)
						}
					}
				}
				assert.ElementsMatch(t, onDemandPrice, []float64{0.332, 0.332, 0.132, 0.132})
				assert.ElementsMatch(t, spotPrice, []float64{0.077, 0.077})
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:      "could not retrieve regions",
			location:  &testStruct{GetRegionsError},
			providers: &testStruct{},
			price:     &test{},
			check: func(prices map[string]map[string]types.Price, err error) {
				assert.EqualError(t, err, GetRegionsError)
				assert.Nil(t, prices, "the prices should be nil")
			},
		},
		{
			name:      "could not retrieve prices",
			location:  &testStruct{},
			providers: &testStruct{},
			price:     &test{GetPriceError},
			check: func(prices map[string]map[string]types.Price, err error) {
				assert.EqualError(t, err, GetPriceError)
				assert.Nil(t, prices, "the prices should be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			azureInfoer := AzureInfoer{log: cloudinfoadapter.NewLogger(logur.NewTestLogger())}

			azureInfoer.subscriptionsClient = test.location
			azureInfoer.providersClient = test.providers
			azureInfoer.rateCardClient = test.price
			test.check(azureInfoer.Initialize())
		})
	}
}
