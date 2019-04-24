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
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/mitchellh/mapstructure"

	"github.com/banzaicloud/cloudinfo/internal/platform/log"
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
func (r *RouteHandler) getProviders() gin.HandlerFunc {
	return func(c *gin.Context) {

		logger := log.WithFieldsForHandlers(c, r.log, nil)

		logger.Info("getting providers")

		providers, err := r.prod.GetProviders()
		if err != nil {
			r.errorResponder.Respond(c, err)
		}
		if len(providers) < 1 {
			r.errorResponder.Respond(c, errors.New("no providers are configured"))
			return
		}

		logger.Debug("successfully retrieved providers")
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
func (r *RouteHandler) getProvider() gin.HandlerFunc {
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

		logger := log.WithFieldsForHandlers(c, r.log, map[string]interface{}{"provider": pathParams.Provider})

		logger.Info("getting provider details")

		provider, err := r.prod.GetProvider(pathParams.Provider)
		if err != nil {
			// todo this code is unreachable, the validation catches the possible problems
			r.errorResponder.Respond(c, err)
			return

		}

		logger.Debug("successfully retrieved provider details")
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
func (r *RouteHandler) getServices() gin.HandlerFunc {
	return func(c *gin.Context) {
		pathParams := GetProviderPathParams{}
		if err := mapstructure.Decode(getPathParamMap(c), &pathParams); err != nil {
			r.errorResponder.Respond(c, emperror.With(err, "validation"))
			return
		}

		logger := log.WithFieldsForHandlers(c, r.log, map[string]interface{}{"provider": pathParams.Provider})

		logger.Info("getting services")

		if ve := ValidatePathData(pathParams); ve != nil {
			r.errorResponder.Respond(c, emperror.With(ve, "validation"))
			return
		}

		services, err := r.prod.GetServices(pathParams.Provider)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"could not retrieve services for provider: %s", pathParams.Provider))
			return
		}

		logger.Debug("successfully retrieved services")
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
func (r *RouteHandler) getService() gin.HandlerFunc {
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

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service})

		logger.Info("getting service details")

		services, err := r.prod.GetServices(pathParams.Provider)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"could not retrieve services for provider: %s", pathParams.Provider))
			return
		}

		for _, service := range services {
			if service.ServiceName() == pathParams.Service {

				logger.Debug("successfully retrieved service details")
				c.JSON(http.StatusOK, NewServiceResponse(service))
				return
			}
		}

		r.errorResponder.Respond(c, emperror.With(errors.New("could not retrieve service"),
			"service", pathParams.Service, "provider", pathParams.Provider))
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
func (r *RouteHandler) getRegions() gin.HandlerFunc {
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

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service})

		logger.Info("getting regions")

		locations, err := r.prod.GetRegions(pathParams.Provider, pathParams.Service)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err, "failed to retrieve regions for provider [%s], service [%s]",
				pathParams.Provider, pathParams.Service))
			return
		}

		var response RegionsResponse
		for continent, regions := range locations {
			response = append(response, Continent{
				Name:    continent,
				Regions: regions,
			})
		}

		logger.Debug("successfully retrieved regions")
		c.JSON(http.StatusOK, response)
	}
}

// swagger:route GET /providers/{provider}/services/{service}/regions/{region} region getRegion
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
func (r *RouteHandler) getRegion() gin.HandlerFunc {
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

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service, "region": pathParams.Region})

		logger.Info("getting region details")

		locations, err := r.prod.GetRegions(pathParams.Provider, pathParams.Service)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"failed to retrieve regions. provider [%s], service [%s]", pathParams.Provider, pathParams.Service))
			return
		}
		zones, err := r.prod.GetZones(pathParams.Provider, pathParams.Service, pathParams.Region)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"failed to retrieve zones. provider [%s], service [%s], region [%s]", pathParams.Provider, pathParams.Service, pathParams.Region))

			return
		}
		var displayName string
		for _, regions := range locations {
			for _, r := range regions {
				if r.Id == pathParams.Region {
					displayName = r.Name
				}
			}
		}

		logger.Debug("successfully retrieved region details")
		c.JSON(http.StatusOK, GetRegionResp{pathParams.Region, displayName, zones})
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
func (r *RouteHandler) getProducts() gin.HandlerFunc {
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

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service, "region": pathParams.Region})

		logger.Info("getting product details")

		scrapingTime, err := r.prod.GetStatus(pathParams.Provider)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"failed to retrieve status. provider [%s]", pathParams.Provider))
			return
		}
		details, err := r.prod.GetProductDetails(pathParams.Provider, pathParams.Service, pathParams.Region)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err,
				"failed to retrieve product details. service [%s], provider [%s], region [%s]", pathParams.Service,
				pathParams.Service, pathParams.Region))
			return
		}

		logger.Debug("successfully retrieved product details")
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
func (r *RouteHandler) getImages() gin.HandlerFunc {
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

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service, "region": pathParams.Region})

		logger.Info("getting image details")

		images, err := r.prod.GetServiceImages(pathParams.Provider, pathParams.Service, pathParams.Region)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err, "failed to retrieve service images details. "+
				"provider [%s], service [%s], region [%s]", pathParams.Provider, pathParams.Service, pathParams.Region))
			return

		}

		if queryParams.Gpu != "" && queryParams.Version != "" {
			filteredImages := make([]string, 0)
			for _, image := range images {
				if queryParams.Version == image.Version {
					if queryParams.Gpu == "0" && !image.GpuAvailable {
						filteredImages = append(filteredImages, image.Name)
					} else if queryParams.Gpu != "0" && image.GpuAvailable {
						filteredImages = append(filteredImages, image.Name)
					}
				}
			}
			logger.Debug("successfully retrieved image details")
			c.JSON(http.StatusOK, filteredImages)
		} else {
			logger.Debug("successfully retrieved image details")
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
func (r *RouteHandler) getVersions() gin.HandlerFunc {
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

		logger := log.WithFieldsForHandlers(c, r.log,
			map[string]interface{}{"provider": pathParams.Provider, "service": pathParams.Service, "region": pathParams.Region})

		logger.Info("getting version details")

		versions, err := r.prod.GetVersions(pathParams.Provider, pathParams.Service, pathParams.Region)
		if err != nil {
			r.errorResponder.Respond(c, emperror.Wrapf(err, "failed to retrieve versions. provider [%s], "+
				"service [%s], region [%s]", pathParams.Provider, pathParams.Service, pathParams.Region))
			return
		}

		logger.Debug("successfully retrieved version details")
		c.JSON(http.StatusOK, versions)
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

// entry point to the search API
func (r *RouteHandler) query() gin.HandlerFunc {
	return func(c *gin.Context) {
		r.graphqlHandler.ServeHTTP(c.Writer, c.Request)
	}
}
