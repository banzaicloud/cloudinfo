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
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/goph/emperror"
	"github.com/pkg/errors"

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

// InstanceType represents a single instance type.
type InstanceType struct {
	Name            string
	Price           float64
	CPU             float64
	Memory          float64
	Gpu             float64
	NetworkCategory NetworkCategory
}

// InstanceTypeQuery represents the input parameters if an instance type query.
type InstanceTypeQuery struct {
	Region *string
	Zone   *string
	Filter InstanceTypeQueryFilter
}

// InstanceTypeQueryFilter filters instance types by their fields.
type InstanceTypeQueryFilter struct {
	Price           *FloatFilter
	CPU             *FloatFilter
	Memory          *FloatFilter
	Gpu             *FloatFilter
	NetworkCategory *NetworkCategoryFilter
}

// IntFilter represents the query operators for an instance type network category field.
type NetworkCategoryFilter struct {
	Eq  *NetworkCategory
	Ne  *NetworkCategory
	In  []NetworkCategory
	Nin []NetworkCategory
}

type NetworkCategory string

const (
	NetworkCategoryLow      NetworkCategory = "LOW"
	NetworkCategoryModerate NetworkCategory = "MODERATE"
	NetworkCategoryHigh     NetworkCategory = "HIGH"
)

var AllNetworkCategory = []NetworkCategory{
	NetworkCategoryLow,
	NetworkCategoryModerate,
	NetworkCategoryHigh,
}

func (e NetworkCategory) IsValid() bool {
	switch e {
	case NetworkCategoryLow, NetworkCategoryModerate, NetworkCategoryHigh:
		return true
	}
	return false
}

func (e NetworkCategory) String() string {
	return string(e)
}

func (e *NetworkCategory) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = NetworkCategory(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid NetworkCategory", str)
	}
	return nil
}

func (e NetworkCategory) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// InstanceTypeQueryValidationError is returned if an instance type query is invalid.
type InstanceTypeQueryValidationError struct {
	Message string
}

// Error implements the error interface.
func (e InstanceTypeQueryValidationError) Error() string {
	return e.Message
}

// IsBusinessError tells the transport layer whether this error should be translated into the transport format
// or an internal error should be returned instead.
func (InstanceTypeQueryValidationError) IsBusinessError() bool {
	return true
}

// Query processes an instance type query and responds with a list match of instance types matching that query.
func (s *InstanceTypeService) Query(ctx context.Context, provider string, service string, query InstanceTypeQuery) ([]InstanceType, error) {
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

	var instanceTypes []InstanceType

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

		if query.Filter.Gpu != nil {
			includeInResults = includeInResults && applyFloatFilter(product.Gpus, *query.Filter.Gpu)
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

func applyNetworkCategoryFilter(value string, filter NetworkCategoryFilter) bool {
	var result = true

	if filter.Eq != nil {
		result = result && value == strings.ToLower(string(*filter.Eq))
	}

	if filter.Ne != nil {
		result = result && value != strings.ToLower(string(*filter.Ne))
	}

	if filter.In != nil {
		var in = false
		for _, v := range filter.In {
			if value == strings.ToLower(string(v)) {
				in = true
				break
			}
		}

		result = result && in
	}

	if filter.Nin != nil {
		var nin = true
		for _, v := range filter.In {
			if value == strings.ToLower(string(v)) {
				nin = false
				break
			}
		}

		result = result && nin
	}

	return result
}

func transform(details cloudinfo.ProductDetails) InstanceType {
	return InstanceType{
		Price:           details.OnDemandPrice,
		Name:            details.Type,
		CPU:             details.Cpus,
		Memory:          details.Mem,
		Gpu:             details.Gpus,
		NetworkCategory: NetworkCategory(strings.ToUpper(details.NtwPerfCat)),
	}
}
