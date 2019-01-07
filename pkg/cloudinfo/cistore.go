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

// ProductStorer contract for storing and retrieving cloud product information
type ProductStorer interface {
	Get(k string) (interface{}, bool)
	Set(k string, x interface{}, d time.Duration)
}

// Storage operations for cloud information
type CloudInfoStore interface {
	StoreRegion(provider, service string, val interface{})
	GetRegion(provider, service string) (interface{}, bool)

	StoreZone(provider string, region string, val interface{})
	GetZone(provider string, region string) (interface{}, bool)

	StorePrice(provider string, region string, instanceType string, val interface{})
	GetPrice(provider string, region string, instanceType string) (interface{}, bool)

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
}

func (cis *CacheProductStore) StoreRegion(provider, service string, val interface{}) {
	cis.Set(cis.getKey(RegionKeyTemplate, provider, service), val, 0)
}

func (cis *CacheProductStore) GetRegion(provider, service string) (interface{}, bool) {
	return cis.Get(cis.getKey(RegionKeyTemplate, provider, service))
}

func (cis *CacheProductStore) StoreZone(provider string, region string, val interface{}) {
	cis.Set(cis.getKey(ZoneKeyTemplate, provider, region), val, 0)
}

func (cis *CacheProductStore) GetZone(provider string, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(ZoneKeyTemplate))
}

func (cis *CacheProductStore) StorePrice(provider string, region string, instanceType string, val interface{}) {
	cis.Set(cis.getKey(PriceKeyTemplate, provider, region, instanceType), val, 0)
}

func (cis *CacheProductStore) GetPrice(provider string, region string, instanceType string) (interface{}, bool) {
	return cis.Get(cis.getKey(PriceKeyTemplate))
}

func (cis *CacheProductStore) StoreAttribute(provider, service, attribute string, val interface{}) {
	cis.Set(cis.getKey(AttrKeyTemplate, provider, service, attribute), val, 0)
}

func (cis *CacheProductStore) GetAttribute(provider, service, attribute string) (interface{}, bool) {
	return cis.Get(cis.getKey(AttrKeyTemplate))
}

func (cis *CacheProductStore) StoreVm(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(VmKeyTemplate, provider, service, region), val, 0)
}

func (cis *CacheProductStore) GetVm(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(VmKeyTemplate))
}

func (cis *CacheProductStore) StoreImage(provider, service, regionId string, val interface{}) {
	cis.Set(cis.getKey(ImageKeyTemplate, provider, service, regionId), val, 0)
}

func (cis *CacheProductStore) GetImage(provider, service, regionId string) (interface{}, bool) {
	return cis.Get(cis.getKey(ImageKeyTemplate))
}

func (cis *CacheProductStore) StoreVersion(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(VersionKeyTemplate, provider, service, region), val, 0)
}

func (cis *CacheProductStore) GetVersion(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(VersionKeyTemplate))
}

func (cis *CacheProductStore) StoreStatus(provider string, val interface{}) {
	cis.Set(cis.getKey(StatusKeyTemplate, provider), val, 0)
}

func (cis *CacheProductStore) GetStatus(provider string) (interface{}, bool) {
	return cis.Get(cis.getKey(StatusKeyTemplate))
}

// todo default expiration!!!
func NewCacheProductStorer(cleanupInterval time.Duration) CloudInfoStore {
	return &CacheProductStore{
		cache.New(cache.NoExpiration, cleanupInterval),
	}
}

func (cis *CacheProductStore) getKey(keyTemplate string, args ... string) string {
	return fmt.Sprintf(keyTemplate, args)
}
