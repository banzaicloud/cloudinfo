package ec2

import (
	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"github.com/stretchr/testify/assert"
	"testing"
)

// MapperSuit test suite for a network performance mapper implementation
func MapperErrorSuit(t *testing.T, mapper productinfo.NetworkPerfMapper) {
	// todo design meaningful suite(s) for testing mapper implementations
	// todo eg: testing all the categories for a given impl, errors, etc ...
}

func TestEc2NetworkMapper_MapNetworkPerf(t *testing.T) {

	mapper := Ec2NetworkMapper{}
	tests := []struct {
		name  string
		vm    productinfo.VmInfo
		check func(cat string, err error)
	}{
		{
			name: "success - mapper maps to the lowest category",
			vm: productinfo.VmInfo{
				NtwPerf: "Very Low",
			},
			check: func(cat string, err error) {
				assert.Equal(t, productinfo.NTW_LOW, cat, "not mapped to the right category")
			},
		},
		{
			name: "error - mapper doesn't map to a category",
			vm: productinfo.VmInfo{
				NtwPerf: "Error",
			},
			check: func(cat string, err error) {
				assert.Equal(t, "", cat, "not mapped to the right category")
				assert.Equal(t, "could not determine network performance for: [Error]", err.Error())
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(mapper.MapNetworkPerf(test.vm))
		})
	}
}
