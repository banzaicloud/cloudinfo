package gce

import (
	"fmt"

	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

var (
	ntwPerfMap = map[string][]string{
		productinfo.NTW_LOW:    {"1", "2"},
		productinfo.NTW_MEDIUM: {"4", "6", "8"},
		productinfo.NTW_HIGH:   {"10", "12", "14"},
		productinfo.NTW_EXTRA:  {"16"},
	}
)

// GceNetworkMapper module object for handling Google Cloud specific VM to Networking capabilities mapping
type GceNetworkMapper struct {
}

// newGceNetworkMapper initializes the network performance mapper struct
func newGceNetworkMapper() *GceNetworkMapper {
	return &GceNetworkMapper{}
}

// MapNetworkPerf maps the network performance of the gce instance to the category supported by telescopes
func (nm *GceNetworkMapper) MapNetworkPerf(vm productinfo.VmInfo) (string, error) {
	for perfCat, strVals := range ntwPerfMap {
		if productinfo.Contains(strVals, vm.NtwPerf) {
			return perfCat, nil
		}
	}
	return "", fmt.Errorf("could not determine network performance for: [%s]", vm.NtwPerf)
}
