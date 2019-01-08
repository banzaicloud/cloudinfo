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
	"github.com/patrickmn/go-cache"
	"time"
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
	StoreRegion(provider, service string, val interface{})
	GetRegion(provider, service string) (interface{}, bool)

	StoreZone(provider, region string, val interface{})
	GetZone(provider, region string) (interface{}, bool)

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
}

// CacheProductStore in memory cloud product information storer
type CacheProductStore struct {
	*cache.Cache
	// all items are cached with this expiry
	itemExpiry time.Duration
}

func (cis *CacheProductStore) StoreRegion(provider, service string, val interface{}) {
	cis.Set(cis.getKey(RegionKeyTemplate, provider, service), val, cis.itemExpiry)
}

func (cis *CacheProductStore) GetRegion(provider, service string) (interface{}, bool) {
	return cis.Get(cis.getKey(RegionKeyTemplate, provider, service))
}

func (cis *CacheProductStore) StoreZone(provider, region string, val interface{}) {
	cis.Set(cis.getKey(ZoneKeyTemplate, provider, region), val, cis.itemExpiry)
}

func (cis *CacheProductStore) GetZone(provider, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(ZoneKeyTemplate, provider, region))
}

func (cis *CacheProductStore) StorePrice(provider, region, instanceType string, val interface{}) {
	cis.Set(cis.getKey(PriceKeyTemplate, provider, region, instanceType), val, cis.itemExpiry)
}

func (cis *CacheProductStore) GetPrice(provider, region, instanceType string) (interface{}, bool) {
	return cis.Get(cis.getKey(PriceKeyTemplate, provider, region, instanceType))
}

func (cis *CacheProductStore) StoreAttribute(provider, service, attribute string, val interface{}) {
	cis.Set(cis.getKey(AttrKeyTemplate, provider, service, attribute), val, cis.itemExpiry)
}

func (cis *CacheProductStore) GetAttribute(provider, service, attribute string) (interface{}, bool) {
	return cis.Get(cis.getKey(AttrKeyTemplate, provider, service, attribute))
}

func (cis *CacheProductStore) StoreVm(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(VmKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *CacheProductStore) GetVm(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(VmKeyTemplate, provider, service, region))
}

func (cis *CacheProductStore) StoreImage(provider, service, regionId string, val interface{}) {
	cis.Set(cis.getKey(ImageKeyTemplate, provider, service, regionId), val, cis.itemExpiry)
}

func (cis *CacheProductStore) GetImage(provider, service, regionId string) (interface{}, bool) {
	return cis.Get(cis.getKey(ImageKeyTemplate, provider, service, regionId))
}

func (cis *CacheProductStore) StoreVersion(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(VersionKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *CacheProductStore) GetVersion(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(VersionKeyTemplate, provider, service, region))
}

func (cis *CacheProductStore) StoreStatus(provider string, val interface{}) {
	cis.Set(cis.getKey(StatusKeyTemplate, provider), val, cis.itemExpiry)
}

func (cis *CacheProductStore) GetStatus(provider string) (interface{}, bool) {
	return cis.Get(cis.getKey(StatusKeyTemplate, provider))
}

// NewCacheProductStore creates a new store instance.
// the backing cache is initialized with the defaultExpiration and cleanupInterval
func NewCacheProductStore(cloudInfoExpiration, cleanupInterval time.Duration) CloudInfoStore {
	return &CacheProductStore{
		cache.New(cloudInfoExpiration, cleanupInterval),
		cleanupInterval,
	}
}

func (cis *CacheProductStore) getKey(keyTemplate string, args ...string) string {
	return fmt.Sprintf(keyTemplate, args)
}
