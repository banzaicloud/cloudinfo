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
	"fmt"
	"io"
	"time"

	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/patrickmn/go-cache"
)

const (
	// VmKeyTemplate format for generating vm cache keys
	VmKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/vms"

	// AttrKeyTemplate format for generating attribute cache keys
	AttrKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/attrValues/%s"

	// PriceKeyTemplate format for generating price cache keys
	PriceKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/regions/%s/prices/%s"

	// ZoneKeyTemplate format for generating zone cache keys
	ZoneKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/regions/%s/zones/"

	// RegionKeyTemplate format for generating region cache keys
	RegionKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/"

	// StatusKeyTemplate format for generating status cache keys
	StatusKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/status/"

	// ImageKeyTemplate format for generating image cache keys
	ImageKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/images"

	// VersionKeyTemplate format for generating kubernetes version cache keys
	VersionKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/versions"
)

// Storage operations for cloud information
type CloudInfoStore interface {
	StoreRegions(provider, service string, val interface{})
	GetRegions(provider, service string) (interface{}, bool)

	StoreZones(provider, region string, val interface{})
	GetZones(provider, region string) (interface{}, bool)

	StorePrice(provider, region, instanceType string, val interface{})
	GetPrice(provider, region, instanceType string) (interface{}, bool)

	StoreAttribute(provider, service, attribute string, val interface{})
	GetAttribute(provider, service, attribute string) (interface{}, bool)

	StoreVm(provider, service, region string, val interface{})
	GetVm(provider, service, region string) (interface{}, bool)

	StoreImage(provider, service, regionId string, val interface{})
	GetImage(provider, service, regionId string) (interface{}, bool)

	StoreVersion(provider, service, region string, val interface{})
	GetVersion(provider, service, region string) (interface{}, bool)

	StoreStatus(provider string, val interface{})
	GetStatus(provider string) (interface{}, bool)

	Export(w io.Writer) error
	Import(r io.Reader) error
}

// cacheProductStore in memory cloud product information storer
type cacheProductStore struct {
	*cache.Cache
	// all items are cached with this expiry
	itemExpiry time.Duration
	log        logur.Logger
}

// Export writes the content of the store into the passed in writer
func (cis *cacheProductStore) Export(w io.Writer) error {
	if err := cis.Save(w); err != nil {
		cis.log.Error("failed to export the store", map[string]interface{}{"op": "export", "destination": "todo"})
		return emperror.WrapWith(err, "failed to export the store", "op", "export", "destination", "todo")
	}
	return nil
}

// Import loads the store data from the standard input
func (cis *cacheProductStore) Import(r io.Reader) error {
	if err := cis.Load(r); err != nil {
		cis.log.Error("failed to load store data", map[string]interface{}{"op": "import", "destination": "todo"})
		return emperror.WrapWith(err, "failed to load the store data", "op", "import", "destination", "todo")
	}
	return nil
}

func (cis *cacheProductStore) StoreRegions(provider, service string, val interface{}) {
	cis.Set(cis.getKey(RegionKeyTemplate, provider, service), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetRegions(provider, service string) (interface{}, bool) {
	return cis.Get(cis.getKey(RegionKeyTemplate, provider, service))
}

func (cis *cacheProductStore) StoreZones(provider, region string, val interface{}) {
	cis.Set(cis.getKey(ZoneKeyTemplate, provider, region), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetZones(provider, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(ZoneKeyTemplate, provider, region))
}

func (cis *cacheProductStore) StorePrice(provider, region, instanceType string, val interface{}) {
	cis.Set(cis.getKey(PriceKeyTemplate, provider, region, instanceType), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetPrice(provider, region, instanceType string) (interface{}, bool) {
	return cis.Get(cis.getKey(PriceKeyTemplate, provider, region, instanceType))
}

func (cis *cacheProductStore) StoreAttribute(provider, service, attribute string, val interface{}) {
	cis.Set(cis.getKey(AttrKeyTemplate, provider, service, attribute), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetAttribute(provider, service, attribute string) (interface{}, bool) {
	return cis.Get(cis.getKey(AttrKeyTemplate, provider, service, attribute))
}

func (cis *cacheProductStore) StoreVm(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(VmKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetVm(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(VmKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) StoreImage(provider, service, regionId string, val interface{}) {
	cis.Set(cis.getKey(ImageKeyTemplate, provider, service, regionId), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetImage(provider, service, regionId string) (interface{}, bool) {
	return cis.Get(cis.getKey(ImageKeyTemplate, provider, service, regionId))
}

func (cis *cacheProductStore) StoreVersion(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(VersionKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetVersion(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(VersionKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) StoreStatus(provider string, val interface{}) {
	cis.Set(cis.getKey(StatusKeyTemplate, provider), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetStatus(provider string) (interface{}, bool) {
	return cis.Get(cis.getKey(StatusKeyTemplate, provider))
}

// NewCacheProductStore creates a new store instance.
// the backing cache is initialized with the defaultExpiration and cleanupInterval
func NewCacheProductStore(cloudInfoExpiration, cleanupInterval time.Duration, logger logur.Logger) CloudInfoStore {
	return &cacheProductStore{
		cache.New(cloudInfoExpiration, cleanupInterval),
		cleanupInterval,
		logger,
	}
}

func (cis *cacheProductStore) getKey(keyTemplate string, args ...string) string {
	return fmt.Sprintf(keyTemplate, args)
}
