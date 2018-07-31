package gce

import (
	"fmt"

	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

var (
	ntwPerfMap = map[string][]string{
		productinfo.NTW_LOW:    {"1 Gbit/s", "2 Gbit/s"},
		productinfo.NTW_MEDIUM: {"4 Gbit/s", "6 Gbit/s", "8 Gbit/s"},
		productinfo.NTW_HIGH:   {"10 Gbit/s", "12 Gbit/s", "14 Gbit/s"},
		productinfo.NTW_EXTRA:  {"16 Gbit/s"},
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
