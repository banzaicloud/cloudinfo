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
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

// CloudInfoer lists operations for retrieving cloud provider information
// Implementers are expected to know the cloud provider specific logic (eg.: cloud provider client usage etc ...)
// This interface abstracts the cloud provider specifics to its clients
type CloudInfoer interface {
	// Initialize is called once per product info renewals so it can be used to download a large price descriptor
	Initialize() (map[string]map[string]types.Price, error)

	// GetVirtualMachines retrieves the available virtual machines in a region
	GetVirtualMachines(region string) ([]types.VMInfo, error)

	// GetProducts gets product information based on the given arguments from an external system
	GetProducts(vms []types.VMInfo, service, regionId string) ([]types.VMInfo, error)

	// GetZones returns the availability zones in a region
	GetZones(region string) ([]string, error)

	// GetRegions retrieves the available regions form the external system
	GetRegions(service string) (map[string]string, error)

	// HasShortLivedPriceInfo signals if a product info provider has frequently changing price info
	HasShortLivedPriceInfo() bool

	// GetCurrentPrices retrieves all the spot prices in a region
	GetCurrentPrices(region string) (map[string]types.Price, error)

	// HasImages signals if a product info provider has image support
	HasImages() bool

	// GetServiceImages retrieves the images supported by the given service in the given region
	GetServiceImages(service, region string) ([]types.Image, error)

	// GetVersions retrieves the  versions supported by the given service in the given region
	GetVersions(service, region string) ([]types.LocationVersion, error)

	// GetServiceProducts retrieves the products supported by the given service in the given region
	GetServiceProducts(region, service string) ([]types.ProductDetails, error)
}
