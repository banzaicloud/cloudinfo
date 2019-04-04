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

	"github.com/banzaicloud/cloudinfo/internal/platform/cassandra"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/gocql/gocql"
	"github.com/goph/emperror"
	"github.com/goph/logur"
)

type cassandraProductStore struct {
	log       logur.Logger
	keySpace  string
	tableName string
	session   *gocql.Session
}

func NewCassandraProductStore(config cassandra.Config, logger logur.Logger) cloudinfo.CloudInfoStore {

	var (
		session *gocql.Session
		err     error
	)
	clusterCfg := cassandra.NewCluster(config)

	if session, err = clusterCfg.CreateSession(); err != nil {
		emperror.Panic(err)
	}
	//defer session.Close()

	// init cassandra store
	keyspaceQuery := fmt.Sprintf("CREATE KEYSPACE IF NOT EXISTS %s WITH replication = {'class': 'SimpleStrategy', 'replication_factor' : 1}", config.Keyspace)
	tableQuery := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s.%s (	key text, value text, PRIMARY KEY(key))", config.Keyspace, config.Table)

	if e := session.Query(keyspaceQuery).Exec(); e != nil {
		logger.Error("failed to create keyspace", map[string]interface{}{"cistore": "cassandra"})
		emperror.Panic(err)
	}

	if e := session.Query(tableQuery).Exec(); e != nil {
		logger.Error("failed to create product table", map[string]interface{}{"cistore": "cassandra"})
		emperror.Panic(err)
	}

	return &cassandraProductStore{
		log:       logur.WithFields(logger, map[string]interface{}{"cistore": "cassandra"}),
		keySpace:  config.Keyspace,
		tableName: config.Table,
		session:   session,
	}
}

func (cps *cassandraProductStore) StoreRegions(provider, service string, val map[string]string) {
	cps.set(cps.getKey(cloudinfo.RegionKeyTemplate, provider, service), val)
}

func (cps *cassandraProductStore) GetRegions(provider, service string) (map[string]string, bool) {
	var (
		res = make(map[string]string)
	)
	_, ok := cps.get(cps.getKey(cloudinfo.RegionKeyTemplate, provider, service), &res)

	return res, ok

}

func (cps *cassandraProductStore) DeleteRegions(provider, service string) {
	cps.delete(cps.getKey(cloudinfo.RegionKeyTemplate, provider, service))
}

func (cps *cassandraProductStore) StoreZones(provider, service, region string, val []string) {
	cps.set(cps.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region), val)
}

func (cps *cassandraProductStore) GetZones(provider, service, region string) ([]string, bool) {
	res := make([]string, 0)

	_, ok := cps.get(cps.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region), &res)
	return res, ok

}

func (cps *cassandraProductStore) DeleteZones(provider, service, region string) {
	cps.delete(cps.getKey(cloudinfo.ZoneKeyTemplate, provider, service, region))
}

func (cps *cassandraProductStore) StorePrice(provider, region, instanceType string, val cloudinfo.Price) {
	cps.set(cps.getKey(cloudinfo.PriceKeyTemplate, provider, region, instanceType), val)
}

func (cps *cassandraProductStore) GetPrice(provider, region, instanceType string) (cloudinfo.Price, bool) {
	var res cloudinfo.Price
	_, ok := cps.get(cps.getKey(cloudinfo.PriceKeyTemplate, provider, region, instanceType), &res)
	return res, ok
}

func (cps *cassandraProductStore) StoreVm(provider, service, region string, val []cloudinfo.VmInfo) {
	cps.set(cps.getKey(cloudinfo.VmKeyTemplate, provider, service, region), val)
}

func (cps *cassandraProductStore) GetVm(provider, service, region string) ([]cloudinfo.VmInfo, bool) {
	res := make([]cloudinfo.VmInfo, 0)
	_, ok := cps.get(cps.getKey(cloudinfo.VmKeyTemplate, provider, service, region), &res)
	return res, ok

}

func (cps *cassandraProductStore) DeleteVm(provider, service, region string) {
	cps.delete(cps.getKey(cloudinfo.VmKeyTemplate, provider, service, region))
}

func (cps *cassandraProductStore) StoreImage(provider, service, regionId string, val []cloudinfo.Image) {
	cps.set(cps.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId), val)
}

func (cps *cassandraProductStore) GetImage(provider, service, regionId string) ([]cloudinfo.Image, bool) {

	var (
		res = make([]cloudinfo.Image, 0)
	)

	cps.get(cps.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId), &res)

	return res, false
}

