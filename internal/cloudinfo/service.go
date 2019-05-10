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

// ServiceStore retrieves services.
type ServiceStore interface {
	// GetServices returns the supported services for a provider.
	GetServices(provider string) ([]cloudinfo.Service, error)
}

// ServiceService returns the list of supported services.
type ServiceService struct {
	store ServiceStore
}

// NewServiceService returns a new ServiceService.
func NewServiceService(store ServiceStore) *ServiceService {
	return &ServiceService{
		store: store,
	}
}

// Service represents a single service.
type Service struct {
	Code string

	providerName string
}

// ProviderName returns the provider name of the service.
// Used when resolving regions based on a service.
func (s *Service) ProviderName() string {
	return s.providerName
}

// ListServices returns a list of services supported by a provider.
func (s *ServiceService) ListServices(ctx context.Context, provider string) ([]Service, error) {
	cloudServices, err := s.store.GetServices(provider)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to list services")
	}

	services := make([]Service, len(cloudServices))

	for i, service := range cloudServices {
		services[i] = Service{
			Code:         service.Service,
			providerName: provider,
		}
	}

	return services, nil
}
