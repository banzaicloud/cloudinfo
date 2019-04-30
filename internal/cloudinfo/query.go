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

package cloudinfo

// FloatFilter represents the query operators for a float field.
type FloatFilter struct {
	Lt  *float64
	Lte *float64
	Gt  *float64
	Gte *float64
	Eq  *float64
	Ne  *float64
	In  []float64
	Nin []float64
}

// IntFilter represents the query operators for an int field.
type IntFilter struct {
	Lt  *int
	Lte *int
	Gt  *int
	Gte *int
	Eq  *int
	Ne  *int
	In  []int
	Nin []int
}

func applyFloatFilter(value float64, filter FloatFilter) bool {
	if filter.Eq != nil && !(value == *filter.Eq) {
		return false
	}

	if filter.Ne != nil && !(value != *filter.Ne) {
		return false
	}

	if filter.Lt != nil && !(value < *filter.Lt) {
		return false
	}

	if filter.Lte != nil && !(value <= *filter.Lte) {
		return false
	}

	if filter.Gt != nil && !(value > *filter.Gt) {
		return false
	}

	if filter.Gte != nil && !(value >= *filter.Gte) {
		return false
	}

	if filter.In != nil {
		var in = false
		for _, v := range filter.In {
			if value == v {
				in = true
				break
			}
		}

		if !in {
			return false
		}
	}

	if filter.Nin != nil {
		for _, v := range filter.In {
			if value == v {
				return false
			}
		}
	}

	return true
}

// nolint: deadcode
func applyIntFilter(value int, filter IntFilter) bool {
	if filter.Eq != nil && !(value == *filter.Eq) {
		return false
	}

	if filter.Ne != nil && !(value != *filter.Ne) {
		return false
	}

	if filter.Lt != nil && !(value < *filter.Lt) {
		return false
	}

	if filter.Lte != nil && !(value <= *filter.Lte) {
		return false
	}

	if filter.Gt != nil && !(value > *filter.Gt) {
		return false
	}

	if filter.Gte != nil && !(value >= *filter.Gte) {
		return false
	}

	if filter.In != nil {
		var in = false
		for _, v := range filter.In {
			if value == v {
				in = true
				break
			}
		}

		if !in {
			return false
		}
	}

	if filter.Nin != nil {
		for _, v := range filter.In {
			if value == v {
				return false
			}
		}
	}

	return true
}
