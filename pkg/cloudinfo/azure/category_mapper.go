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

package azure

import (
	"strings"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
)

var (
	categoryMap = map[string][]string{
		cloudinfo.CategoryGeneral: {"Dv2", "Av2", "Dv3", "DSv2", "DSv3", "BS", "DS", "D", "A0_A7", "A", "A8_A11", "DCS"},
		cloudinfo.CategoryCompute: {"H", "FSv2", "FS", "", "HCS", "HBS"},
		cloudinfo.CategoryMemory:  {"Ev3", "ESv3", "MS", "G", "GS", "EIv3", "EISv3"},
		cloudinfo.CategoryStorage: {"LS", "LSv2"},
		cloudinfo.CategoryGpu:     {"NC", "NV", "NCSv3", "NCSv2", "NDS", "NVSv2"},
	}
)

// mapCategory maps the family of the azure instance to category
func (a *AzureInfoer) mapCategory(name string) (string, error) {
	family := strings.TrimRight(name, "Family")
	family = strings.TrimLeft(family, "standard")
	family = strings.TrimRight(family, "Promo")
	family = strings.TrimLeft(family, "basic")

	for category, strVals := range categoryMap {
		if cloudinfo.Contains(strVals, family) {
			return category, nil
		}
	}
	return "", emperror.Wrap(errors.New(family), "could not determine the category")
}
