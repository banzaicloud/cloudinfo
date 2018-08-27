package api

import (
	"net/http"

	"fmt"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// getServices retrieves services supported by the gibven provider in the given region
func (r *RouteHandler) getServices(c *gin.Context) {

	pathParams := getPathParams(c)
	log.Debug("Path params: [%s]", pathParams)

	var err error
	if infoer, err := r.prod.GetInfoer(pathParams.Provider); err == nil {
		if services, err := infoer.GetServices(pathParams.Region); err == nil {
			c.JSON(http.StatusOK, services)
			return
		}
	}

	c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving services: %s", err.Error()))

}

func (r *RouteHandler) getService(c *gin.Context) {

	pathParams := getPathParams(c)
	log.Debug("Path params: [%s]", pathParams)

	c.JSON(http.StatusInternalServerError, "not yet implemented")

}

func (r *RouteHandler) getServiceImages(c *gin.Context) {

	pathParams := getPathParams(c)
	log.Debug("Path params: [%s]", pathParams)

	var err error
	if infoer, err := r.prod.GetInfoer(pathParams.Provider); err == nil {
		if services, err := infoer.GetServiceImages(pathParams.Region, pathParams.Service); err == nil {
			c.JSON(http.StatusOK, services)
			return
		}
	}

	c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service images: %s", err.Error()))

}

func (r *RouteHandler) getServiceProducts(c *gin.Context) {

	pathParams := getPathParams(c)
	log.Debug("Path params: [%s]", pathParams)

	var err error
	if infoer, err := r.prod.GetInfoer(pathParams.Provider); err == nil {
		if services, err := infoer.GetServiceProducts(pathParams.Region, pathParams.Service); err == nil {
			c.JSON(http.StatusOK, services)
			return
		}
	}

	c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service products: %s", err.Error()))

}

func (r *RouteHandler) getServiceAttributes(c *gin.Context) {
	pathParams := getPathParams(c)
	log.Debug("Path params: [%s]", pathParams)

	var err error
	if infoer, err := r.prod.GetInfoer(pathParams.Provider); err == nil {
		if services, err := infoer.GetServiceAttributes(pathParams.Region, pathParams.Service, pathParams.Attribute); err == nil {
			c.JSON(http.StatusOK, services)
			return
		}
	}

	c.JSON(http.StatusInternalServerError, fmt.Sprintf("error while retrieving service attribute values: %s", err.Error()))

}

// getPathParam handles path params retrieval in case tha path param is not set a warning is logged
func getPathParam(pathParam string, c *gin.Context) string {
	param := c.Param(pathParam)
	if param == "" {
		log.Warnf("path param %s is not set", pathParam)
	}
	return param
}

// getPathParams parses the path information from the gin Contexts
func getPathParams(c *gin.Context) ServicePathParams {
	provider := getPathParam(providerParam, c)
	region := getPathParam(regionParam, c)
	service := getPathParam(serviceParam, c)
	attribute := getPathParam(attributeParam, c)
	return newPathParams(provider, region, service, attribute)
}
