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

package cistore

import (
	"fmt"
	"io"
	"time"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/patrickmn/go-cache"
)

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
	cis.Set(cis.getKey(cloudinfo.RegionKeyTemplate, provider, service), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetRegions(provider, service string) (interface{}, bool) {
	return cis.Get(cis.getKey(cloudinfo.RegionKeyTemplate, provider, service))
}

func (cis *cacheProductStore) StoreZones(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetZones(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) StorePrice(provider, region, instanceType string, val interface{}) {
	cis.Set(cis.getKey(cloudinfo.PriceKeyTemplate, provider, region, instanceType), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetPrice(provider, region, instanceType string) (interface{}, bool) {
	return cis.Get(cis.getKey(cloudinfo.PriceKeyTemplate, provider, region, instanceType))
}

func (cis *cacheProductStore) StoreVm(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(cloudinfo.VmKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetVm(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(cloudinfo.VmKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) DeleteVm(provider, service, region string) {
	cis.Delete(cis.getKey(cloudinfo.VmKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) StoreImage(provider, service, regionId string, val interface{}) {
	cis.Set(cis.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetImage(provider, service, regionId string) (interface{}, bool) {
	return cis.Get(cis.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId))
}

func (cis *cacheProductStore) StoreVersion(provider, service, region string, val interface{}) {
	cis.Set(cis.getKey(cloudinfo.VersionKeyTemplate, provider, service, region), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetVersion(provider, service, region string) (interface{}, bool) {
	return cis.Get(cis.getKey(cloudinfo.VersionKeyTemplate, provider, service, region))
}

func (cis *cacheProductStore) StoreStatus(provider string, val interface{}) {
	cis.Set(cis.getKey(cloudinfo.StatusKeyTemplate, provider), val, cis.itemExpiry)
}

func (cis *cacheProductStore) GetStatus(provider string) (interface{}, bool) {
	return cis.Get(cis.getKey(cloudinfo.StatusKeyTemplate, provider))
}

func (cis *cacheProductStore) StoreServices(provider string, services interface{}) {
	cis.Set(cis.getKey(cloudinfo.ServicesKeyTemplate, provider), services, cis.itemExpiry)
}

func (cis *cacheProductStore) GetServices(provider string) (interface{}, bool) {
	return cis.Get(cis.getKey(cloudinfo.ServicesKeyTemplate, provider))
}

// NewCacheProductStore creates a new store instance.
// the backing cache is initialized with the defaultExpiration and cleanupInterval
func NewCacheProductStore(cloudInfoExpiration, cleanupInterval time.Duration, logger logur.Logger) cloudinfo.CloudInfoStore {
	return &cacheProductStore{
		cache.New(cloudInfoExpiration, cleanupInterval),
		cleanupInterval,
		logger,
	}
}

func (cis *cacheProductStore) getKey(keyTemplate string, args ...interface{}) string {
	return fmt.Sprintf(keyTemplate, args...)
}
