package api

import (
	"net/http"

	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"
)

// swagger:route GET /providers/{provider}/services/{service} services getServices
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

// getPathParamMap transforms the path params into a map to be able to easily bind to param structs
func getPathParamMap(c *gin.Context) map[string]string {
	pm := make(map[string]string)
	for _, p := range c.Params {
		pm[p.Key] = p.Value
	}
	return pm
}
