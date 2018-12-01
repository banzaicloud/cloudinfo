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
	"fmt"
	"net/http"
	"reflect"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/go-playground/validator.v8"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(ctx context.Context, providers []string, pi *cloudinfo.CachingCloudInfo) error {
	// retrieve the gin validator
	v := binding.Validator.Engine().(*validator.Validate)

	if err := v.RegisterValidation("provider", func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		for _, p := range providers {
			if field.String() == p {
				return true
			}
		}
		return false
	}); err != nil {
		return fmt.Errorf("could not register provider validator. error: %s", err)
	}

	if err := v.RegisterValidation("attribute", func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		for _, p := range pi.GetAttributes() {
			if field.String() == p {
				return true
			}
		}
		return false
	}); err != nil {
		return fmt.Errorf("could not register attribute validator. error: %s", err)
	}

	// register validator for the service parameter in the request path
	if err := v.RegisterValidation("service", serviceValidator(ctx, pi)); err != nil {
		return fmt.Errorf("could not register service validator. error: %s", err)
	}

	// register validator for the region parameter in the request path
	if err := v.RegisterValidation("region", regionValidator(ctx, pi)); err != nil {
		return fmt.Errorf("could not register provider validator. . error: %s", err)
	}
	return nil
}

// ValidatePathParam is a gin middleware handler function that validates a named path parameter with specific Validate tags
func ValidatePathParam(ctx context.Context, name string, validate *validator.Validate, tags ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Param(name)
		for _, tag := range tags {
			err := validate.Field(p, tag)
			if err != nil {
				logger.Extract(ctx).WithError(err).Error("validation failed.")
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
func ValidatePathData(ctx context.Context, validate *validator.Validate) gin.HandlerFunc {
	const (
		regionParam = "region"
	)
	log := logger.Extract(ctx)
	return func(c *gin.Context) {

		var pathData interface{}
		// build the appropriate internal struct based on the path params
		_, hasRegion := c.Params.Get(regionParam)

		if hasRegion {
			pathData = &GetRegionPathParams{}
		} else {
			pathData = &GetServicesPathParams{}
		}

		if err := mapstructure.Decode(getPathParamMap(c), pathData); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"status": http.StatusBadRequest, "message": fmt.Sprintf("%s", err)})
			return
		}

		log.Debugf("path data is being validated: %s", pathData)
		err := validate.Struct(pathData)
		if err != nil {
			log.WithError(err).Error("validation failed.")
			c.Abort()
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "bad_params",
				"message": fmt.Sprintf("invalid path parameter value: %s", pathData),
				"params":  pathData,
			})
			return
		}
	}

}

// validationFn validation logic for the region data to be registered with the validator
func regionValidator(ctx context.Context, cpi *cloudinfo.CachingCloudInfo) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		currentProvider := digValueForName(currentStruct, "Provider")
		currentService := digValueForName(currentStruct, "Service")
		currentRegion := digValueForName(currentStruct, "Region")

		ctx = logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(currentProvider).
			WithService(currentService).
			WithRegion(currentRegion).
			Build())

		log := logger.Extract(ctx)
		regions, err := cpi.GetRegions(ctx, currentProvider, currentService)
		if err != nil {
			log.WithError(err).Error("could not get regions")
		}

		log.Debugf("current region: %s, regions: %#v", currentRegion, regions)
		for reg := range regions {
			if reg == currentRegion {
				return true
			}
		}
		return false
	}
}

// serviceValidator validates the `service` path parameter
func serviceValidator(ctx context.Context, cpi *cloudinfo.CachingCloudInfo) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		currentProvider := digValueForName(currentStruct, "Provider")
		currentService := digValueForName(currentStruct, "Service")

		ctx = logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(currentProvider).
			WithService(currentService).
			Build())

		log := logger.Extract(ctx)
		infoer, err := cpi.GetInfoer(currentProvider)
		if err != nil {
			log.WithError(err).Error("could not get information")
		}
		services, err := infoer.GetServices()
		if err != nil {
			log.WithError(err).Error("could not get services")
		}

		for _, svc := range services {
			if svc.ServiceName() == currentService {
				return true
			}
		}
		return false
	}
}

func digValueForName(value reflect.Value, field string) string {
	var ret string
	switch value.Kind() {
	case reflect.Struct:
		ret = value.FieldByName(field).String()
	case reflect.Ptr:
		ret = value.Elem().FieldByName(field).String()
	}
	return ret
}
