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

package cloudinfo

const (
	// Memory represents the memory attribute for the product info
	Memory = "memory"

	// Cpu represents the cpu attribute for the product info
	Cpu = "cpu"
)

// CloudInfo is the main entry point for retrieving vm type characteristics and pricing information on different cloud providers
type CloudInfo interface {
	// GetProviders returns the supported providers
	GetProviders() ([]Provider, error)

	// GetProvider retrieves information about the provider
	GetProvider(provider string) (Provider, error)

	// GetServices returns the supported services for a provider
	GetServices(provider string) ([]Service, error)

	// GetZones returns all the availability zones for a region
	GetZones(provider, service, region string) ([]string, error)

	// GetRegions returns all the regions for a cloud provider
	GetRegions(provider string, service string) (map[string]string, error)

	GetStatus(provider string) (string, error)

	GetProductDetails(provider, service, region string) ([]ProductDetails, error)

	GetServiceImages(provider, service, region string) ([]Image, error)

	GetVersions(provider, service, region string) ([]LocationVersion, error)

	GetContinentsData(provider, service string) (map[string][]Region, error)

	GetContinents() []string
}

const (
	// telescope supported network performance of vm-s

	// NtwLow the low network performance category
	NtwLow = "low"
	// NtwMedium the medium network performance category
	NtwMedium = "medium"
	// NtwHight the high network performance category
	NtwHight = "high"
	// NtwExtra the highest network performance category
	NtwExtra = "extra"

	// Telescopes supports categories of virtual machines
	CategoryGeneral = "General purpose"
	CategoryCompute = "Compute optimized"
	CategoryMemory  = "Memory optimized"
	CategoryGpu     = "GPU instance"
	CategoryStorage = "Storage optimized"

	ContinentNorthAmerica = "North America"
	ContinentSouthAmerica = "South America"
	ContinentEurope       = "Europe"
	ContinentAfrica       = "Africa"
	ContinentAsia         = "Asia"
	ContinentAustralia    = "Australia"
)

// NetworkPerfMapper operations related  to mapping between virtual machines to network performance categories
type NetworkPerfMapper interface {
	// MapNetworkPerf gets the network performance category for the given
	MapNetworkPerf(ntwPerf string) (string, error)
}

// ZonePrice struct for displaying price information per zone
type ZonePrice struct {
	Zone  string  `json:"zone"`
	Price float64 `json:"price"`
}

// newZonePrice creates a new zone price struct and returns its pointer
func newZonePrice(zone string, price float64) *ZonePrice {
	return &ZonePrice{
		Zone:  zone,
		Price: price,
	}
}

// LocationVersion struct for displaying version information per location
type LocationVersion struct {
	Location string   `json:"location"`
	Versions []string `json:"versions"`
	Default  string   `json:"default"`
}

// NewLocationVersion creates a new location version struct
func NewLocationVersion(location string, versions []string, def string) LocationVersion {
	if len(versions) > 0 && def == "" {
		def = versions[0]
	}
	return LocationVersion{
		Location: location,
		Versions: versions,
		Default:  def,
	}
}

// ProductDetails extended view of the virtual machine details
type ProductDetails struct {
	// Embedded struct!
	VmInfo

	// Burst this is derived for now
	Burst bool `json:"burst,omitempty"`
}

// ProductDetailSource product details related set of operations
type ProductDetailSource interface {
	// GetProductDetails gathers the product details information known by telescope
	GetProductDetails(provider string, region string) ([]ProductDetails, error)
}

// newProductDetails creates a new ProductDetails struct and returns a pointer to it
func newProductDetails(vm VmInfo) *ProductDetails {
	pd := ProductDetails{}
	pd.VmInfo = vm
	pd.Burst = vm.IsBurst()
	return &pd
}

// ServiceDescriber represents a service; eg.: oke, eks
// Extend this interface with other operations if needed
type ServiceDescriber interface {
	// ServiceName abstracts the name assembly for the service
	ServiceName() string
}

// Service represents a service supported by a given provider.
// it's intended to implement the ServiceDescriber interface
type Service struct {
	Service  string `json:"service"`
	IsStatic bool   `json:"isStatic"`
}

// ServiceName returns the service name
func (s Service) ServiceName() string {
	return s.Service
}

// ProviderDescriber describes a provider
type ProviderDescriber interface {
	// ProviderName returns the name of the provider
	ProviderName() string
}

// Provider represents a cloud provider
type Provider struct {
	Provider string    `json:"provider"`
	Services []Service `json:"services"`
}

// ProviderName returns the name of the provider
func (p Provider) ProviderName() string {
	return p.Provider
}

// NewProvider create new provider describer struct
func NewProvider(name string) Provider {
	return Provider{
		Provider: name,
	}
}

// Image represents an image
type Image struct {
	Name         string `json:"name"`
	Version      string `json:"version,omitempty"`
	GpuAvailable bool   `json:"gpu,omitempty"`
}

// NewImage create new provider describer struct
func NewImage(name, version string, gpu bool) Image {
	return Image{
		Name:         name,
		Version:      version,
		GpuAvailable: gpu,
	}
}

// Version represents a version
type Version struct {
	Version string `json:"versions"`
}

// VersionName returns the name of the version
func (v Version) VersionName() string {
	return v.Version
}

// Region hold the id and name of a cloud provider region
type Region struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}
