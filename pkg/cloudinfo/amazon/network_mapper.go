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
	"fmt"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
)

var (
	ntwPerfMap = map[string][]string{
		// available categories
		//"10 Gigabit"
		//"20 Gigabit"
		//"25 Gigabit"
		//"High"
		//"Low to Moderate"
		//"Low"
		//"Moderate"
		//"NA"
		//"Up to 10 Gigabit"
		//"Very Low"

		cloudinfo.NTW_LOW:    {"Very Low", "Low", "Low to Moderate"},
		cloudinfo.NTW_MEDIUM: {"Moderate", "High"},
		cloudinfo.NTW_HIGH:   {"Up to 10 Gigabit", "10 Gigabit"},
		cloudinfo.NTW_EXTRA:  {"20 Gigabit", "25 Gigabit"},
	}
)

// AmazonNetworkMapper module object for handling amazon specific VM to Networking capabilities mapping
type AmazonNetworkMapper struct {
}

// newAmazonNetworkMapper initializes the network performance mapper struct
func newAmazonNetworkMapper() AmazonNetworkMapper {
	return AmazonNetworkMapper{}
}

// MapNetworkPerf maps the network performance of the ec2 to the category supported ny telescope
func (nm *AmazonNetworkMapper) MapNetworkPerf(ntwPerf string) (string, error) {
	for perfCat, strVals := range ntwPerfMap {
		if cloudinfo.Contains(strVals, ntwPerf) {
			return perfCat, nil
		}
	}
	return "", fmt.Errorf("could not determine network performance for: [%s]", ntwPerf)
}
