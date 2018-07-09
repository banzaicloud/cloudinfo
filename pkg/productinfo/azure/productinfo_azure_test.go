package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			azureInfoer := AzureInfoer{}
			test.check(azureInfoer.toRegionID(test.sourceRegion, regionMap))
		})
	}
}

func TestAzureInfoer_transformMachineType(t *testing.T) {
	tests := []struct {
		name     string
		sourceMt string
		check    func(mt string)
	}{
		{
			name:     "return source if transformation not needed",
			sourceMt: "Standard_A1_v2",
			check: func(mt string) {
				assert.Equal(t, "Standard_A1_v2", mt, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Basic.A",
			sourceMt: "BASIC.A8",
			check: func(mt string) {
				assert.Equal(t, "Basic_A8", mt, "invalid machine type returned")
			},
		},
		{
			name:     "successful check for Standard_A",
			sourceMt: "A6",
			check: func(mt string) {
				assert.Equal(t, "Standard_A6", mt, "invalid machine type returned")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			azureInfoer := AzureInfoer{}
			test.check(azureInfoer.transformMachineType(test.sourceMt))
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
				assert.Equal(t, []string{"Standard_M16-2ms", "Standard_M16-4ms", "Standard_M16-8ms", "Standard_M16-16ms", "Standard_M16-32ms"}, mts, "invalid machine type returned")
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
			azureInfoer := AzureInfoer{}
			test.check(azureInfoer.getMachineTypeVariants(test.sourceMt))
		})
	}
}
