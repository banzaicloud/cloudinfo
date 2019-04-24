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
	"os"

	ginprometheus "github.com/banzaicloud/go-gin-prometheus"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/goph/logur"

	"github.com/banzaicloud/cloudinfo/internal/platform/buildinfo"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
)

// RouteHandler configures the REST API routes in the gin router
type RouteHandler struct {
	log            logur.Logger
	prod           cloudinfo.CloudInfo
	buildInfo      buildinfo.BuildInfo
	errorResponder Responder
}

// NewRouteHandler creates a new RouteHandler and returns a reference to it
func NewRouteHandler(p cloudinfo.CloudInfo, bi buildinfo.BuildInfo, store cloudinfo.CloudInfoStore, log logur.Logger) *RouteHandler {
	return &RouteHandler{
		prod:           p,
		buildInfo:      bi,
		errorResponder: NewErrorResponder(),
		log:            log,
	}
}

// ConfigureRoutes configures the gin engine, defines the rest API for this application
func (r *RouteHandler) ConfigureRoutes(router *gin.Engine) {
	r.log.Info("configuring routes")

	basePath := "/"
	if basePathFromEnv := os.Getenv("CLOUDINFO_BASEPATH"); basePathFromEnv != "" {
		basePath = basePathFromEnv
	}

	router.Use(log.MiddlewareCorrelationId())
	router.Use(log.Middleware())
	router.Use(cors.Default())
	router.Use(static.Serve(basePath, static.LocalFile("./web/dist/ui", true)))

	base := router.Group(basePath)
	{
		base.GET("/status", r.signalStatus)
		base.GET("/version", r.versionHandler)
	}

	v1 := base.Group("/api/v1")

	providerGroup := v1.Group("/providers")
	{
		providerGroup.GET("/", r.getProviders())
		providerGroup.GET("/:provider", r.getProvider())
		providerGroup.GET("/:provider/services", r.getServices())
		providerGroup.GET("/:provider/services/:service", r.getService())
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
