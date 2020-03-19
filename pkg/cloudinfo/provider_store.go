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

// InMemoryProviderStore keeps providers in the memory.
// Use it in tests or for development/demo purposes.
type InMemoryProviderStore struct {
	providers []types.Provider
}

// NewInMemoryProviderStore returns a new InMemoryProviderStore.
func NewInMemoryProviderStore() *InMemoryProviderStore {
	return &InMemoryProviderStore{
		providers: []types.Provider{},
	}
}

func (s *InMemoryProviderStore) GetProviders() ([]types.Provider, error) {
	return s.providers, nil
}
