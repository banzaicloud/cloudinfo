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

import (
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

const (
	exact   = "exact"
	exclude = "exclude"
	include = "include"
)

// ServiceData service data representation corresponding to the data to parsed from the external yaml / json
type ServiceData struct {
	// embedded service
	Service  `mapstructure:",squash"`
	Provider string
	Regions  []Region
}

type Region struct {
	Name string
	Id   string
	Data RegionData
}

type RegionData struct {
	Zones    ZoneData
	Images   ImageData
	Versions VersionData
	Vms      VmData
}

type ZoneData struct {
	Strategy string
	Data     []string
}

type ImageData struct {
	Strategy string
	Data     []types.Image
}

type VersionData struct {
	Strategy string
	Data     []types.LocationVersion
}

type VmData struct {
	Strategy string
	Data     []types.VMInfo
}

type Service struct {
	Name         string
	IsStatic     bool
	Source       string
	DataLocation string
	DataFile     string
	DataType     string
}
