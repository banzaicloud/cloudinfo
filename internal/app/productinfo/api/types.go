package api

import (
	"fmt"
	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

const (
	providerParam  = "provider"
	regionParam    = "region"
	attributeParam = "attribute"
	serviceParam   = "service"
)

// GetProductDetailsParams is a placeholder for the get products route's path parameters
// swagger:parameters getProductDetails
type GetProductDetailsParams struct {
	// in:path
	Provider string `json:"provider"`
	// in:path
	Region string `json:"region"`
}

// ProductDetailsResponse Api object to be mapped to product info response
// swagger:model ProductDetailsResponse
type ProductDetailsResponse struct {
	// Products represents a slice of products for a given provider (VMs with attributes and process)
	Products []productinfo.ProductDetails `json:"products"`
	// ScrapingTime represents scraping time for a given provider in milliseconds
	ScrapingTime string `json:"scrapingTime"`
}

// GetRegionsParams is a placeholder for the get regions route's path parameters
// swagger:parameters getRegions
type GetRegionsParams struct {
	// in:path
	Provider string `json:"provider"`
}

// RegionsResponse holds the list of available regions of a cloud provider
// swagger:model RegionsResponse
type RegionsResponse []Region

// Region hold the id and name of a cloud provider region
type Region struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// GetRegionParams is a placeholder for the get region route's path parameters
// swagger:parameters getRegion
type GetRegionParams struct {
	// in:path
	Provider string `json:"provider"`
	// in:path
	Region string `json:"region"`
}

// GetRegionResp holds the detailed description of a specific region of a cloud provider
// swagger:model RegionResponse
type GetRegionResp struct {
	Id    string   `json:"id"`
	Name  string   `json:"name"`
	Zones []string `json:"zones"`
}

// GetAttributeValuesParams is a placeholder for the get attribute values route's path parameters
// swagger:parameters getAttributeValues
type GetAttributeValuesParams struct {
	// in:path
	Provider string `json:"provider"`
	// in:path
	Region string `json:"region"`
	// in:path
	Attribute string `json:"attribute"`
}

// AttributeResponse holds attribute values
// swagger:model AttributeResponse
type AttributeResponse struct {
	AttributeName   string    `json:"attributeName"`
	AttributeValues []float64 `json:"attributeValues"`
}

// ProviderResponse is the response used for the supported providers
// swagger:model ProviderResponse
type ProviderResponse []string

// ServicePathParams represents the information from the path
// Serves as an abstraction of the request path information to be used in handlers
type ServicePathParams struct {
	// Embedding the already defined structg
	GetRegionParams
	// Service represents the service from the request path
	Service string
	// Attribute represents a service resource (cpu, hdd etc ...)
	Attribute string
}

// newPathParams creates a new struct with the provided information
func newPathParams(provider, region, service, attribute string) ServicePathParams {
	rp := GetRegionParams{
		Provider: provider,
		Region:   region,
	}

	return ServicePathParams{
		GetRegionParams: rp,
		Service:         service,
		Attribute:       attribute,
	}
}

func (p *ServicePathParams) String() string {
	return fmt.Sprintf("Provider: %s, Region: %s, Service: %s", p.Provider, p.Region, p.Service)
}
