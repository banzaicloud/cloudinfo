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

package cloudinfo

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
)

func TestInstanceTypeService_BasicValidation(t *testing.T) {
	t.Parallel()

	service := NewInstanceTypeService(NewInMemoryInstanceTypeStore())

	t.Run("provider", func(t *testing.T) {
		_, err := service.Query(context.Background(), "", "service", InstanceTypeQuery{})
		require.Error(t, err)
		require.IsType(t, InstanceTypeQueryValidationError{}, errors.Cause(err))
		require.EqualError(t, err, "provider field must not be empty")
	})

	t.Run("service", func(t *testing.T) {
		_, err := service.Query(context.Background(), "provider", "", InstanceTypeQuery{})
		require.Error(t, err)
		require.IsType(t, InstanceTypeQueryValidationError{}, errors.Cause(err))
		require.EqualError(t, err, "service field must not be empty")
	})

	t.Run("region_nil", func(t *testing.T) {
		_, err := service.Query(context.Background(), "provider", "service", InstanceTypeQuery{})
		require.Error(t, err)
		require.IsType(t, InstanceTypeQueryValidationError{}, errors.Cause(err))
		require.EqualError(t, err, "region field must not be empty")
	})

	t.Run("region", func(t *testing.T) {
		region := ""
		_, err := service.Query(context.Background(), "provider", "service", InstanceTypeQuery{
			Region: &region,
		})
		require.Error(t, err)
		require.IsType(t, InstanceTypeQueryValidationError{}, errors.Cause(err))
		require.EqualError(t, err, "region field must not be empty")
	})
}

func TestInstanceTypeService_Query(t *testing.T) {
	provider := "banzaicloud"
	service := "pke"
	region := "eu-west-1"

	productsFile, err := os.Open("testdata/products.json")
	if err != nil {
		t.Fatal(err)
	}

	var products []cloudinfo.ProductDetails
	decoder := json.NewDecoder(productsFile)

	err = decoder.Decode(&products)
	if err != nil {
		t.Fatal(err)
	}

	store := NewInMemoryInstanceTypeStore()
	store.products = map[string]map[string]map[string][]cloudinfo.ProductDetails{
		provider: {
			service: {
				region: products,
			},
		},
	}

	instanceTypeService := NewInstanceTypeService(store)

	price := 0.11
	cpu := float64(2)
	mem := float64(8)
	gpu := float64(0)
	net := NetworkCategoryHigh

	query := InstanceTypeQuery{
		Region: &region,
		Filter: InstanceTypeQueryFilter{
			Price: &FloatFilter{
				Gte: &price,
			},
			CPU: &FloatFilter{
				Eq: &cpu,
			},
			Memory: &FloatFilter{
				Eq: &mem,
			},
			Gpu: &FloatFilter{
				Eq: &gpu,
			},
			NetworkCategory: &NetworkCategoryFilter{
				Eq: &net,
			},
		},
	}

	result, err := instanceTypeService.Query(context.Background(), provider, service, query)
	require.NoError(t, err)

	t.Logf("%+v", result)
}
