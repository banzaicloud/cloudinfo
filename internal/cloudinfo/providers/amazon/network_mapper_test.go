// Copyright © 2018 Banzai Cloud
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

package amazon

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

func TestEc2NetworkMapper_MapNetworkPerf(t *testing.T) {

	mapper := AmazonNetworkMapper{}
	tests := []struct {
		name  string
		vm    types.VMInfo
		check func(cat string, err error)
	}{
		{
			name: "success - mapper maps to the lowest category",
			vm: types.VMInfo{
				NtwPerf: "Very Low",
			},
			check: func(cat string, err error) {
				assert.Equal(t, types.NtwLow, cat, "not mapped to the right category")
			},
		},
		{
			name: "error - mapper doesn't map to a category",
			vm: types.VMInfo{
				NtwPerf: "invalid",
			},
			check: func(cat string, err error) {
				assert.Equal(t, "", cat, "not mapped to the right category")
				assert.EqualError(t, err, "could not determine network performance")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(mapper.MapNetworkPerf(test.vm.NtwPerf))
		})
	}
}
