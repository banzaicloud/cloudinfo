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

	// GetZones returns all the availability zones for a region.
	GetZones(provider, service, region string) ([]string, error)
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
	Region          string
	Zone            string
	Price           float64
	SpotPrice       float64
	CPU             float64
	Memory          float64
	Gpu             float64
	NetworkCategory NetworkCategory
	Category        InstanceTypeCategory
}

// InstanceTypeQuery represents the input parameters if an instance type query.
type InstanceTypeQuery struct {
	Region *string
	Zone   *string
	Filter *InstanceTypeQueryFilter
}

// InstanceTypeQueryFilter filters instance types by their fields.
type InstanceTypeQueryFilter struct {
	Price           *FloatFilter
	SpotPrice       *FloatFilter
	Spot            *bool
	CPU             *FloatFilter
	Memory          *FloatFilter
	Gpu             *FloatFilter
	NetworkCategory *NetworkCategoryFilter
	Category        *InstanceTypeCategoryFilter
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

// instanceTypeCategoryMap mapping between instance type (graphql) categories and cloudinfo generalisation
var instanceTypeCategoryMap = map[InstanceTypeCategory]string{
	InstanceTypeCategoryGeneralPurpose:   cloudinfo.CategoryGeneral,
	InstanceTypeCategoryComputeOptimized: cloudinfo.CategoryCompute,
	InstanceTypeCategoryStorageOptimized: cloudinfo.CategoryStorage,
	InstanceTypeCategoryMemoryOptimized:  cloudinfo.CategoryMemory,
}

// instanceTypeCategoryReverseMap mapping between instance type (graphql) categories and cloudinfo generalisation
var instanceTypeCategoryReverseMap = map[string]InstanceTypeCategory{
	cloudinfo.CategoryGeneral: InstanceTypeCategoryGeneralPurpose,
	cloudinfo.CategoryCompute: InstanceTypeCategoryComputeOptimized,
	cloudinfo.CategoryStorage: InstanceTypeCategoryStorageOptimized,
	cloudinfo.CategoryMemory:  InstanceTypeCategoryMemoryOptimized,
}

type InstanceTypeCategory string

const (
	InstanceTypeCategoryGeneralPurpose   InstanceTypeCategory = "GENERAL_PURPOSE"
	InstanceTypeCategoryMemoryOptimized  InstanceTypeCategory = "MEMORY_OPTIMIZED"
	InstanceTypeCategoryStorageOptimized InstanceTypeCategory = "STORAGE_OPTIMIZED"
	InstanceTypeCategoryComputeOptimized InstanceTypeCategory = "COMPUTE_OPTIMIZED"
)

var AllInstanceTypeCategory = []InstanceTypeCategory{
	InstanceTypeCategoryGeneralPurpose,
	InstanceTypeCategoryMemoryOptimized,
	InstanceTypeCategoryStorageOptimized,
	InstanceTypeCategoryComputeOptimized,
}

func (e InstanceTypeCategory) IsValid() bool {
	switch e {
	case InstanceTypeCategoryGeneralPurpose, InstanceTypeCategoryMemoryOptimized, InstanceTypeCategoryStorageOptimized, InstanceTypeCategoryComputeOptimized:
		return true
	}
	return false
}

func (e InstanceTypeCategory) String() string {
	return string(e)
}

func (e *InstanceTypeCategory) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = InstanceTypeCategory(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid InstanceTypeCategory", str)
	}
	return nil
}

func (e InstanceTypeCategory) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

// IntFilter represents the query operators for an instance type network category field.
type InstanceTypeCategoryFilter struct {
	Eq  *InstanceTypeCategory
	Ne  *InstanceTypeCategory
	In  []InstanceTypeCategory
	Nin []InstanceTypeCategory
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
		zones := product.Zones

		if len(zones) == 0 {
			var err error

			zones, err = s.store.GetZones(provider, service, *query.Region)
			if err != nil {
				return nil, emperror.Wrap(err, "failed to retrieve zones")
			}
		}

		if len(zones) == 0 {
			zones = []string{""}
		}

		for _, zone := range zones {
			if query.Filter != nil && !applyInstanceTypeFilter(product, zone, *query.Filter) {
				continue
			}

			instanceTypes = append(instanceTypes, transform(product, *query.Region, zone))
		}
	}

	return instanceTypes, nil
}

