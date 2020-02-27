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
	"emperror.dev/errors"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/types"
)

var (
	networkPerformanceCategories = map[string][]string{
		// available categories
		// "Up to 25 Gigabit"
		// "50 Gigabit"
		// "100 Gigabit"
		// "10 Gigabit"
		// "20 Gigabit"
		// "25 Gigabit"
		// "High"
		// "Low to Moderate"
		// "Low"
		// "Moderate"
		// "NA"
		// "Up to 10 Gigabit"
		// "Very Low"

		types.NtwLow:    {"Very Low", "Low", "Low to Moderate"},
		types.NtwMedium: {"Moderate", "High"},
		types.NtwHight:  {"Up to 10 Gigabit", "10 Gigabit"},
		types.NtwExtra:  {"20 Gigabit", "25 Gigabit", "Up to 25 Gigabit", "50 Gigabit", "100 Gigabit"},
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
func (nm *AmazonNetworkMapper) MapNetworkPerf(networkPerformance string) (string, error) {
	for category, values := range networkPerformanceCategories {
		if cloudinfo.Contains(values, networkPerformance) {
			return category, nil
		}
	}
	return "", errors.NewWithDetails("could not determine network performance", "networkPerf", networkPerformance)
}
