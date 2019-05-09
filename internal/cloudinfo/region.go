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
}

// RegionService returns the list of supported regions.
type RegionService struct {
	store RegionStore
}

// NewRegionService returns a new RegionService.
func NewRegionService(store RegionStore) *RegionService {
	return &RegionService{
		store: store,
	}
}

type Region struct {
	ID   string
	Name string
}

// ListRegions returns a list of regions supported by a service.
func (s *RegionService) ListRegions(ctx context.Context, provider string, service string) ([]Region, error) {
	cloudRegions, err := s.store.GetRegions(provider, service)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to list regions")
	}

	regions := make([]Region, len(cloudRegions))
	i := 0

	for id, name := range cloudRegions {
		regions[i] = Region{
			ID:   id,
			Name: name,
		}

		i++
	}

	return regions, nil
}
