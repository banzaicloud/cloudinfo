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
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/cloudinfo/pkg/logger"
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
//       200: ProvidersResponse
func (r *RouteHandler) getProviders(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		providers := r.prod.GetProviders(ctxLog)
		if len(providers) < 1 {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "no providers are configured"})
		}

		c.JSON(http.StatusOK, ProvidersResponse{
			Providers: providers,
		})
	}
}

// swagger:route GET /providers/{provider} provider getProvider
//
// Returns the requested provider
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
func (r *RouteHandler) getProvider(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetProviderPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		provider, err := r.prod.GetProvider(ctxLog, pathParams.Provider)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": fmt.Sprintf("%s", err)})
			return

		}

		c.JSON(http.StatusOK, ProviderResponse{
			Provider: provider,
		})
	}
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
//       200: ServicesResponse
//       503: ErrorResponse
func (r *RouteHandler) getServices(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetProviderPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

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

		c.JSON(http.StatusOK, NewServicesResponse(services))
	}
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
//       200: ServiceResponse
//       503: ErrorResponse
func (r *RouteHandler) getService(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// bind the path parameters
		pathParams := GetServicesPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		infoer, err := r.prod.GetInfoer(pathParams.Provider)
		if err != nil {
			er := NewErrorResponse(fmt.Sprintf("%d", http.StatusInternalServerError), fmt.Sprintf("error while retrieving service: %v", err))
			c.JSON(http.StatusInternalServerError, er)
			return
		}

		service, err := infoer.GetService(ctxLog, pathParams.Service)
		if err != nil {
			er := NewErrorResponse(fmt.Sprintf("%d", http.StatusServiceUnavailable), fmt.Sprintf("error while retrieving service: %v", err))
			c.JSON(http.StatusServiceUnavailable, er)
			return
		}

		c.JSON(http.StatusOK, NewServiceResponse(service))
	}
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
func (r *RouteHandler) getRegions(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetServicesPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		regions, err := r.prod.GetRegions(ctxLog, pathParams.Provider, pathParams.Service)
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
func (r *RouteHandler) getRegion(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRegionPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithRegion(pathParams.Region).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		regions, err := r.prod.GetRegions(ctxLog, pathParams.Provider, pathParams.Service)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
			return
		}
		zones, err := r.prod.GetZones(ctxLog, pathParams.Provider, pathParams.Region)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
			return
		}
		c.JSON(http.StatusOK, GetRegionResp{pathParams.Region, regions[pathParams.Region], zones})
	}
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
func (r *RouteHandler) getProducts(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRegionPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithRegion(pathParams.Region).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		log := logger.Extract(ctxLog)
		log.Info("getting product details")

		scrapingTime, err := r.prod.GetStatus(pathParams.Provider)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
			return
		}
		details, err := r.prod.GetProductDetails(ctxLog, pathParams.Provider, pathParams.Service, pathParams.Region)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
			return
		}

		log.Debug("successfully retrieved product details")
		c.JSON(http.StatusOK, ProductDetailsResponse{details, scrapingTime})
	}
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
//       200: ImagesResponse
func (r *RouteHandler) getImages(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRegionPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithRegion(pathParams.Region).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		log := logger.Extract(ctxLog)
		log.Info("getting image details")

		images, err := r.prod.GetServiceImages(ctxLog, pathParams.Provider, pathParams.Service, pathParams.Region)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
			return
		}

		log.Debug("successfully retrieved image details")
		c.JSON(http.StatusOK, images)
	}
}

// swagger:route GET /providers/{provider}/services/{service}/regions/{region}/versions versions getVersions
//
// Provides a list of available versions on a given provider in a specific region for a service.
//
//     Produces:
//     - application/json
//
//     Schemes: http
//
//     Security:
//
//     Responses:
//       200: VersionsResponse
func (r *RouteHandler) getVersions(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRegionPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithRegion(pathParams.Region).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		log := logger.Extract(ctxLog)
		log.Info("getting versions")

		versions, err := r.prod.GetVersions(ctxLog, pathParams.Provider, pathParams.Service, pathParams.Region)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
			return
		}

		log.Debug("successfully retrieved version details")
		c.JSON(http.StatusOK, versions)
	}
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
func (r *RouteHandler) getAttrValues(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetAttributeValuesPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithRegion(pathParams.Region).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		log := logger.Extract(ctxLog)
		log.Infof("getting %s attribute values", pathParams.Attribute)

		attributes, err := r.prod.GetAttrValues(ctxLog, pathParams.Provider, pathParams.Service, pathParams.Attribute)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
			return
		}
		log.Debugf("successfully retrieved %s attribute values", pathParams.Attribute)

		c.JSON(http.StatusOK, AttributeResponse{pathParams.Attribute, attributes})
	}
}

// getPathParamMap transforms the path params into a map to be able to easily bind to param structs
func getPathParamMap(c *gin.Context) map[string]string {
	pm := make(map[string]string)
	for _, p := range c.Params {
		pm[p.Key] = p.Value
	}
	return pm
}
