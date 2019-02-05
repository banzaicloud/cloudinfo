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
	"net/http"
	"os"

	"github.com/banzaicloud/cloudinfo/internal/platform/buildinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/banzaicloud/go-gin-prometheus"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
)

// RouteHandler configures the REST API routes in the gin router
type RouteHandler struct {
	prod           cloudinfo.CloudInfo
	buildInfo      buildinfo.BuildInfo
	errorResponder Responder
}

// NewRouteHandler creates a new RouteHandler and returns a reference to it
func NewRouteHandler(p cloudinfo.CloudInfo, bi buildinfo.BuildInfo) *RouteHandler {
	return &RouteHandler{
		prod:           p,
		buildInfo:      bi,
		errorResponder: NewErrorResponder(),
	}
}

// ConfigureRoutes configures the gin engine, defines the rest API for this application
func (r *RouteHandler) ConfigureRoutes(ctx context.Context, router *gin.Engine) {
	logger.Extract(ctx).Info("configuring routes")

	basePath := "/"
	if basePathFromEnv := os.Getenv("CLOUDINFO_BASEPATH"); basePathFromEnv != "" {
		basePath = basePathFromEnv
	}

	router.Use(logger.MiddlewareCorrelationId())
	router.Use(logger.Middleware())
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

		providerGroup.GET("/", r.getProviders(ctx))
		providerGroup.GET("/:provider", r.getProvider(ctx))
		providerGroup.GET("/:provider/services", r.getServices(ctx))
		providerGroup.GET("/:provider/services/:service", r.getService(ctx))
		providerGroup.GET("/:provider/services/:service/regions", r.getRegions(ctx))
		providerGroup.GET("/:provider/services/:service/regions/:region", r.getRegion(ctx))
		providerGroup.GET("/:provider/services/:service/regions/:region/images", r.getImages(ctx))
		providerGroup.GET("/:provider/services/:service/regions/:region/versions", r.getVersions(ctx))
		providerGroup.GET("/:provider/services/:service/regions/:region/products", r.getProducts(ctx))
		providerGroup.GET("/:provider/services/:service/regions/:region/products/:attribute", r.getAttrValues(ctx))
	}

}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}

func (r *RouteHandler) versionHandler(c *gin.Context) {
	c.JSON(http.StatusOK, r.buildInfo)
}

func (r *RouteHandler) EnableMetrics(ctx context.Context, router *gin.Engine, metricsAddr string) {
	p := ginprometheus.NewPrometheus("http", []string{"provider", "service", "region"})
	p.SetListenAddress(metricsAddr)
	p.Use(router, "/metrics")
	p.UseWithCustomMetrics(router, metrics.GetPriceGatherers(), "/metrics/price")
	p.UseWithCustomMetrics(router, metrics.GetSpotPriceGatherers(), "/metrics/spotprice")

}
