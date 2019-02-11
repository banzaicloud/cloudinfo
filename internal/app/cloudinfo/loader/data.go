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

package loader

import "github.com/banzaicloud/cloudinfo/pkg/cloudinfo"

// ServiceData service data representation corresponding to the data to parsed from the external yaml / json
type ServiceData struct {
	// embedded service
	Service  `mapstructure:",squash"`
	Provider string
	Regions  []RegionData
}

type RegionData struct {
	RegionId   string
	Region     string
	Zones      []string
	Images     []cloudinfo.Image
	Versions   []string
	Attributes []AttributeData
	Vms        []cloudinfo.VmInfo
	Prices     []PriceData
}

type AttributeData struct {
	Name   string
	Values cloudinfo.AttrValues
}

type PriceData struct {
	Instancetype string
	Price        cloudinfo.Price
}

type Service struct {
	Name         string
	IsStatic     bool
	DataLocation string
	DataFile     string
}