func (cps *cassandraProductStore) DeleteImage(provider, service, regionId string) {
	cps.delete(cps.getKey(cloudinfo.ImageKeyTemplate, provider, service, regionId))
}

func (cps *cassandraProductStore) StoreVersion(provider, service, region string, val []cloudinfo.LocationVersion) {
	cps.set(cps.getKey(cloudinfo.VersionKeyTemplate, provider, service, region), val)
}

func (cps *cassandraProductStore) GetVersion(provider, service, region string) ([]cloudinfo.LocationVersion, bool) {
	res := make([]cloudinfo.LocationVersion, 0)
	_, ok := cps.get(cps.getKey(cloudinfo.VersionKeyTemplate, provider, service, region), &res)
	return res, ok

}

func (cps *cassandraProductStore) DeleteVersion(provider, service, region string) {
	cps.delete(cps.getKey(cloudinfo.VersionKeyTemplate, provider, service, region))
}

func (cps *cassandraProductStore) StoreStatus(provider string, val string) {
	cps.set(cps.getKey(cloudinfo.StatusKeyTemplate, provider), val)
}

func (cps *cassandraProductStore) GetStatus(provider string) (string, bool) {
	var res string
	_, ok := cps.get(cps.getKey(cloudinfo.StatusKeyTemplate, provider), &res)
	return res, ok

}

func (cps *cassandraProductStore) StoreServices(provider string, services []cloudinfo.Service) {
	cps.set(cps.getKey(cloudinfo.ServicesKeyTemplate, provider), services)
}

func (cps *cassandraProductStore) GetServices(provider string) ([]cloudinfo.Service, bool) {
	res := make([]cloudinfo.Service, 0)
	_, ok := cps.get(cps.getKey(cloudinfo.ServicesKeyTemplate, provider), &res)
	return res, ok
}

func (cps *cassandraProductStore) Export(w io.Writer) error {
	panic("implement me")
}

func (cps *cassandraProductStore) Import(r io.Reader) error {
	panic("implement me")
}

func (cps *cassandraProductStore) getKey(keyTemplate string, args ...interface{}) string {
	key := fmt.Sprintf(keyTemplate, args...)

	return key
}

func (cps *cassandraProductStore) set(key string, value interface{}) (interface{}, bool) {
	//session, _ := cps.clusterCfg.CreateSession()
	//defer session.Close()

	var (
		mJson []byte
		err   error
	)

	// marshal the value into a json representation
	if mJson, err = json.Marshal(value); err != nil {
		cps.log.Debug("failed to marshal value into json", map[string]interface{}{"key": key, "value": value})
		return nil, false
	}

	ins := fmt.Sprintf("INSERT INTO %s.%s (key, value) VALUES (?, ?)", cps.keySpace, cps.tableName)
	if err = cps.session.Query(ins, key, mJson).Exec(); err != nil {
		cps.log.Debug("failed to save value", map[string]interface{}{"key": key, "value": value})
		return nil, false
	}

	return nil, true
}

// get retrieves the value of the passed in key in it's raw format
func (cps *cassandraProductStore) get(key string, toTypePtr interface{}) (interface{}, bool) {
	//session, _ := cps.clusterCfg.CreateSession()
	//defer session.Close()

	var (
		cachedJson string
		err        error
	)

	getQ := fmt.Sprintf("SELECT value FROM  %s.%s WHERE key = ?", cps.keySpace, cps.tableName)
	if err = cps.session.Query(getQ, key).Scan(&cachedJson); err != nil {
		cps.log.Debug("failed to get entry", map[string]interface{}{"key": key})
		return nil, false
	}

	if cachedJson == "" {
		cps.log.Debug("nil value for key", map[string]interface{}{"key": key})
		return nil, false
	}

	// unmarshal the cache value into th desired struct
	if err = json.Unmarshal([]byte(cachedJson), &toTypePtr); err != nil {
		cps.log.Debug("failed to unmarshal cache entry", map[string]interface{}{"val": cachedJson})
		return nil, false
	}

	return &toTypePtr, true
}

func (cps *cassandraProductStore) delete(key string) {
	//session, _ := cps.clusterCfg.CreateSession()
	//defer session.Close()

	delQ := fmt.Sprintf("DELETE FROM %s.%s WHERE key = ?", cps.keySpace, cps.tableName)
	if err := cps.session.Query(delQ, key).Exec(); err != nil {
		cps.log.Error("failed to delete key", map[string]interface{}{"key": key})
	}
}
