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

// InMemoryRegionStore keeps regions in the memory.
// Use it in tests or for development/demo purposes.
type InMemoryRegionStore struct {
	regions map[string]map[string]map[string]string
	zones   map[string]map[string]map[string][]string
}

// NewInMemoryRegionStore returns a new InMemoryRegionStore.
func NewInMemoryRegionStore() *InMemoryRegionStore {
	return &InMemoryRegionStore{
		regions: make(map[string]map[string]map[string]string),
		zones:   make(map[string]map[string]map[string][]string),
	}
}

func (s *InMemoryRegionStore) GetRegions(provider string, service string) (map[string]string, error) {
	return s.regions[provider][service], nil
}

func (s *InMemoryRegionStore) GetZones(provider string, service string, region string) ([]string, error) {
	return s.zones[provider][service][region], nil
}
