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
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEc2NetworkMapper_MapNetworkPerf(t *testing.T) {

	mapper := AmazonNetworkMapper{}
	tests := []struct {
		name  string
		vm    cloudinfo.VmInfo
		check func(cat string, err error)
	}{
		{
			name: "success - mapper maps to the lowest category",
			vm: cloudinfo.VmInfo{
				NtwPerf: "Very Low",
			},
			check: func(cat string, err error) {
				assert.Equal(t, cloudinfo.NetwLow, cat, "not mapped to the right category")
			},
		},
		{
			name: "error - mapper doesn't map to a category",
			vm: cloudinfo.VmInfo{
				NtwPerf: "Error",
			},
			check: func(cat string, err error) {
				assert.Equal(t, "", cat, "not mapped to the right category")
				assert.Equal(t, "could not determine network performance for: [Error]", err.Error())
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(mapper.MapNetworkPerf(test.vm.NtwPerf))
		})
	}
}
