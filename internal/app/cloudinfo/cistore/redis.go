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

func (rps *redisProductStore) DeleteRegions(provider, service string) {
	rps.delete(rps.getKey(cloudinfo.RegionKeyTemplate, provider, service))
}

func (rps *redisProductStore) DeleteZones(provider, service, region string) {
	rps.delete(rps.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region))
}

func (rps *redisProductStore) DeleteImage(provider, service, regionId string) {
	rps.delete(rps.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId))
}

func (rps *redisProductStore) DeleteVersion(provider, service, region string) {
	rps.delete(rps.getKey(cloudinfo.VersionKeyTemplate, provider, service, region))
}

// get retrieves the value of the passed in key in it's raw format
func (rps *redisProductStore) get(key string, toTypePtr interface{}) (interface{}, bool) {
	conn := rps.pool.Get()
	defer conn.Close()

	var (
		cachedJson interface{}
		err        error
	)

	if cachedJson, err = conn.Do("GET", key); err != nil {
		rps.log.Debug("failed to get entry", map[string]interface{}{"key": key})
		return nil, false
	}

	if cachedJson == nil {
		rps.log.Debug("nil value for key", map[string]interface{}{"key": key})
		return nil, false
	}

	// unmarshal the cache value into th desired struct
	if err = json.Unmarshal(cachedJson.([]byte), toTypePtr); err != nil {
		rps.log.Debug("failed to unmarshal cache entry", map[string]interface{}{"val": cachedJson})
		return nil, false
	}

	return &toTypePtr, true
}

// set sets the value of the given key to the json representation of the value
func (rps *redisProductStore) set(key string, value interface{}) (interface{}, bool) {
	conn := rps.pool.Get()
	defer conn.Close()

	var (
		mJson []byte
		err   error
	)

	// marshal the value into a json representation
	if mJson, err = json.Marshal(value); err != nil {
		rps.log.Debug("failed to marshal value into json", map[string]interface{}{"key": key, "value": value})
		return nil, false
	}

	if _, err = conn.Do("SET", key, mJson); err != nil {
		rps.log.Debug("failed to set key to value", map[string]interface{}{"key": key, "value": value})
		return nil, false
	}

	return mJson, true
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

func (rps *redisProductStore) StoreRegions(provider, service string, val map[string]string) {
	rps.set(rps.getKey(cloudinfo.RegionKeyTemplate, provider, service), val)
}

func (rps *redisProductStore) GetRegions(provider, service string) (map[string]string, bool) {
	var (
		res = make(map[string]string)
	)
	_, ok := rps.get(rps.getKey(cloudinfo.RegionKeyTemplate, provider, service), &res)

	return res, ok
}

func (rps *redisProductStore) StoreZones(provider, service, region string, val []string) {
	rps.set(rps.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region), val)
}

func (rps *redisProductStore) GetZones(provider, service, region string) ([]string, bool) {
	var (
		res = make([]string, 0)
	)

	_, ok := rps.get(rps.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region), &res)
	return res, ok
}

func (rps *redisProductStore) StorePrice(provider, region, instanceType string, val cloudinfo.Price) {
	rps.set(rps.getKey(cloudinfo.PriceKeyTemplate, provider, region, instanceType), val)
}

func (rps *redisProductStore) GetPrice(provider, region, instanceType string) (cloudinfo.Price, bool) {
	var (
		res = cloudinfo.Price{}
	)
	_, ok := rps.get(rps.getKey(cloudinfo.PriceKeyTemplate, provider, region, instanceType), &res)

	return res, ok
}

func (rps *redisProductStore) StoreVm(provider, service, region string, val []cloudinfo.VmInfo) {
	rps.set(rps.getKey(cloudinfo.VmKeyTemplate, provider, service, region), val)
}

func (rps *redisProductStore) GetVm(provider, service, region string) ([]cloudinfo.VmInfo, bool) {
	var (
		res = make([]cloudinfo.VmInfo, 0)
	)
	_, ok := rps.get(rps.getKey(cloudinfo.VmKeyTemplate, provider, service, region), &res)

	return res, ok
}

func (rps *redisProductStore) DeleteVm(provider, service, region string) {
	rps.delete(rps.getKey(cloudinfo.VmKeyTemplate, provider, service, region))
}

func (rps *redisProductStore) StoreImage(provider, service, regionId string, val []cloudinfo.Image) {
	rps.set(rps.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId), val)
}

func (rps *redisProductStore) GetImage(provider, service, regionId string) ([]cloudinfo.Image, bool) {
	var (
		res = make([]cloudinfo.Image, 0)
	)
	_, ok := rps.get(rps.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId), &res)

	return res, ok
}

func (rps *redisProductStore) StoreVersion(provider, service, region string, val []cloudinfo.LocationVersion) {
	rps.set(rps.getKey(cloudinfo.VersionKeyTemplate, provider, service, region), val)
}

func (rps *redisProductStore) GetVersion(provider, service, region string) ([]cloudinfo.LocationVersion, bool) {
	var (
		res = make([]cloudinfo.LocationVersion, 0)
	)
	_, ok := rps.get(rps.getKey(cloudinfo.VersionKeyTemplate, provider, service, region), &res)

	return res, ok
}

func (rps *redisProductStore) StoreStatus(provider string, val string) {
	rps.set(rps.getKey(cloudinfo.StatusKeyTemplate, provider), val)
}

func (rps *redisProductStore) GetStatus(provider string) (string, bool) {
	var (
		res string
	)
	_, ok := rps.get(rps.getKey(cloudinfo.StatusKeyTemplate, provider), &res)

	return res, ok
}

func (rps *redisProductStore) StoreServices(provider string, services []cloudinfo.Service) {
	rps.set(rps.getKey(cloudinfo.ServicesKeyTemplate, provider), services)
}

func (rps *redisProductStore) GetServices(provider string) ([]cloudinfo.Service, bool) {
	var (
		res = make([]cloudinfo.Service, 0)
	)
	_, ok := rps.get(rps.getKey(cloudinfo.ServicesKeyTemplate, provider), &res)

	return res, ok
}

func (rps *redisProductStore) getKey(keyTemplate string, args ...interface{}) string {
	key := fmt.Sprintf(keyTemplate, args...)

	return key
}
