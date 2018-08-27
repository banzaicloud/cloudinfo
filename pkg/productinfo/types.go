package productinfo

import (
	"context"
	"time"
)

const (
	// Memory represents the memory attribute for the recommender
	Memory = "memory"

	// Cpu represents the cpu attribute for the recommender
	Cpu = "cpu"

	// VmKeyTemplate format for generating vm cache keys
	VmKeyTemplate = "/banzaicloud.com/recommender/%s/%s/vms"

	// AttrKeyTemplate format for generating attribute cache keys
	AttrKeyTemplate = "/banzaicloud.com/recommender/%s/attrValues/%s"

	// PriceKeyTemplate format for generating price cache keys
	PriceKeyTemplate = "/banzaicloud.com/recommender/%s/%s/prices/%s"

	// ZoneKeyTemplate format for generating zone cache keys
	ZoneKeyTemplate = "/banzaicloud.com/recommender/%s/%s/zones/"

	// RegionKeyTemplate format for generating region cache keys
	RegionKeyTemplate = "/banzaicloud.com/recommender/%s/regions/"

	// StatusKeyTemplate format for generating status cache keys
	StatusKeyTemplate = "/banzaicloud.com/recommender/%s/status/"
)

// ProductInfoer lists operations for retrieving cloud provider information
// Implementers are expected to know the cloud provider specific logic (eg.: cloud provider client usage etc ...)
// This interface abstracts the cloud provider specifics to its clients
type ProductInfoer interface {
	// Initialize is called once per product info renewals so it can be used to download a large price descriptor
	Initialize() (map[string]map[string]Price, error)

	// GetAttributeValues gets the attribute values for the given attribute from the external system
	GetAttributeValues(attribute string) (AttrValues, error)

	// GetProducts gets product information based on the given arguments from an external system
	GetProducts(regionId string) ([]VmInfo, error)

	// GetZones returns the availability zones in a region
	GetZones(region string) ([]string, error)

	// GetRegions retrieves the available regions form the external system
	GetRegions() (map[string]string, error)

	// HasShortLivedPriceInfo signals if a product info provider has frequently changing price info
	HasShortLivedPriceInfo() bool

	// GetCurrentPrices retrieves all the spot prices in a region
	GetCurrentPrices(region string) (map[string]Price, error)

	// GetMemoryAttrName returns the provider representation of the memory attribute
	GetMemoryAttrName() string

	// GetCpuAttrName returns the provider representation of the cpu attribute
	GetCpuAttrName() string

	// GetNetworkPerformanceMapper returns the provider specific network performance mapper
	GetNetworkPerformanceMapper() (NetworkPerfMapper, error)

	// GetServices returns the available services on the  given region
	GetServices(region string) ([]ProductService, error)

	// GetServiceImages retrieves the images supported by the given service in the given region
	GetServiceImages(region, service string) ([]ImageDescriber, error)

	// GetServiceProducts retrieves the products supported by the given service in the given region
	GetServiceProducts(region, service string) ([]ProductDetails, error)

	// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
	GetServiceAttributes(region, service, attribute string) (AttrValues, error)
}

// ProductInfo is the main entry point for retrieving vm type characteristics and pricing information on different cloud providers
// todo this interface should be reduced not to contain methods defined in the Productinfoer interface;
// todo it's enough to get the provider specific infoer implementation and delegate to that ...
type ProductInfo interface {
	// GetProviders returns the supported providers
	GetProviders() []string

	// Start starts the product information retrieval in a new goroutine
	Start(ctx context.Context)

	// Initialize is called once per product info renewals so it can be used to download a large price descriptor
	Initialize(provider string) (map[string]map[string]Price, error)

	// GetAttributes returns the supported attribute names
	GetAttributes() []string

	// GetAttrValues returns a slice with the possible values for a given attribute on a specific provider
	GetAttrValues(provider string, attribute string) ([]float64, error)

	// GetZones returns all the availability zones for a region
	GetZones(provider string, region string) ([]string, error)

	// GetRegions returns all the regions for a cloud provider
	GetRegions(provider string) (map[string]string, error)

	// HasShortLivedPriceInfo signals if a product info provider has frequently changing price info
	HasShortLivedPriceInfo(provider string) bool

	// GetPrice returns the on demand price and the zone averaged computed spot price for a given instance type in a given region
	GetPrice(provider string, region string, instanceType string, zones []string) (float64, float64, error)

	// GetNetworkPerfMapper retrieves the network performance mapper implementation
	GetNetworkPerfMapper(provider string) (NetworkPerfMapper, error)

	// GetInfoer gets the cloud provider specific Infoer implementation (discriminator for cloud providers)
	GetInfoer(provider string) (ProductInfoer, error)
}

// AttrValue represents an attribute value
type AttrValue struct {
	StrValue string
	Value    float64
}

// AttrValues a slice of AttrValues
type AttrValues []AttrValue

var (
	// telescope supported network performance of vm-s

	// NTW_LOW the low network performance category
	NTW_LOW = "low"
	// NTW_MEDIUM the medium network performance category
	NTW_MEDIUM = "medium"
	// NTW_HIGH the high network performance category
	NTW_HIGH = "high"
	// NTW_EXTRA the highest network performance category
	NTW_EXTRA = "extra"
)

// NetworkPerfMapper operations related  to mapping between virtual machines to network performance categories
type NetworkPerfMapper interface {
	// MapNetworkPerf gets the network performance category for the given
	MapNetworkPerf(vm VmInfo) (string, error)
}

// ProductStorer interface collects the necessary cache operations
type ProductStorer interface {
	Get(k string) (interface{}, bool)
	Set(k string, x interface{}, d time.Duration)
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

// ProductDetails extended view of the virtual machine details
type ProductDetails struct {
	// Embedded struct!
	VmInfo

	// Burst this is derived for now
	Burst bool `json:"burst,omitempty"`

	// ZonePrice holds spot price information per zone
	SpotInfo []ZonePrice `json:"spotPrice,omitempty"`
}

// ProductDetailSource product details related set of operations
type ProductDetailSource interface {
	// GetProductDetails gathers the product details information known by telescope
	GetProductDetails(cloud string, region string) ([]ProductDetails, error)
}

// newProductDetails creates a new ProductDetails struct and returns a pointer to it
func newProductDetails(vm VmInfo) *ProductDetails {
	pd := ProductDetails{}
	pd.VmInfo = vm
	pd.Burst = vm.IsBurst()
	return &pd
}

// ProductService represents a service; eg.: oke, eks
type ProductService interface {
	// GetName abstracts the name assembly for the service
	GetName() string

	// GetResources returns a slice with the resources available for the given service
	GetResources() []string // todo is this required?
}

// ImageDescriber is a placeholder interface for image information
// to be extended with other operations if needed
type ImageDescriber interface {
	// ImageName returns the image name
	ImageName() string
}
