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
)

// RegionStore retrieves regions.
type RegionStore interface {
	// GetRegions returns the supported regions for a service.
	GetRegions(provider string, service string) (map[string]string, error)

	// GetZones returns the supported zones within a region.
	GetZones(provider string, service string, region string) ([]string, error)
}

// RegionService provides access to regions supported by a service.
type RegionService struct {
	store RegionStore
}

// NewRegionService returns a new RegionService.
func NewRegionService(store RegionStore) *RegionService {
	return &RegionService{
		store: store,
	}
}

// Region is a general area in which a cloud provider has services available.
type Region struct {
	Code string
	Name string

	providerName string
	serviceName  string
}

// ProviderName returns the provider name of the region.
// Used when resolving zones based on a region.
func (r *Region) ProviderName() string {
	return r.providerName
}

// ServiceName returns the service name of the region.
// Used when resolving zones based on a region.
func (r *Region) ServiceName() string {
	return r.serviceName
}

// Zone is a specific location within a region.
type Zone struct {
	Code string
}

// ListRegions returns a list of regions supported by a service.
func (s *RegionService) ListRegions(ctx context.Context, provider string, service string) ([]Region, error) {
	cloudRegions, err := s.store.GetRegions(provider, service)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to list regions")
	}

	regions := make([]Region, len(cloudRegions))
	i := 0

	for code, name := range cloudRegions {
		regions[i] = Region{
			Code:         code,
			Name:         name,
			providerName: provider,
			serviceName:  service,
		}

		i++
	}

	return regions, nil
}

// ListZones returns a list of zones within a region.
func (s *RegionService) ListZones(ctx context.Context, provider string, service string, region string) ([]Zone, error) {
	cloudZones, err := s.store.GetZones(provider, service, region)
	if err != nil {
		return nil, emperror.WrapWith(
			err,
			"failed to list zones",
			"provider", provider,
			"service", service,
			"region", region,
		)
	}

	zones := make([]Zone, len(cloudZones))

	for i, code := range cloudZones {
		zones[i] = Zone{
			Code: code,
		}
	}

	return zones, nil
}
