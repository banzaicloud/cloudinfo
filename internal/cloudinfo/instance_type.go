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
	"strings"

	"github.com/goph/emperror"
	"github.com/pkg/errors"

	"github.com/banzaicloud/cloudinfo/.gen/api/graphql"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
)

// InstanceTypeStore retrieves instance types from the given provider and region.
type InstanceTypeStore interface {
	// GetProductDetails retrieves product details from the given provider and region.
	GetProductDetails(provider string, service string, region string) ([]cloudinfo.ProductDetails, error)
}

// InstanceTypeService filters instance types according to the received query.
type InstanceTypeService struct {
	store InstanceTypeStore
}

// NewInstanceTypeService returns a new InstanceTypeService.
func NewInstanceTypeService(store InstanceTypeStore) *InstanceTypeService {
	return &InstanceTypeService{
		store: store,
	}
}

// InstanceTypeQuery represents the input parameters if an instance type query.
type InstanceTypeQuery struct {
	Region *string
	Zone   *string
	Filter graphql.InstanceTypeQueryInput
}

// InstanceTypeQueryValidationError is returned if an instance type query is invalid.
type InstanceTypeQueryValidationError struct {
	Message string
}

// Error implements the error interface.
func (e InstanceTypeQueryValidationError) Error() string {
	return e.Message
}

// Query processes an instance type query and responds with a list match of instance types matching that query.
func (s *InstanceTypeService) Query(ctx context.Context, provider string, service string, query InstanceTypeQuery) ([]graphql.InstanceType, error) {
	if provider == "" {
		return nil, errors.WithStack(InstanceTypeQueryValidationError{
			Message: "provider field must not be empty",
		})
	}

	if service == "" {
		return nil, errors.WithStack(InstanceTypeQueryValidationError{
			Message: "service field must not be empty",
		})
	}

	// TODO: make region optional
	if query.Region == nil || *query.Region == "" {
		return nil, errors.WithStack(InstanceTypeQueryValidationError{
			Message: "region field must not be empty",
		})
	}

	var instanceTypes []graphql.InstanceType

	// load the data from the store
	products, err := s.store.GetProductDetails(provider, service, *query.Region)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to retrieve product details")
	}

	// filter the data
	for _, product := range products {
		includeInResults := true

		if query.Filter.Price != nil {
			includeInResults = includeInResults && applyFloatFilter(product.OnDemandPrice, *query.Filter.Price)
		}

		if query.Filter.CPU != nil {
			includeInResults = includeInResults && applyFloatFilter(product.Cpus, *query.Filter.CPU)
		}

		if query.Filter.Memory != nil {
			includeInResults = includeInResults && applyFloatFilter(product.Mem, *query.Filter.Memory)
		}

		if query.Filter.NetworkCategory != nil {
			includeInResults = includeInResults && applyNetworkCategoryFilter(product.NtwPerfCat, *query.Filter.NetworkCategory)
		}

		if includeInResults {
			instanceTypes = append(instanceTypes, transform(product))
		}
	}

	return instanceTypes, nil
}

func transform(details cloudinfo.ProductDetails) graphql.InstanceType {
	return graphql.InstanceType{
		Price:           details.OnDemandPrice,
		Name:            details.Type,
		CPU:             details.Cpus,
		Memory:          details.Mem,
		NetworkCategory: graphql.NetworkCategory(strings.ToUpper(details.NtwPerfCat)),
	}
}
