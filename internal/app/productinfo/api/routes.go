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
	"github.com/banzaicloud/productinfo/pkg/logger"
	"net/http"
	"os"

	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"
)

// RouteHandler configures the REST API routes in the gin router
type RouteHandler struct {
	prod *productinfo.CachingProductInfo
}

// NewRouteHandler creates a new RouteHandler and returns a reference to it
func NewRouteHandler(p *productinfo.CachingProductInfo) *RouteHandler {
	return &RouteHandler{
		prod: p,
	}
}

func getCorsConfig() cors.Config {
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	if !config.AllowAllOrigins {
		config.AllowOrigins = []string{"http://", "https://"}
	}
	config.AllowMethods = []string{http.MethodPut, http.MethodDelete, http.MethodGet, http.MethodPost, http.MethodOptions}
	config.AllowHeaders = []string{"Origin", "Authorization", "Content-Type"}
	config.ExposeHeaders = []string{"Content-Length"}
	config.AllowCredentials = true
	config.MaxAge = 12
	return config
}

// ConfigureRoutes configures the gin engine, defines the rest API for this application
func (r *RouteHandler) ConfigureRoutes(ctx context.Context, router *gin.Engine) {
	logger.Extract(ctx).Info("configuring routes")

	v := binding.Validator.Engine().(*validator.Validate)

	basePath := "/"
	if basePathFromEnv := os.Getenv("PRODUCTINFO_BASEPATH"); basePathFromEnv != "" {
		basePath = basePathFromEnv
	}

	router.Use(logger.MiddlewareCorrelationId())
	router.Use(logger.Middleware())
	router.Use(cors.New(getCorsConfig()))
	router.Use(static.Serve(basePath, static.LocalFile("./web/dist/ui", true)))

	base := router.Group(basePath)
	{
		base.GET("/status", r.signalStatus)
	}

	v1 := base.Group("/api/v1")

	providerGroup := v1.Group("/providers")
	{

		providerGroup.GET("/", r.getProviders(ctx)).Use(ValidatePathParam(ctx, providerParam, v, "provider"))
		providerGroup.GET("/:provider", r.getProvider(ctx))
		providerGroup.GET("/:provider/services", r.getServices(ctx)).Use(ValidatePathData(ctx, v))
		providerGroup.GET("/:provider/services/:service", r.getService(ctx))
		providerGroup.GET("/:provider/services/:service/regions", r.getRegions(ctx)).Use(ValidatePathData(ctx, v))
		providerGroup.GET("/:provider/services/:service/regions/:region", r.getRegion(ctx))
		providerGroup.GET("/:provider/services/:service/regions/:region/images", r.getImages(ctx))
		providerGroup.GET("/:provider/services/:service/regions/:region/versions", r.getVersions(ctx))
		providerGroup.GET("/:provider/services/:service/regions/:region/products", r.getProducts(ctx))
		providerGroup.GET("/:provider/services/:service/regions/:region/products/:attribute", r.getAttrValues(ctx)).
			Use(ValidatePathParam(ctx, attributeParam, v, "attribute"))
	}

}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}
