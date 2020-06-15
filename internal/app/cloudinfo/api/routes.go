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
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	ginprometheus "github.com/banzaicloud/go-gin-prometheus"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/markbates/pkger"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
	"github.com/banzaicloud/cloudinfo/internal/platform/buildinfo"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
)

// RouteHandler configures the REST API routes in the gin router
type RouteHandler struct {
	log            cloudinfo.Logger
	prod           types.CloudInfo
	buildInfo      buildinfo.BuildInfo
	errorResponder Responder
	graphqlHandler http.Handler
}

// NewRouteHandler creates a new RouteHandler and returns a reference to it
func NewRouteHandler(p types.CloudInfo, bi buildinfo.BuildInfo, graphqlHandler http.Handler, log cloudinfo.Logger) *RouteHandler {
	return &RouteHandler{
		prod:           p,
		buildInfo:      bi,
		errorResponder: NewErrorResponder(),
		graphqlHandler: graphqlHandler,
		log:            log,
	}
}

// ConfigureRoutes configures the gin engine, defines the rest API for this application
func (r *RouteHandler) ConfigureRoutes(router *gin.Engine, basePath string) {
	r.log.Info("configuring routes")

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Banzai-Cloud-Pipeline-UUID")

	router.Use(log.MiddlewareCorrelationId())
	router.Use(log.Middleware())
	router.Use(cors.New(corsConfig))

	dir := pkger.Dir("/web/dist/web")
	router.Use(static.Serve(basePath, pkgerFileSystem{dir}))

	base := router.Group(basePath)

	{
		indexFile, err := dir.Open("/index.html")
		emperror.Panic(errors.WrapIf(err, "open index.html"))

		indexContent, err := ioutil.ReadAll(indexFile)
		emperror.Panic(err)

		newIndexContent := []byte(strings.Replace(
			string(indexContent),
			"<base href=\"/\">",
			fmt.Sprintf("<base href=\"%s/\">", basePath),
			-1,
		))

		base.GET("/", func(c *gin.Context) {
			_, _ = c.Writer.Write(newIndexContent)
		})
		base.GET("index.html", func(c *gin.Context) {
			c.Redirect(http.StatusMovedPermanently, "./")
		})
	}

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
