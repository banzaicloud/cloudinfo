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

	"emperror.dev/errors"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(providers []string, ci types.CloudInfo, logger cloudinfo.Logger) error {
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
func regionValidator(cpi types.CloudInfo, logger cloudinfo.Logger) validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		regionPathParams, ok := currentStruct.Interface().(GetRegionPathParams)
		if !ok {
			return false
		}

		logger = logger.WithFields(map[string]interface{}{"provider": regionPathParams.Provider, "service": regionPathParams.Service, "region": regionPathParams.Region})

		regions, err := cpi.GetRegions(regionPathParams.Provider, regionPathParams.Service)
		if err != nil {
			logger.Error("validation failed, could not retrieve regions")
			return false
		}

		for reg := range regions {
			if reg == regionPathParams.Region {
				return true
			}
		}
		return false
	}
}

// serviceValidator validates the `service` path parameter
func serviceValidator(cpi types.CloudInfo, logger cloudinfo.Logger) validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value,
		fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		servicesPathParams, ok := currentStruct.Interface().(GetServicesPathParams)
		if !ok {
			return false
		}

		logger = logger.WithFields(map[string]interface{}{"provider": servicesPathParams.Provider, "service": servicesPathParams.Service})

		services, err := cpi.GetServices(servicesPathParams.Provider)
		if err != nil {
			logger.Error("validation failed, could not retrieve services")
			return false
		}

		for _, svc := range services {
			if svc.ServiceName() == servicesPathParams.Service {
				return true
			}
		}
		return false
	}
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
