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
	// vmKeyTemplate format for generating vm cache keys
	vmKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/vms"

	// priceKeyTemplate format for generating price cache keys
	priceKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/regions/%s/prices/%s"

	// zoneKeyTemplate format for generating zone cache keys
	zoneKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/zones/"

	// regionKeyTemplate format for generating region cache keys
	regionKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/"

	// statusKeyTemplate format for generating status cache keys
	statusKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/status/"

	// imageKeyTemplate format for generating image cache keys
	imageKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/images"

	// versionKeyTemplate format for generating kubernetes version cache keys
	versionKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/versions"

	// servicesKeyTemplate key for storing provider specific services
	servicesKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services"
)

// Storage operations for cloud information
type CloudInfoStore interface {
	StoreRegions(provider, service string, val interface{})
	GetRegions(provider, service string) (interface{}, bool)

	StoreZones(provider, service, region string, val interface{})
	GetZones(provider, service, region string) (interface{}, bool)

	StorePrice(provider, region, instanceType string, val interface{})
	GetPrice(provider, region, instanceType string) (interface{}, bool)

	StoreVm(provider, service, region string, val interface{})
	GetVm(provider, service, region string) (interface{}, bool)
	DeleteVm(provider, service, region string)

	StoreImage(provider, service, regionId string, val interface{})
	GetImage(provider, service, regionId string) (interface{}, bool)

	StoreVersion(provider, service, region string, val interface{})
	GetVersion(provider, service, region string) (interface{}, bool)

	StoreStatus(provider string, val interface{})
	GetStatus(provider string) (interface{}, bool)

	StoreServices(provider string, services interface{})
	GetServices(provider string) (interface{}, bool)

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
	cis.Set(cis.getKey(regionKeyTemplate, provider, service), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetRegions(provider, service string) (interface{}, bool) {
	return cis.Get(cis.getKey(regionKeyTemplate, provider, service))
}

func (cis *cacheProductStore) StoreZones(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(zoneKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetZones(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(zoneKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) StorePrice(provider, region, instanceType string, val interface{}) {
	cis.Set(cis.getKey(priceKeyTemplate, provider, region, instanceType), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetPrice(provider, region, instanceType string) (interface{}, bool) {
	return cis.Get(cis.getKey(priceKeyTemplate, provider, region, instanceType))
}

func (cis *cacheProductStore) StoreVm(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(vmKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetVm(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(vmKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) DeleteVm(provider, service, region string) {
	cis.Delete(cis.getKey(vmKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) StoreImage(provider, service, regionId string, val interface{}) {
	cis.Set(cis.getKey(imageKeyTemplate, provider, service, regionId), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetImage(provider, service, regionId string) (interface{}, bool) {
	return cis.Get(cis.getKey(imageKeyTemplate, provider, service, regionId))
}

func (cis *cacheProductStore) StoreVersion(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(versionKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetVersion(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(versionKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) StoreStatus(provider string, val interface{}) {
	cis.Set(cis.getKey(statusKeyTemplate, provider), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetStatus(provider string) (interface{}, bool) {
	return cis.Get(cis.getKey(statusKeyTemplate, provider))
}

func (cis *cacheProductStore) StoreServices(provider string, services interface{}) {
	cis.Set(cis.getKey(servicesKeyTemplate, provider), services, cis.itemExpiry)
}

func (cis *cacheProductStore) GetServices(provider string) (interface{}, bool) {
	return cis.Get(cis.getKey(servicesKeyTemplate, provider))
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
