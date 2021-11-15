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

	"emperror.dev/emperror"
	"emperror.dev/errors"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

var (
	categoryMap = map[string][]string{
		types.CategoryGeneral: {"Dv2", "Av2", "Dv3", "DSv2", "DSv3", "BS", "DS", "D", "A0_A7", "A", "A8_A11", "DCS"},
		types.CategoryCompute: {"H", "FSv2", "FS", "F", "", "HCS", "HBS"},
		types.CategoryMemory:  {"Ev3", "ESv3", "MS", "G", "GS", "EIv3", "EISv3", "PBS", "MSv2", "MDSv2", "EDSv4", "ESv4"},
		types.CategoryStorage: {"LS", "LSv2"},
		types.CategoryGpu:     {"NC", "NV", "NCSv3", "NCSv2", "NDS", "NVSv2", "NVSv3", "ND"},
	}

	customMap = map[string]string{
		"MDSMediumMemoryv2":  "MDSv2",
		"MIDSMediumMemoryv2": "MDSv2",
		"MISMediumMemoryv2":  "MSv2",
		"MSMediumMemoryv2":   "MSv2",
		"NDASv4_A100":        "ND",
		"XEIDSv4":            "EDSv4",
		"XEISv4":             "ESv4",
	}
)

// mapCategory maps the family of the azure instance to category
func (a *AzureInfoer) mapCategory(name string) (string, error) {
	family := GetFamily(name)

	for category, strVals := range categoryMap {
		if cloudinfo.Contains(strVals, family) {
			return category, nil
		}
	}
	return "", emperror.Wrap(errors.New(family), "could not determine the category")
}

// mapSeries get instance series associated with the instanceType
func (a *AzureInfoer) mapSeries(name string) string {
	return GetFamily(name)
}

func GetFamily(name string) string {
	family := strings.TrimRight(name, "Family")
	family = strings.TrimLeft(family, "standard") // nolint: staticcheck
	family = strings.TrimLeft(family, "Standard") // nolint: staticcheck
	family = strings.TrimRight(family, "Promo")   // nolint: staticcheck
	family = strings.TrimLeft(family, "basic")

	for _, key := range customMap {
		if key == family {
			return customMap[family]
		}
	}

	return family
}
