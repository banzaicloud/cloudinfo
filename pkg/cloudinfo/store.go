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

import (
	"io"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/types"
)

const (
	// vmKeyTemplate format for generating vm cache keys
	VmKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/vms"

	// priceKeyTemplate format for generating price cache keys
	PriceKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/regions/%s/prices/%s"

	// zoneKeyTemplate format for generating zone cache keys
	ZoneKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/zones/"

	// regionKeyTemplate format for generating region cache keys
	RegionKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/"

	// statusKeyTemplate format for generating status cache keys
	StatusKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/status/"

	// imageKeyTemplate format for generating image cache keys
	ImageKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/images"

	// versionKeyTemplate format for generating kubernetes version cache keys
	VersionKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/versions"

	// servicesKeyTemplate key for storing provider specific services
	ServicesKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services"
)

// Storage operations for cloud information
type CloudInfoStore interface {
	StoreRegions(provider, service string, val map[string]string)
	GetRegions(provider, service string) (map[string]string, bool)
	DeleteRegions(provider, service string)

	StoreZones(provider, service, region string, val []string)
	GetZones(provider, service, region string) ([]string, bool)
	DeleteZones(provider, service, region string)

	StorePrice(provider, region, instanceType string, val types.Price)
	GetPrice(provider, region, instanceType string) (types.Price, bool)

	StoreVm(provider, service, region string, val []types.VMInfo)
	GetVm(provider, service, region string) ([]types.VMInfo, bool)
	DeleteVm(provider, service, region string)

	StoreImage(provider, service, regionId string, val []types.Image)
	GetImage(provider, service, regionId string) ([]types.Image, bool)
	DeleteImage(provider, service, regionId string)

	StoreVersion(provider, service, region string, val []types.LocationVersion)
	GetVersion(provider, service, region string) ([]types.LocationVersion, bool)
	DeleteVersion(provider, service, region string)

	StoreStatus(provider string, val string)
	GetStatus(provider string) (string, bool)

	StoreServices(provider string, services []types.Service)
	GetServices(provider string) ([]types.Service, bool)

	Export(w io.Writer) error
	Import(r io.Reader) error

	Close()
}
