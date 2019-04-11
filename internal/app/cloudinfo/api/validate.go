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
	"reflect"

	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/gin-gonic/gin/binding"
	"github.com/goph/logur"
	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v8"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(providers []string, ci cloudinfo.CloudInfo, logger logur.Logger) error {
	// retrieve the gin validator
	v := binding.Validator.Engine().(*validator.Validate)

	// register validator for the provider parameter in the request path
	if err := v.RegisterValidation("provider", providerValidator(providers)); err != nil {
		return errors.Wrap(err, "could not register provider validator")
	}

	// register validator for the service parameter in the request path
	if err := v.RegisterValidation("service", serviceValidator(ci, logger)); err != nil {
		return errors.Wrap(err, "could not register service validator")
	}

	// register validator for the region parameter in the request path
	if err := v.RegisterValidation("region", regionValidator(ci, logger)); err != nil {
		return errors.Wrap(err, "could not register region validator")
	}

	return nil
}

// regionValidator validates the `region` path parameter
func regionValidator(cpi cloudinfo.CloudInfo, logger logur.Logger) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		currentProvider := digValueForName(currentStruct, "Provider")
		currentService := digValueForName(currentStruct, "Service")
		currentRegion := digValueForName(currentStruct, "Region")

		logger = log.WithFields(logger,
			map[string]interface{}{"provider": currentProvider, "service": currentService, "region": currentRegion})

		locations, err := cpi.GetRegions(currentProvider, currentService)
		if err != nil {
			logger.Error("could not get regions")
			return false
		}

		for _, regions := range locations {
			for _, r := range regions {
				if r.Id == currentRegion {
					return true
				}
			}
		}
		return false
	}
}

// serviceValidator validates the `service` path parameter
func serviceValidator(cpi cloudinfo.CloudInfo, logger logur.Logger) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		currentProvider := digValueForName(currentStruct, "Provider")
		currentService := digValueForName(currentStruct, "Service")

		logger = log.WithFields(logger,
			map[string]interface{}{"provider": currentProvider, "service": currentService})

		services, err := cpi.GetServices(currentProvider)
		if err != nil {
			logger.Error("could not get services")
			return false
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

// providerValidator validates the `provider` path parameter
func providerValidator(providers []string) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		for _, p := range providers {
			if field.String() == p {
				return true
			}
		}
		return false
	}
}

// ValidatePathData explicitly calls validation on the parsed path data structs
func ValidatePathData(pathParams interface{}) error {
	v := binding.Validator.Engine().(*validator.Validate)
	return v.Struct(pathParams)
}
