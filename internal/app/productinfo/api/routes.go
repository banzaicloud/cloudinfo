package api

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
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
		providerGroup.Use(ValidatePathParam(providerParam, v, "provider"))

		providerGroup.GET("/", r.getProviders)
		providerGroup.GET("/:provider/services", r.getServices)
		providerGroup.GET("/:provider/services/:service", r.getService)
		providerGroup.GET("/:provider/services/:service/regions", r.getRegions).Use(ValidateRegionData(v))
		providerGroup.GET("/:provider/services/:service/regions/:region", r.getRegion)
		//providerGroup.GET("/:provider/services/:service/regions/:region/images", r.getServiceImages)
		providerGroup.GET("/:provider/services/:service/regions/:region/products", r.getProductDetails)
		providerGroup.GET("/:provider/services/:service/regions/:region/products/:attribute", r.getAttrValues).
			Use(ValidatePathParam(attributeParam, v, "attribute"))
	}

}

func (r *RouteHandler) signalStatus(c *gin.Context) {
	c.JSON(http.StatusOK, "ok")
}

// swagger:route GET /providers/{provider}/services/{service}/regions/{region}/products products getProductDetails
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
func (r *RouteHandler) getProductDetails(c *gin.Context) {

	pathParams := GetRegionPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	log.Infof("getting product details for provider: %s, region: %s", pathParams.Provider, pathParams.Region)

	scrapingTime, err := r.prod.GetStatus(pathParams.Provider)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"status": http.StatusNotFound, "message": fmt.Sprintf("%s", err)})
		return
	}
	details, err := r.prod.GetProductDetails(pathParams.Provider, pathParams.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}

	log.Debugf("successfully retrieved product details:  %s, region: %s", pathParams.Provider, pathParams.Region)
	c.JSON(http.StatusOK, newProductDetailsResponse(details, scrapingTime))
	return
}

func newProductDetailsResponse(result []productinfo.ProductDetails, scrTime string) ProductDetailsResponse {
	if result == nil {
		result = make([]productinfo.ProductDetails, 0)
	}
	return ProductDetailsResponse{Products: result, ScrapingTime: scrTime}
}

// swagger:route GET /providers/{provider}/services/{service}/regions/{region}/products/{attribute} attributes getAttributeValues
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
func (r *RouteHandler) getAttrValues(c *gin.Context) {

	pathParams := GetAttributeValuesPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	log.Infof("getting %s attribute values for provider: %s, region: %s", pathParams.Attribute, pathParams.Provider, pathParams.Region)

	attributes, err := r.prod.GetAttrValues(pathParams.Provider, pathParams.Attribute)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}
	log.Debugf("successfully retrieved %s attribute values:  %s, region: %s", pathParams.Attribute, pathParams.Provider, pathParams.Region)
	c.JSON(http.StatusOK, AttributeResponse{pathParams.Attribute, attributes})
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
func (r *RouteHandler) getRegions(c *gin.Context) {

	pathParams := GetServicesPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	regions, err := r.prod.GetRegions(pathParams.Provider)
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
func (r *RouteHandler) getRegion(c *gin.Context) {
	pathParams := GetRegionPathParams{}
	mapstructure.Decode(getPathParamMap(c), &pathParams)

	regions, err := r.prod.GetRegions(pathParams.Provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}
	zones, err := r.prod.GetZones(pathParams.Provider, pathParams.Region)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": fmt.Sprintf("%s", err)})
		return
	}
	c.JSON(http.StatusOK, GetRegionResp{pathParams.Region, regions[pathParams.Region], zones})
}

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
//       200: ProviderResponse
func (r *RouteHandler) getProviders(c *gin.Context) {

	providers := r.prod.GetProviders()
	if len(providers) < 1 {
		c.JSON(http.StatusInternalServerError, gin.H{"status": http.StatusInternalServerError, "message": "no providers are configured"})
	}
	c.JSON(http.StatusOK, providers)
}
