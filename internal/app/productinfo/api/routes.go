package api

import (
	"net/http"
	"os"

	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
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
func (r *RouteHandler) ConfigureRoutes(router *gin.Engine) {
	log.Info("configuring routes")

	v := binding.Validator.Engine().(*validator.Validate)

	basePath := "/"
	if basePathFromEnv := os.Getenv("PRODUCTINFO_BASEPATH"); basePathFromEnv != "" {
		basePath = basePathFromEnv
	}

	router.Use(cors.New(getCorsConfig()))
	router.Use(static.Serve(basePath, static.LocalFile("./web/dist/ui", true)))

	base := router.Group(basePath)
	{
		base.GET("/status", r.signalStatus)
	}

	v1 := base.Group("/api/v1")

	providerGroup := v1.Group("/providers")
	{

		providerGroup.GET("/", r.getProviders).Use(ValidatePathParam(providerParam, v, "provider"))
		providerGroup.GET("/:provider", r.getProvider)
		providerGroup.GET("/:provider/services", r.getServices)
		providerGroup.GET("/:provider/services/:service", r.getService)
		providerGroup.GET("/:provider/services/:service/regions", r.getRegions).Use(ValidateRegionData(v))
		providerGroup.GET("/:provider/services/:service/regions/:region", r.getRegion)
		providerGroup.GET("/:provider/services/:service/regions/:region/images", r.getImages)
		providerGroup.GET("/:provider/services/:service/regions/:region/products", r.getProducts)
		providerGroup.GET("/:provider/services/:service/regions/:region/products/:attribute", r.getAttrValues).
			Use(ValidatePathParam(attributeParam, v, "attribute"))
	}

}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}
