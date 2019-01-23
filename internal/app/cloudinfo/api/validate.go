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
	"reflect"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/gin-gonic/gin/binding"
	"gopkg.in/go-playground/validator.v8"
)

// ConfigureValidator configures the Gin validator with custom validator functions
func ConfigureValidator(ctx context.Context, providers []string, pi cloudinfo.CloudInfo) error {
	// retrieve the gin validator
	v := binding.Validator.Engine().(*validator.Validate)

	if err := v.RegisterValidation("provider", providerValidator(providers)); err != nil {
		return fmt.Errorf("could not register provider validator. error: %s", err)
	}

	if err := v.RegisterValidation("attribute", func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		for _, p := range pi.GetAttributes(ctx) {
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

// regionValidator validates the `region` path parameter
func regionValidator(ctx context.Context, cpi cloudinfo.CloudInfo) validator.Func {

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
		ci, err := cpi.GetInfoer(ctx, currentProvider)
		if err != nil {
			log.WithError(err).Error("could not get infoer")
			return false
		}

		regions, err := ci.GetRegions(ctx, currentService)
		if err != nil {
			log.WithError(err).Error("could not get regions")
			return false
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
func serviceValidator(ctx context.Context, cpi cloudinfo.CloudInfo) validator.Func {

	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {

		currentProvider := digValueForName(currentStruct, "Provider")
		currentService := digValueForName(currentStruct, "Service")

		ctx = logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(currentProvider).
			WithService(currentService).
			Build())

		log := logger.Extract(ctx)
		infoer, err := cpi.GetInfoer(ctx, currentProvider)
		if err != nil {
			log.WithError(err).Error("could not get infoer")
			return false
		}
		services, err := infoer.GetServices()
		if err != nil {
			log.WithError(err).Error("could not get services")
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
