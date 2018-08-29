package api

import (
	"net/http"

	log "github.com/sirupsen/logrus"

	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
)

// swagger:route GET /providers providers getProviders
//
// Returns the supported providers
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: ProviderResponse
func (r *RouteHandler) getProviders(c *gin.Context) {

	providers := r.prod.GetProviders()
	if len(providers) < 1 {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "no providers are configured"})
	}
	c.JSON(http.StatusOK, providers)
}

// swagger:route GET /providers/{provider}/services services getServices
//
// Provides a list with the available services for the provider
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
//       503: ErrorResponse
func (r *RouteHandler) getServices(c *gin.Context) {

	pathParams := GetProviderPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	infoer, err := r.prod.GetInfoer(pathParams.Provider)
	if err != nil {
		er := NewErrorResponse(fmt.Sprintf("%d", http.StatusInternalServerError), fmt.Sprintf("error while retrieving services: %v", err))
		c.JSON(http.StatusInternalServerError, er)
		return
	}

	services, err := infoer.GetServices()
	if err != nil {
		er := NewErrorResponse(fmt.Sprintf("%d", http.StatusServiceUnavailable), fmt.Sprintf("error while retrieving service: %v", err))
		c.JSON(http.StatusServiceUnavailable, er)
		return
	}

	c.JSON(http.StatusOK, services)

}

// swagger:route GET /providers/{provider}/services/{service} service getService
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
//       503: ErrorResponse
func (r *RouteHandler) getService(c *gin.Context) {
	// bind the path parameters
	pathParams := GetServicesPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	infoer, err := r.prod.GetInfoer(pathParams.Provider)
	if err != nil {
		er := NewErrorResponse(fmt.Sprintf("%d", http.StatusInternalServerError), fmt.Sprintf("error while retrieving service: %v", err))
		c.JSON(http.StatusInternalServerError, er)
		return
	}

	service, err := infoer.GetService(pathParams.Service)
	if err != nil {
		er := NewErrorResponse(fmt.Sprintf("%d", http.StatusServiceUnavailable), fmt.Sprintf("error while retrieving service: %v", err))
		c.JSON(http.StatusServiceUnavailable, er)
		return
	}

	c.JSON(http.StatusOK, service)

}

// swagger:route GET /providers/{provider}/services/{service}/regions regions getRegions
//
// Provides the list of available regions of a cloud provider
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: RegionsResponse
//
func (r *RouteHandler) getRegions(c *gin.Context) {

	pathParams := GetServicesPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	regions, err := r.prod.GetRegions(pathParams.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}
	var response RegionsResponse
	for id, name := range regions {
		response = append(response, Region{id, name})
	}
	c.JSON(http.StatusOK, response)
}

// swagger:route GET /providers/{provider}/services/{service}/regions/{region} regions getRegion
//
// Provides the detailed info of a specific region of a cloud provider
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: RegionResponse
func (r *RouteHandler) getRegion(c *gin.Context) {
	pathParams := GetRegionPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	regions, err := r.prod.GetRegions(pathParams.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}
	zones, err := r.prod.GetZones(pathParams.Provider, pathParams.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}
	c.JSON(http.StatusOK, GetRegionResp{pathParams.Region, regions[pathParams.Region], zones})
}

// swagger:route GET /providers/{provider}/services/{service}/regions/{region}/products products getProducts
//
// Provides a list of available machine types on a given provider in a specific region.
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
func (r *RouteHandler) getProducts(c *gin.Context) {

	pathParams := GetRegionPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	log.Infof("getting product details for provider: %s, region: %s", pathParams.Provider, pathParams.Region)

	scrapingTime, err := r.prod.GetStatus(pathParams.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}
	details, err := r.prod.GetProductDetails(pathParams.Provider, pathParams.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}

	log.Debugf("successfully retrieved product details:  %s, region: %s", pathParams.Provider, pathParams.Region)
	c.JSON(http.StatusOK, ProductDetailsResponse{details, scrapingTime})

}

// swagger:route GET /providers/{provider}/services/{service}/regions/{region}/images images getImages
//
// Provides a list of available images on a given provider in a specific region for a service.
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
func (r *RouteHandler) getImages(c *gin.Context) {

	pathParams := GetRegionPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	log.Infof("getting product details for provider: %s, region: %s", pathParams.Provider, pathParams.Region)

	scrapingTime, err := r.prod.GetStatus(pathParams.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}
	details, err := r.prod.GetProductDetails(pathParams.Provider, pathParams.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}

	log.Debugf("successfully retrieved product details:  %s, region: %s", pathParams.Provider, pathParams.Region)
	c.JSON(http.StatusOK, ProductDetailsResponse{details, scrapingTime})

}

// swagger:route GET /providers/{provider}/services/{service}/regions/{region}/products/{attribute} attributes getAttrValues
//
// Provides a list of available attribute values in a provider's region.
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
func (r *RouteHandler) getAttrValues(c *gin.Context) {

	pathParams := GetAttributeValuesPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	log.Infof("getting %s attribute values for provider: %s, region: %s", pathParams.Attribute, pathParams.Provider, pathParams.Region)

	attributes, err := r.prod.GetAttrValues(pathParams.Provider, pathParams.Attribute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}
	log.Debugf("successfully retrieved %s attribute values:  %s, region: %s", pathParams.Attribute, pathParams.Provider, pathParams.Region)
	c.JSON(http.StatusOK, AttributeResponse{pathParams.Attribute, attributes})
}

// getPathParamMap transforms the path params into a map to be able to easily bind to param structs
func getPathParamMap(c *gin.Context) map[string]string {
	pm := make(map[string]string)
	for _, p := range c.Params {
		pm[p.Key] = p.Value
	}
	return pm
}
