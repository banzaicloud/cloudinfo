package ec2

import (
	"fmt"

	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

var (
	ntwPerfMap = map[string][]string{
		// available categories
		//"10 Gigabit"
		//"20 Gigabit"
		//"25 Gigabit"
		//"High"
		//"Low to Moderate"
		//"Low"
		//"Moderate"
		//"NA"
		//"Up to 10 Gigabit"
		//"Very Low"

		productinfo.NTW_LOW:    {"Very Low", "Low", "Low to Moderate"},
		productinfo.NTW_MEDIUM: {"Moderate", "High"},
		productinfo.NTW_HIGH:   {"Up to 10 Gigabit", "10 Gigabit"},
		productinfo.NTW_EXTRA:  {"20 Gigabit", "25 Gigabit"},
	}
)

// Ec2NetworkMapper module object for handling amazon specific VM to Networking capabilities mapping
type Ec2NetworkMapper struct {
}

// newEc2NetworkMapper initializes the network performance mapper struct
func newEc2NetworkMapper() Ec2NetworkMapper {
	return Ec2NetworkMapper{}
}

// MapNetworkPerf maps the network performance of the ec2 to the category supported ny telescope
func (nm *Ec2NetworkMapper) MapNetworkPerf(vm productinfo.VmInfo) (string, error) {
	for perfCat, strVals := range ntwPerfMap {
		if productinfo.Contains(strVals, vm.NtwPerf) {
			return perfCat, nil
		}
	}
	return "", fmt.Errorf("could not determine network performance for: [%s]", vm.NtwPerf)
}
