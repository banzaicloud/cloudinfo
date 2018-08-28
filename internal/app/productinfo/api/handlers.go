package api

import (
	"net/http"

	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
)

// swagger:route GET /providers/{provider}/regions/{region}/services services getServices
//
// Provides a list with the available services for the provider in the given region
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: ProductServiceResponse
func (r *RouteHandler) getServices(c *gin.Context) {

}

// swagger:route GET /providers/{provider}/regions/{region}/services/{service} service getService
//
// Provides service details for the given service on the provider in the given region
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: ProductServiceResponse
func (r *RouteHandler) getService(c *gin.Context) {
	// bind the path parameters
	pathParams := GetServicePathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	infoer, err := r.prod.GetInfoer(pathParams.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service: %s", err.Error()))
		return
	}

	service, err := infoer.GetService(pathParams.Region, pathParams.Service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service: %s", err.Error()))
		return
	}

	c.JSON(http.StatusOK, service)

}

// swagger:route GET /providers/{provider}/regions/{region}/services/{service}/images images getServiceImages
//
// Provides a list with the images available for the given service on the provider in the given region
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: ServiceImageResponse
func (r *RouteHandler) getServiceImages(c *gin.Context) {
	// bind the path parameters
	pathParams := GetServicePathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	infoer, err := r.prod.GetInfoer(pathParams.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service images: %s", err.Error()))
		return
	}

	images, err := infoer.GetServiceImages(pathParams.Region, pathParams.Service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service imaged: %s", err.Error()))
		return
	}

	c.JSON(http.StatusOK, images)

}

// swagger:route GET /providers/{provider}/regions/{region}/services/{service}/products serviceProducts getServiceProducts
//
// Provides a list with the products available for the given service on the provider in the given region
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: ProductDetailsResponse
func (r *RouteHandler) getServiceProducts(c *gin.Context) {
	// bind the path parameters
	pathParams := GetServicePathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	infoer, err := r.prod.GetInfoer(pathParams.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service products: %s", err.Error()))
		return
	}
	products, err := infoer.GetServiceProducts(pathParams.Region, pathParams.Service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service products: %s", err.Error()))
		return
	}

	c.JSON(http.StatusOK, products)

}

// swagger:route GET /providers/{provider}/regions/{region}/services/{service}/products/{attribute} serviceAttributeValues getServiceAttributeValues
//
// Provides a list with the attribute values available for the given service on the provider in the given region
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: AttributeResponse
func (r *RouteHandler) getServiceAttributeValues(c *gin.Context) {

	pathParams := GetServiceAttributeValuesParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)
	infoer, err := r.prod.GetInfoer(pathParams.Provider)

	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service attribute values: %s", err.Error()))
		return
	}
	services, err := infoer.GetServiceAttributes(pathParams.Region, pathParams.Service, pathParams.Attribute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service attribute values: %s", err.Error()))
		return
	}

	c.JSON(http.StatusOK, services)

}

// getPathParamMap transforms the path params into a map to be able to easily bind to param structs
func getPathParamMap(c *gin.Context) map[string]string {
	pm := make(map[string]string)
	for _, p := range c.Params {
		pm[p.Key] = p.Value
	}
	return pm
}
