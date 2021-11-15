// Copyright © 2021 Banzai Cloud
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
	"strings"
)

// Instance series known to us
// a2 , c2 , e2 , f1, g1, m1, n1, n2, n2d, t2d

// mapSeries get instance series associated with the instanceType
func (g *GceInfoer) mapSeries(instanceType string) string {
	instanceTypeParts := strings.Split(instanceType, "-")

	if len(instanceTypeParts) == 2 || len(instanceTypeParts) == 3 {
		return instanceTypeParts[0]
	}

	g.log.Warn("error parsing instance series from instanceType", map[string]interface{}{"instanceType": instanceType})

	// return instanceType as fallback so that it can be easily debugged from the caller of the APIs
	return instanceType
}
