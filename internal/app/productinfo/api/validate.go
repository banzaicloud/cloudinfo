package api

import (
	"fmt"
	"net/http"
	"reflect"

	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v8"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(providers []string, pi *productinfo.CachingProductInfo) {
	// retrieve the gin validator
	v := binding.Validator.Engine().(*validator.Validate)

	v.RegisterValidation("provider", func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		for _, p := range providers {
			if field.String() == p {
				return true
			}
		}
		return false
	})
	v.RegisterValidation("attribute", func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		for _, p := range pi.GetAttributes() {
			if field.String() == p {
				return true
			}
		}
		return false
	})

	// register validator for the service parameter in the request path
	v.RegisterValidation("service", serviceValidator(pi))

	// register validator for the region parameter in the request path
	v.RegisterValidation("region", regionValidator(pi))

}

// ValidatePathParam is a gin middleware handler function that validates a named path parameter with specific Validate tags
func ValidatePathParam(name string, validate *validator.Validate, tags ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Param(name)
		for _, tag := range tags {
			err := validate.Field(p, tag)
			if err != nil {
				logrus.Errorf("validation failed. err: %s", err.Error())
				c.Abort()
				c.JSON(http.StatusBadRequest, gin.H{
					"code":    "bad_params",
					"message": fmt.Sprintf("invalid %s parameter", name),
					"params":  map[string]string{name: p},
				})
				return
			}
		}
	}
}

// ValidatePathData middleware function to validate region information in the request path.
func ValidatePathData(validate *validator.Validate) gin.HandlerFunc {
	const (
		providerParam = "provider"
		serviceParam  = "service"
		regionParam   = "region"
	)
	return func(c *gin.Context) {
		var data interface{}
		// build the appropriate internal struct based on the path params
		provider, _ := c.Params.Get(providerParam)
		service, _ := c.Params.Get(serviceParam)
		region, hasRegion := c.Params.Get(regionParam)

		data = newServiceData(provider, service)

		if hasRegion {
			data = regionData{pathData: data.(pathData), Region: region}
		}

		logrus.Debugf("path data is being validated: %s", data)
		err := validate.Struct(data)
		if err != nil {
			logrus.Errorf("validation failed. err: %s", err.Error())
			c.Abort()
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "bad_params",
				"message": fmt.Sprintf("invalid path parameter value: %s", data),
				"params":  data,
			})
			return
		}
	}

}

// pathData struct encapsulates request path data for validation purposes
type pathData struct {
	// Cloud the cloud provider from the request path
	Cloud string `binding:"required"`
	// Service the service in the request path
	Service string `binding:"service"`
}

type regionData struct {
	// embedded
	pathData
	// Region the region in the request path
	Region string `binding:"region"`
}

// String representation of the path data
func (rd *pathData) String() string {
	return fmt.Sprintf("Cloud: %s, Service: %s", rd.Cloud, rd.Service)
}

// newPathData constructs a new struct
func newServiceData(cloud, service string) pathData {
	return pathData{Cloud: cloud, Service: service}
}

// regionValidator validates the `region` path parameter
func regionValidator(cpi *productinfo.CachingProductInfo) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		currentProvider := currentStruct.FieldByName("Cloud").String()
		currentRegion := currentStruct.FieldByName("Region").String()

		regions, err := cpi.GetRegions(currentProvider)
		if err != nil {
			logrus.Errorf("could not get regions for provider: %s, err: %s", currentProvider, err.Error())
		}

		logrus.Debugf("current region: %s, regions: %#v", currentRegion, regions)
		for reg := range regions {
			if reg == currentRegion {
				return true
			}
		}
		return false
	}
}

// serviceValidator validates the `service` path parameter
func serviceValidator(cpi *productinfo.CachingProductInfo) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		currentProvider := currentStruct.FieldByName("Cloud").String()
		currentService := currentStruct.FieldByName("Service").String()

		infoer, err := cpi.GetInfoer(currentProvider)
		if err != nil {
			logrus.Errorf("could not get information for provider: %s, err: %s", currentProvider, err.Error())
		}
		services, err := infoer.GetServices()
		if err != nil {
			logrus.Errorf("could not get services for provider: [%s], err: %s", currentProvider, err.Error())
		}

		for _, svc := range services {
			if svc.ServiceName() == currentService {
				return true
			}
		}
		return false
	}
}
