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

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

// nolint: gochecknoglobals
var providerNames = map[string]string{
	"amazon":  "Amazon Web Services",
	"google":  "Google Cloud",
	"alibaba": "Alibaba Cloud",
	"oracle":  "Oracle",
	"azure":   "Microsoft Azure",
}

// ProviderStore retrieves providers.
type ProviderStore interface {
	// GetProviders returns the supported providers.
	GetProviders() ([]types.Provider, error)
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
	Code string
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
		name, ok := providerNames[provider.Provider]
		if !ok {
			name = provider.Provider
		}

		providers[i] = Provider{
			Code: provider.Provider,
			Name: name,
		}
	}

	return providers, nil
}
