package oci

import (
	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

// OCINetworkMapper module object for handling Oracle specific VM to Networking capabilities mapping
type OCINetworkMapper struct {
}

// newNetworkMapper initializes the network performance mapper struct
func newNetworkMapper() *OCINetworkMapper {
	return &OCINetworkMapper{}
}

// MapNetworkPerf maps the network performance of the gce instance to the category supported by telescopes
// Currently it always gives back productinfo.NTW_MEDIUM
func (nm *OCINetworkMapper) MapNetworkPerf(vm productinfo.VmInfo) (string, error) {

	return productinfo.NTW_MEDIUM, nil
}
