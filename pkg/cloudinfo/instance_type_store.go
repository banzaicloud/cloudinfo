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
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/types"
)

// InMemoryInstanceTypeStore keeps instance types in the memory.
// Use it in tests or for development/demo purposes.
type InMemoryInstanceTypeStore struct {
	products map[string]map[string]map[string][]types.ProductDetails
}

// NewInMemoryInstanceTypeStore returns a new InMemoryInstanceTypeStore.
func NewInMemoryInstanceTypeStore() *InMemoryInstanceTypeStore {
	return &InMemoryInstanceTypeStore{
		products: make(map[string]map[string]map[string][]types.ProductDetails),
	}
}

// GetProductDetails retrieves product details from the given provider and region.
func (s *InMemoryInstanceTypeStore) GetProductDetails(provider string, service string, region string) ([]types.ProductDetails, error) {
	return s.products[provider][service][region], nil
}

// GetZones returns all the availability zones for a region.
func (s *InMemoryInstanceTypeStore) GetZones(provider, service, region string) ([]string, error) {
	return []string{}, nil
}
