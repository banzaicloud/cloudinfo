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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegionService_ListRegions(t *testing.T) {
	store := NewInMemoryRegionStore()
	store.regions = map[string]map[string]map[string]string{
		"amazon": {
			"compute": {
				"eu-west-1": "EU (Ireland)",
				"eu-west-2": "EU (London)",
				"eu-west-3": "EU (Paris)",
			},
		},
	}

	serviceService := NewRegionService(store)

	regions, err := serviceService.ListRegions(context.Background(), "amazon", "compute")
	require.NoError(t, err)

	assert.Equal(
		t,
		[]Region{
			{Code: "eu-west-1", Name: "EU (Ireland)", providerName: "amazon", serviceName: "compute"},
			{Code: "eu-west-2", Name: "EU (London)", providerName: "amazon", serviceName: "compute"},
			{Code: "eu-west-3", Name: "EU (Paris)", providerName: "amazon", serviceName: "compute"},
		},
		regions,
	)
}

func TestRegionService_ListZones(t *testing.T) {
	store := NewInMemoryRegionStore()
	store.zones = map[string]map[string]map[string][]string{
		"amazon": {
			"compute": {
				"eu-west-1": []string{"eu-west-1a", "eu-west-1b", "eu-west-1c"},
			},
		},
	}

	serviceService := NewRegionService(store)

	zones, err := serviceService.ListZones(context.Background(), "amazon", "compute", "eu-west-1")
	require.NoError(t, err)

	assert.Equal(
		t,
		[]Zone{
			{Code: "eu-west-1a"},
			{Code: "eu-west-1b"},
			{Code: "eu-west-1c"},
		},
		zones,
	)
}
