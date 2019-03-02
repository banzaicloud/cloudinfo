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
	"encoding/json"
	"fmt"
	"io"

	"github.com/banzaicloud/cloudinfo/internal/platform/redis"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/goph/logur"
)

type redisProductStore struct {
	pool *redigo.Pool
	log  logur.Logger
}

func (rps *redisProductStore) set(key string, value interface{}) {
	conn := rps.pool.Get()
	defer conn.Close()

	var (
		mJson []byte
		err   error
	)
	// marshal the value into a json representation
	if mJson, err = json.Marshal(value); err != nil {
		rps.log.Debug("failed to marshal values", map[string]interface{}{"key": key, "value": value})
	}

	if _, err = conn.Do("SET", key, mJson); err != nil {
		rps.log.Debug("failed to save entry", map[string]interface{}{"key": key, "value": value})
	}
}

func (rps *redisProductStore) get(key string) (interface{}, bool) {
	conn := rps.pool.Get()
	defer conn.Close()

	var (
		val interface{}
		err error
	)

	if val, err = conn.Do("GET", key); err != nil {
		rps.log.Debug("failed to get entry", map[string]interface{}{"key": key})
	}

	if val == nil {
		return nil, false
	}

	return val, true
}

func (rps *redisProductStore) getUnmarshalled(key string, toType interface{}) (interface{}, bool) {
	var (
		cacheJson interface{}
		ok        bool
	)

	if cacheJson, ok = rps.get(key); !ok {
		return nil, ok
	}

	if err := json.Unmarshal(cacheJson.([]byte), toType); err != nil {
		return nil, false
	}

	return toType, true

}

func (rps *redisProductStore) delete(key string) {
	conn := rps.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("DEL", key); err != nil {
		rps.log.Debug("failed to delete entry", map[string]interface{}{"key": key})
	}
}

func NewRedisProductStore(config redis.Config, log logur.Logger) cloudinfo.CloudInfoStore {
	pool := redis.NewPool(config)

	return &redisProductStore{
		pool: pool,
		log:  logur.WithFields(log, map[string]interface{}{"cistore": "redis"}),
	}
}

// Export writes the content of the store into the passed in writer
func (rps *redisProductStore) Export(w io.Writer) error {
	//todo implement the export
	return nil
}

// Import loads the store data from the standard input
func (rps *redisProductStore) Import(r io.Reader) error {
	//todo implement the import
	return nil
}

func (rps *redisProductStore) StoreRegions(provider, service string, val interface{}) {
	rps.set(rps.getKey(cloudinfo.RegionKeyTemplate, provider, service), val)
}

func (rps *redisProductStore) GetRegions(provider, service string) (interface{}, bool) {
	var res map[string]string
	_, ok := rps.getUnmarshalled(rps.getKey(cloudinfo.RegionKeyTemplate, provider, service), &res)
	return res, ok
}

func (rps *redisProductStore) StoreZones(provider, service, region string, val interface{}) {
	rps.set(rps.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region), val)
}

func (rps *redisProductStore) GetZones(provider, service, region string) (interface{}, bool) {
	raw, _ := rps.get(rps.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region))
	var z []string
	if err := json.Unmarshal(raw.([]byte), &z); err != nil {
		rps.log.Error("failed to get zones", map[string]interface{}{"key": rps.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region)})
	}
	return z, true
}

func (rps *redisProductStore) StorePrice(provider, region, instanceType string, val interface{}) {
	rps.set(rps.getKey(cloudinfo.PriceKeyTemplate, provider, region, instanceType), val)
}

func (rps *redisProductStore) GetPrice(provider, region, instanceType string) (interface{}, bool) {
	var price cloudinfo.Price
	_, ok := rps.getUnmarshalled(rps.getKey(cloudinfo.PriceKeyTemplate, provider, region, instanceType), &price)

	return price, ok
}

func (rps *redisProductStore) StoreVm(provider, service, region string, val interface{}) {
	rps.set(rps.getKey(cloudinfo.VmKeyTemplate, provider, service, region), val)
}

func (rps *redisProductStore) GetVm(provider, service, region string) (interface{}, bool) {
	var (
		err       error
		vms       []cloudinfo.VmInfo
		cacheJson interface{}
		found     bool
	)
	if cacheJson, found = rps.get(rps.getKey(cloudinfo.VmKeyTemplate, provider, service, region)); !found {
		rps.log.Debug("cache entry not found", map[string]interface{}{"key": rps.getKey(cloudinfo.VmKeyTemplate, provider, service, region)})
		return nil, false
	}

	if err = json.Unmarshal(cacheJson.([]byte), &vms); err != nil {
		rps.log.Error("failed to unmarshal cached data", map[string]interface{}{"key": rps.getKey(cloudinfo.VmKeyTemplate, provider, service, region)})
		return nil, false
	}

	return vms, true
}

func (rps *redisProductStore) DeleteVm(provider, service, region string) {
	rps.delete(rps.getKey(cloudinfo.VmKeyTemplate, provider, service, region))
}

func (rps *redisProductStore) StoreImage(provider, service, regionId string, val interface{}) {
	rps.set(rps.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId), val)
}

func (rps *redisProductStore) GetImage(provider, service, regionId string) (interface{}, bool) {
	var imgs []cloudinfo.Image
	_, ok := rps.getUnmarshalled(rps.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId), imgs)
	return imgs, ok
}

func (rps *redisProductStore) StoreVersion(provider, service, region string, val interface{}) {
	rps.set(rps.getKey(cloudinfo.VersionKeyTemplate, provider, service, region), val)
}

func (rps *redisProductStore) GetVersion(provider, service, region string) (interface{}, bool) {
	return rps.get(rps.getKey(cloudinfo.VersionKeyTemplate, provider, service, region))
}

func (rps *redisProductStore) StoreStatus(provider string, val interface{}) {
	rps.set(rps.getKey(cloudinfo.StatusKeyTemplate, provider), val)
}

func (rps *redisProductStore) GetStatus(provider string) (interface{}, bool) {
	var str string
	_, ok := rps.getUnmarshalled(rps.getKey(cloudinfo.StatusKeyTemplate, provider), &str)
	return str, ok
}

func (rps *redisProductStore) StoreServices(provider string, services interface{}) {
	rps.set(rps.getKey(cloudinfo.ServicesKeyTemplate, provider), services)
}

func (rps *redisProductStore) GetServices(provider string) (interface{}, bool) {
	svcs := make([]cloudinfo.Service, 0)
	_, ok := rps.getUnmarshalled(rps.getKey(cloudinfo.ServicesKeyTemplate, provider), &svcs)
	return svcs, ok
}

func (rps *redisProductStore) getKey(keyTemplate string, args ...interface{}) string {
	key := fmt.Sprintf(keyTemplate, args...)

	return key
}
