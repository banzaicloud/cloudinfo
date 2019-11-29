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

package alibaba

import (
	"strings"

	"emperror.dev/emperror"
	"emperror.dev/errors"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

var (
	categoryMap = map[string][]string{
		types.CategoryGeneral: {"g5", "sn2ne", "hfg5", "ebmhfg5", "ebmg5", "ebmg5s", "sccg5", "t5", "xn4", "n4", "mn4", "sn2", "n1", "n2", "s2", "t1", "s1", "s3"},
		types.CategoryCompute: {"ic5", "c5", "sn1ne", "hfc5", "ebmc4", "scch5", "sn1", "c4", "ce4", "cm4", "c1", "c2"},
		types.CategoryMemory:  {"r5", "re4", "re4e", "se1ne", "se1", "e4", "e3", "m1", "m2"},
		types.CategoryStorage: {"d1ne", "d1", "i2", "i2g", "i1"},
		types.CategoryGpu:     {"gn6v", "gn5", "gn5i", "gn4", "ga1", "f1", "f3", "vgn5i"},
	}
)

// mapCategory maps the family of the alibaba instance to category
func (a *AlibabaInfoer) mapCategory(name string) (string, error) {
	family := strings.Split(name, ".")[1]
	if strings.Contains(family, "-") {
		family = strings.Split(family, "-")[0]
	}

	for category, strVals := range categoryMap {
		if cloudinfo.Contains(strVals, family) {
			return category, nil
		}
	}
	return "", emperror.Wrap(errors.New(family), "could not determine the category")
}
