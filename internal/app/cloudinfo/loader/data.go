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

type ServiceDataList []ServiceData

// ServiceData serviuce data representation corresponding to the data to parsed from the external yaml / json
type ServiceData struct {
	Name    string       `json:"name"`
	Regions []RegionData `json:"regions"`
}

type RegionData struct {
	RegionId   string
	Region     string
	Zones      []ZoneData
	Images     []ImageData
	Versions   []VersionData
	Attributes []AttributeData
	Vms        []VmData
}

type ZoneData struct {
}

type ImageData struct {
}

type VersionData struct {
}

type AttributeData struct {
}

type VmData struct {
}
