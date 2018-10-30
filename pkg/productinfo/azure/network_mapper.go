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

package azure

import (
	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

var (
	// TODO
	ntwPerfMap = map[string][]string{
		productinfo.NTW_LOW:    {"Low"},
		productinfo.NTW_MEDIUM: {"Moderate"},
		productinfo.NTW_HIGH:   {""},
	}
)

// AzureNetworkMapper module object for handling Azure specific VM to Networking capabilities mapping
type AzureNetworkMapper struct {
}

// MapNetworkPerf maps the network performance of the azure instance to the category supported by telescopes
func (nm *AzureNetworkMapper) MapNetworkPerf(ntwPerf string) (string, error) {
	return ntwPerfMap[productinfo.NTW_MEDIUM][0], nil
}
