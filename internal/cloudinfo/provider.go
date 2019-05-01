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

	"github.com/goph/emperror"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
)

// ProviderStore retrieves providers.
type ProviderStore interface {
	// GetProviders returns the supported providers.
	GetProviders() ([]cloudinfo.Provider, error)

	// GetServices returns the supported services for a provider.
	GetServices(provider string) ([]cloudinfo.Service, error)
}

// ProviderService returns the list of supported providers and relevant information.
type ProviderService struct {
	store ProviderStore
}

// NewProviderService returns a new ProviderService.
func NewProviderService(store ProviderStore) *ProviderService {
	return &ProviderService{
		store: store,
	}
}

// Provider represents a single cloud provider.
type Provider struct {
	Name     string
	Services []Service
}

type Service struct {
	Name string
}

// ListProviders returns a list of providers.
func (s *ProviderService) ListProviders(ctx context.Context) ([]Provider, error) {
	cloudProviders, err := s.store.GetProviders()
	if err != nil {
		return nil, emperror.Wrap(err, "failed to list providers")
	}

	providers := make([]Provider, len(cloudProviders))

	for i, provider := range cloudProviders {
		cloudServices, err := s.store.GetServices(provider.Provider)
		if err != nil {
			return nil, err
		}

		services := make([]Service, len(cloudServices))

		for j, service := range cloudServices {
			services[j] = Service{
				Name: service.Service,
			}
		}

		providers[i] = Provider{
			Name:     provider.Provider,
			Services: services,
		}
	}

	return providers, nil
}
