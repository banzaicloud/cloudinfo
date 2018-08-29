package azure

import (
	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

var (
	// TODO
	ntwPerfMap = map[string][]string{
		productinfo.NTW_LOW:    {"Low"},
		productinfo.NTW_MEDIUM: {"Moderate"},
		productinfo.NTW_HIGH:   {""},
	}
)

// AzureNetworkMapper module object for handling Azure specific VM to Networking capabilities mapping
type AzureNetworkMapper struct {
}

// newAzureNetworkMapper initializes the network performance mapper struct
func newAzureNetworkMapper() *AzureNetworkMapper {
	return &AzureNetworkMapper{}
}

// MapNetworkPerf maps the network performance of the azure instance to the category supported by telescopes
func (nm *AzureNetworkMapper) MapNetworkPerf(vm productinfo.VmInfo) (string, error) {
	return ntwPerfMap[productinfo.NTW_MEDIUM][0], nil
}
