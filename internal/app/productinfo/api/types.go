package api

import (
	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

const (
	providerParam  = "provider"
	regionParam    = "region"
	attributeParam = "attribute"
	serviceParam   = "service"
)

// GetProviderPathParams is a placeholder for the providers related route path parameters
// swagger:parameters getRegions
type GetProviderPathParams struct {
	// in:path
	Provider string `json:"provider"`
}

// GetRegionPathParams is a placeholder for the regions related route path parameters
// swagger:parameters getRegion getServices getProductDetails
type GetRegionPathParams struct {
	GetProviderPathParams `mapstructure:",squash"`
	// in:path
	Region string `json:"region"`
}

// GetAttributeValuesPathParams is a placeholder for the get attribute values route's path parameters
// swagger:parameters getAttributeValues
type GetAttributeValuesPathParams struct {
	GetRegionPathParams `mapstructure:",squash"`
	// in:path
	Attribute string `json:"attribute"`
}

// GetServicePathParams is a placeholder for the services related route path parameters
// swagger:parameters getService getServiceImages getServiceProducts
type GetServicePathParams struct {
	GetRegionPathParams `mapstructure:",squash"`
	// in:path
	Service string `json:"service"`
}

// GetServiceAttributeValuesParams is a placeholder for the get service attribute values route's path parameters
// swagger:parameters getServiceAttributeValues
type GetServiceAttributeValuesParams struct {
	GetServicePathParams `mapstructure:",squash"`
	// in:path
	Attribute string `json:"attribute"`
}

// ProductDetailsResponse Api object to be mapped to product info response
// swagger:model ProductDetailsResponse
type ProductDetailsResponse struct {
	// Products represents a slice of products for a given provider (VMs with attributes and process)
	Products []productinfo.ProductDetails `json:"products"`
	// ScrapingTime represents scraping time for a given provider in milliseconds
	ScrapingTime string `json:"scrapingTime"`
}

// RegionsResponse holds the list of available regions of a cloud provider
// swagger:model RegionsResponse
type RegionsResponse []Region

// Region hold the id and name of a cloud provider region
type Region struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

// GetRegionResp holds the detailed description of a specific region of a cloud provider
// swagger:model RegionResponse
type GetRegionResp struct {
	Id    string   `json:"id"`
	Name  string   `json:"name"`
	Zones []string `json:"zones"`
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

// ProductServiceResponse holds the list of available services
// swagger:model ProductServiceResponse
type ProductServiceResponse []productinfo.ServiceDescriber

// ServiceImageResponse holds the list of available images
// swagger:model ServiceImageResponse
type ServiceImageResponse struct {
	Services productinfo.ServiceDescriber
}
