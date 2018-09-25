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

package api

import (
	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

const (
	providerParam  = "provider"
	attributeParam = "attribute"
)

// GetProviderPathParams is a placeholder for the providers related route path parameters
// swagger:parameters getServices getProvider
type GetProviderPathParams struct {
	// in:path
	Provider string `json:"provider"`
}

// GetServicesPathParams is a placeholder for the services related route path parameters
// swagger:parameters getRegions getService
type GetServicesPathParams struct {
	GetProviderPathParams `mapstructure:",squash"`
	// in:path
	Service string `binding:"service" json:"service"`
}

// GetRegionPathParams is a placeholder for the regions related route path parameters
// swagger:parameters getRegion getImages getProducts
type GetRegionPathParams struct {
	GetServicesPathParams `mapstructure:",squash"`
	// in:path
	Region string `binding:"region" json:"region"`
}

// GetAttributeValuesPathParams is a placeholder for the get attribute values route's path parameters
// swagger:parameters getAttrValues
type GetAttributeValuesPathParams struct {
	GetRegionPathParams `mapstructure:",squash"`
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

// ProviderResponse is the response used for the requested provider
// swagger:model ProviderResponse
type ProviderResponse struct {
	Provider productinfo.Provider `json:"provider"`
}

// ProvidersResponse is the response used for the supported providers
// swagger:model ProvidersResponse
type ProvidersResponse struct {
	Providers []productinfo.Provider `json:"providers"`
}

// ServicesResponse holds the list of available services
// swagger:model ServicesResponse
type ServicesResponse struct {
	Services []productinfo.Service `json:"services"`
}

// ServiceResponse holds the list of available services
// swagger:model ServiceResponse
type ServiceResponse struct {
	Service productinfo.Service `json:"service"`
}

// ImagesResponse holds the list of available images
// swagger:model ImagesResponse
type ImagesResponse struct {
	Images []productinfo.Image `json:"images"`
}

// ErrorResponse struct for error responses
// // swagger:model ErrorResponse
type ErrorResponse struct {
	ErrorCode    string `json:"code,omitempty"`
	ErrorMessage string `json:"message,omitempty"`
}

// NewErrorResponse creates a new ERrorResponse struct
func NewErrorResponse(code, message string) ErrorResponse {
	return ErrorResponse{
		ErrorCode:    code,
		ErrorMessage: message,
	}
}

// NewServiceResponse assembles a service response
func NewServiceResponse(sd productinfo.ServiceDescriber) ServiceResponse {
	return ServiceResponse{
		Service: productinfo.NewService(sd.ServiceName()),
	}
}

// NewServicesResponse assembles a new services response
func NewServicesResponse(sds []productinfo.ServiceDescriber) ServicesResponse {
	var services []productinfo.Service
	for _, sd := range sds {
		services = append(services, productinfo.NewService(sd.ServiceName()))
	}
	return ServicesResponse{
		Services: services,
	}
}
