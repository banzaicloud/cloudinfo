// Copyright © 2019 Banzai Cloud
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
	"testing"

	"gopkg.in/go-playground/validator.v8"
)

func TestGetProviderPathParamsValidation(t *testing.T) {
	tests := []struct {
		name      string
		pathParam interface{}
		check     func(t *testing.T, err error)
	}{
		//{
		//	name:      "getProvider path params validation should fail when provider not specified",
		//	pathParam: &GetProviderPathParams{},
		//	check: func(t *testing.T, err error) {
		//		assert.NotNil(t, err, "validation should fail", err)
		//	},
		//},
		//{
		//	name:      "getProvider path params validation should fail when provider is not supported",
		//	pathParam: &GetProviderPathParams{Provider: "unsupported"},
		//	check: func(t *testing.T, err error) {
		//		assert.NotNil(t, err, "validation should fail %#V", err)
		//	},
		//},
		//{
		//	name:      "getProvider path params validation should pass when provider is supported",
		//	pathParam: &GetProviderPathParams{Provider: "test-provider-1"},
		//	check: func(t *testing.T, err error) {
		//		assert.Nil(t, err, "validation should not fail")
		//	},
		//},
	}

	// setup the validator
	config := &validator.Config{TagName: "binding"}
	v := validator.New(config)
	v.RegisterValidation("provider", providerValidator([]string{"test-provider-1", "test-provider-2"}))

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(t, v.Struct(test.pathParam))
		})
	}
}
