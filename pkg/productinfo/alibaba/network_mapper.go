package alibaba

import (
	"fmt"
	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

var (
	ntwPerfMap = map[string][]string{
		productinfo.NTW_LOW:    {"0.1 Gbit/s", "0.2 Gbit/s", "0.4 Gbit/s", "0.5 Gbit/s", "0.8 Gbit/s", "1.0 Gbit/s", "1.2 Gbit/s", "1.5 Gbit/s", "2.0 Gbit/s"},
		productinfo.NTW_MEDIUM: {"2.5 Gbit/s", "3.0 Gbit/s", "4.0 Gbit/s", "4.5 Gbit/s", "5.0 Gbit/s", "6.0 Gbit/s", "8.0 Gbit/s"},
		productinfo.NTW_HIGH:   {"10.0 Gbit/s"},
		productinfo.NTW_EXTRA:  {"17.0 Gbit/s", "25.0 Gbit/s"},
	}
)

// AlibabaNetworkMapper module object for handling Alibaba Cloud specific VM to Networking capabilities mapping
type AlibabaNetworkMapper struct {
}

// newAlibabaNetworkMapper initializes the network performance mapper struct
func newAlibabaNetworkMapper() *AlibabaNetworkMapper {
	return &AlibabaNetworkMapper{}
}

// MapNetworkPerf maps the network performance of the alibaba instance to the category supported by telescopes
func (nm *AlibabaNetworkMapper) MapNetworkPerf(vm productinfo.VmInfo) (string, error) {
	for perfCat, strVals := range ntwPerfMap {
		if productinfo.Contains(strVals, vm.NtwPerf) {
			return perfCat, nil
		}
	}
	return "", fmt.Errorf("could not determine network performance for: [%s]", vm.NtwPerf)
}
