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
	"net/http"
	"strings"

	ginprometheus "github.com/banzaicloud/go-gin-prometheus"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gobuffalo/packr/v2"
	"github.com/gobuffalo/packr/v2/file"
	"github.com/goph/logur"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
	"github.com/banzaicloud/cloudinfo/internal/platform/buildinfo"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
)

// RouteHandler configures the REST API routes in the gin router
type RouteHandler struct {
	log            logur.Logger
	prod           types.CloudInfo
	buildInfo      buildinfo.BuildInfo
	errorResponder Responder
	graphqlHandler http.Handler
}

// NewRouteHandler creates a new RouteHandler and returns a reference to it
func NewRouteHandler(p types.CloudInfo, bi buildinfo.BuildInfo, graphqlHandler http.Handler, log logur.Logger) *RouteHandler {
	return &RouteHandler{
		prod:           p,
		buildInfo:      bi,
		errorResponder: NewErrorResponder(),
		graphqlHandler: graphqlHandler,
		log:            log,
	}
}

type binaryFileSystem struct {
	fs *packr.Box
}

func (b *binaryFileSystem) Open(name string) (http.File, error) {
	// This is necessary because of https://github.com/gobuffalo/packr/issues/173
	if b.fs.HasDir(name) {
		return file.NewDir(name)
	}

	return b.fs.Open(name)
}

func (b *binaryFileSystem) Exists(prefix string, filepath string) bool {
	if p := strings.TrimPrefix(filepath, prefix); len(p) < len(filepath) {
		return b.fs.Has(p)
	}

	return false
}

// ConfigureRoutes configures the gin engine, defines the rest API for this application
func (r *RouteHandler) ConfigureRoutes(router *gin.Engine, basePath string, box *packr.Box) {
	r.log.Info("configuring routes")

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Banzai-Cloud-Pipeline-UUID")

	router.Use(log.MiddlewareCorrelationId())
	router.Use(log.Middleware())
	router.Use(cors.New(corsConfig))
	router.Use(static.Serve(basePath, &binaryFileSystem{fs: box}))

	base := router.Group(basePath)
	{
		base.GET("/status", r.signalStatus)
		base.GET("/version", r.versionHandler)
	}

	v1 := base.Group("/api/v1")

	v1.GET("/continents", r.getContinents())

	providerGroup := v1.Group("/providers")
	{
		providerGroup.GET("/", r.getProviders())
		providerGroup.GET("/:provider", r.getProvider())
		providerGroup.GET("/:provider/services", r.getServices())
		providerGroup.GET("/:provider/services/:service", r.getService())
		providerGroup.GET("/:provider/services/:service/continents", r.getContinentsData())
		providerGroup.GET("/:provider/services/:service/regions", r.getRegions())
		providerGroup.GET("/:provider/services/:service/regions/:region", r.getRegion())
		providerGroup.GET("/:provider/services/:service/regions/:region/images", r.getImages())
		providerGroup.GET("/:provider/services/:service/regions/:region/versions", r.getVersions())
		providerGroup.GET("/:provider/services/:service/regions/:region/products", r.getProducts())
	}

	base.POST("/graphql", r.query())
}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}

func (r *RouteHandler) versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, r.buildInfo)
}

func (r *RouteHandler) EnableMetrics(router *gin.Engine, metricsAddr string) {
	p := ginprometheus.NewPrometheus("http", []string{"provider", "service", "region"})
	p.SetListenAddress(metricsAddr)
	p.Use(router, "/metrics")
	p.UseWithCustomMetrics(router, metrics.GetPriceGatherers(), "/metrics/price")
	p.UseWithCustomMetrics(router, metrics.GetSpotPriceGatherers(), "/metrics/spotprice")

}
