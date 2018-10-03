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

package google

import (
	"fmt"

	"github.com/banzaicloud/productinfo/pkg/productinfo"
)

var (
	ntwPerfMap = map[string][]string{
		productinfo.NTW_LOW:    {"1 Gbit/s", "2 Gbit/s"},
		productinfo.NTW_MEDIUM: {"4 Gbit/s", "6 Gbit/s", "8 Gbit/s"},
		productinfo.NTW_HIGH:   {"10 Gbit/s", "12 Gbit/s", "14 Gbit/s"},
		productinfo.NTW_EXTRA:  {"16 Gbit/s"},
	}
)

// GceNetworkMapper module object for handling Google Cloud specific VM to Networking capabilities mapping
type GceNetworkMapper struct {
}

// newGceNetworkMapper initializes the network performance mapper struct
func newGceNetworkMapper() *GceNetworkMapper {
	return &GceNetworkMapper{}
}

// MapNetworkPerf maps the network performance of the google instance to the category supported by telescopes
func (nm *GceNetworkMapper) MapNetworkPerf(ntwPerf string) (string, error) {
	for perfCat, strVals := range ntwPerfMap {
		if productinfo.Contains(strVals, ntwPerf) {
			return perfCat, nil
		}
	}
	return "", fmt.Errorf("could not determine network performance for: [%s]", ntwPerf)
}
