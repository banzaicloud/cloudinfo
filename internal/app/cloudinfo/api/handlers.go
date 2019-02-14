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
	"errors"
	"net/http"

	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
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
//       200: ProvidersResponse
func (r *RouteHandler) getProviders(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		providers := r.prod.GetProviders(ctxLog)
		if len(providers) < 1 {
			r.errorResponder.Respond(c, emperror.With(errors.New("no providers are configured")))
			return
		}

		c.JSON(http.StatusOK, ProvidersResponse{Providers: providers})
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
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		provider, err := r.prod.GetProvider(ctxLog, pathParams.Provider)
		if err != nil {
			// todo this code is unreachable, the validation catches the possible problems
			r.errorResponder.Respond(c, emperror.With(err))
			return

		}

		c.JSON(http.StatusOK, ProviderResponse{Provider: provider})
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
func (r *RouteHandler) getServices(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetProviderPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
			return
		}

		infoer, err := r.prod.GetInfoer(ctx, pathParams.Provider)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrap(err, "could not retrieve cloud info provider"))
			return
		}

		services, err := infoer.GetServices()
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"could not retrieve services for provider: %s", pathParams.Provider))
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
func (r *RouteHandler) getService(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		// bind the path parameters
		pathParams := GetServicesPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
			return
		}

		infoer, err := r.prod.GetInfoer(ctx, pathParams.Provider)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrap(err, "could not retrieve cloud info provider"))
			return
		}

		service, err := infoer.GetService(pathParams.Service)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"could not retrieve service [%s] for cloud info provider [%s]", pathParams.Service, pathParams.Provider))
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
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		regions, err := r.prod.GetRegions(ctxLog, pathParams.Provider, pathParams.Service)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err, "failed to retrieve regions for provider [%s], service [%s]",
				pathParams.Provider, pathParams.Service))
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
//       200: GetRegionResp
func (r *RouteHandler) getRegion(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetRegionPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
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
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"failed to retrieve regions. provider [%s], service [%s]", pathParams.Provider, pathParams.Service))
			return
		}
		zones, err := r.prod.GetZones(ctxLog, pathParams.Provider, pathParams.Region)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"failed to retrieve zones. provider [%s], service [%s], region [%s]", pathParams.Provider, pathParams.Service, pathParams.Region))

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
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
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
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"failed to retrieve status. provider [%s]", pathParams.Provider))
			return
		}
		details, err := r.prod.GetProductDetails(ctxLog, pathParams.Provider, pathParams.Service, pathParams.Region)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"failed to retrieve product details. service [%s], provider [%s], region [%s]", pathParams.Service,
				pathParams.Service, pathParams.Region))
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
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}
		queryParams := GetImagesQueryParams{}
		if err := mapstructure.Decode(getQueryParamMap(c, "gpu", "version"), &queryParams); err != nil {
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
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
			r.errorResponder.Respond(c, emperror.Wrapf(err, "failed to retrieve service images details. "+
				"provider [%s], service [%s], region [%s]", pathParams.Provider, pathParams.Service, pathParams.Region))
			return

		}

		if queryParams.Gpu != "" && queryParams.Version != "" {
			filteredImages := make([]string, 0)
			for _, image := range images {
				if queryParams.Version == image.VersionName() {
					if queryParams.Gpu == "0" && !image.GpuAvailability() {
						filteredImages = append(filteredImages, image.ImageName())
					} else if queryParams.Gpu != "0" && image.GpuAvailability() {
						filteredImages = append(filteredImages, image.ImageName())
					}
				}
			}
			log.Debug("successfully retrieved image details")
			c.JSON(http.StatusOK, filteredImages)
		} else {
			log.Debug("successfully retrieved image details")
			c.JSON(http.StatusOK, images)
		}
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
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
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
			r.errorResponder.Respond(c, emperror.Wrapf(err, "failed to retrieve versions. provider [%s], "+
				"service [%s], region [%s]", pathParams.Provider, pathParams.Service, pathParams.Region))
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
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
			return
		}

		ctxLog := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(pathParams.Provider).
			WithService(pathParams.Service).
			WithRegion(pathParams.Region).
			WithCorrelationId(logger.GetCorrelationId(c)).
			Build())

		log := logger.Extract(ctxLog)
		log.Info("retrieving attribute values...", map[string]interface{}{"attribute": pathParams.Attribute})

		attributes, err := r.prod.GetAttrValues(ctxLog, pathParams.Provider, pathParams.Service, pathParams.Attribute)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err, "failed to retrieve attribute values. provider [%s], "+
				"service [%s], attributes [%s]", pathParams.Provider, pathParams.Service, pathParams.Attribute))
			return
		}
		log.Info("retrieved attribute values...", map[string]interface{}{"attribute": pathParams.Attribute})

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

// getQueryParamMap transforms the query params into a map to be able to easily bind to param structs
func getQueryParamMap(c *gin.Context, queries ...string) map[string]string {
	queriesMap := make(map[string]string, 0)
	for _, query := range queries {
		if value, ok := c.GetQuery(query); ok {
			queriesMap[query] = value
		}
	}
	return queriesMap
}