func applyInstanceTypeFilter(product cloudinfo.ProductDetails, zone string, filter InstanceTypeQueryFilter) bool {
	if filter.Price != nil && !applyFloatFilter(product.OnDemandPrice, *filter.Price) {
		return false
	}

	if filter.CPU != nil && !applyFloatFilter(product.Cpus, *filter.CPU) {
		return false
	}

	if filter.Memory != nil && !applyFloatFilter(product.Mem, *filter.Memory) {
		return false
	}

	if filter.Gpu != nil && !applyFloatFilter(product.Gpus, *filter.Gpu) {
		return false
	}

	if filter.NetworkCategory != nil && !applyNetworkCategoryFilter(product.NtwPerfCat, *filter.NetworkCategory) {
		return false
	}

	if filter.Category != nil && !applyInstanceTypeCategoryFilter(product.Category, *filter.Category) {
		return false
	}

	if filter.SpotPrice != nil || filter.Spot != nil {
		var spotPrice float64

		for _, zonePrice := range product.SpotPrice {
			if zonePrice.Zone == zone {
				spotPrice = zonePrice.Price
				break
			}
		}

		if filter.Spot != nil {
			if (*filter.Spot && spotPrice == 0.0) || (!*filter.Spot && spotPrice != 0.0) {
				return false
			}
		}

		if filter.SpotPrice != nil && !applyFloatFilter(spotPrice, *filter.SpotPrice) {
			return false
		}
	}

	return true
}

func applyNetworkCategoryFilter(value string, filter NetworkCategoryFilter) bool {
	if filter.Eq != nil && !(value == strings.ToLower(string(*filter.Eq))) {
		return false
	}

	if filter.Ne != nil && !(value != strings.ToLower(string(*filter.Ne))) {
		return false
	}

	if filter.In != nil {
		var in = false
		for _, v := range filter.In {
			if value == strings.ToLower(string(v)) {
				in = true
				break
			}
		}

		if !in {
			return false
		}
	}

	if filter.Nin != nil {
		for _, v := range filter.In {
			if value == strings.ToLower(string(v)) {
				return false
			}
		}
	}

	return true
}

func applyInstanceTypeCategoryFilter(value string, filter InstanceTypeCategoryFilter) bool {
	if filter.Eq != nil && !(value == instanceTypeCategoryMap[InstanceTypeCategory(*filter.Eq)]) {
		return false
	}

	if filter.Ne != nil && !(value != instanceTypeCategoryMap[InstanceTypeCategory(*filter.Ne)]) {
		return false
	}

	if filter.In != nil {
		var in = false
		for _, v := range filter.In {
			if value == instanceTypeCategoryMap[InstanceTypeCategory(v)] {
				in = true
				break
			}
		}

		if !in {
			return false
		}
	}

	if filter.Nin != nil {
		for _, v := range filter.In {
			if value == instanceTypeCategoryMap[InstanceTypeCategory(v)] {
				return false
			}
		}
	}

	return true
}

func transform(details cloudinfo.ProductDetails, region string, zone string) InstanceType {
	var spotPrice float64

	for _, zonePrice := range details.SpotPrice {
		if zonePrice.Zone == zone {
			spotPrice = zonePrice.Price
			break
		}
	}

	return InstanceType{
		Name:            details.Type,
		Region:          region,
		Zone:            zone,
		Price:           details.OnDemandPrice,
		SpotPrice:       spotPrice,
		CPU:             details.Cpus,
		Memory:          details.Mem,
		Gpu:             details.Gpus,
		NetworkCategory: NetworkCategory(strings.ToUpper(details.NtwPerfCat)),
		Category:        instanceTypeCategoryReverseMap[details.Category],
	}
}
