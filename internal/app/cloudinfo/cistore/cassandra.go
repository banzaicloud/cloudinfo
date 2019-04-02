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
	"io"

	"github.com/banzaicloud/cloudinfo/internal/platform/cassandra"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/gocql/gocql"
	"github.com/goph/logur"
)

type cassandraProductStore struct {
	log        logur.Logger
	clusterCfg *gocql.ClusterConfig
}

func (cps *cassandraProductStore) StoreRegions(provider, service string, val map[string]string) {
	panic("implement me")
}

func (cps *cassandraProductStore) GetRegions(provider, service string) (map[string]string, bool) {
	panic("implement me")
}

func (cps *cassandraProductStore) DeleteRegions(provider, service string) {
	panic("implement me")
}

func (cps *cassandraProductStore) StoreZones(provider, service, region string, val []string) {
	panic("implement me")
}

func (cps *cassandraProductStore) GetZones(provider, service, region string) ([]string, bool) {
	panic("implement me")
}

func (cps *cassandraProductStore) DeleteZones(provider, service, region string) {
	panic("implement me")
}

func (cps *cassandraProductStore) StorePrice(provider, region, instanceType string, val cloudinfo.Price) {
	panic("implement me")
}

func (cps *cassandraProductStore) GetPrice(provider, region, instanceType string) (cloudinfo.Price, bool) {
	panic("implement me")
}

func (cps *cassandraProductStore) StoreVm(provider, service, region string, val []cloudinfo.VmInfo) {
	panic("implement me")
}

func (cps *cassandraProductStore) GetVm(provider, service, region string) ([]cloudinfo.VmInfo, bool) {
	panic("implement me")
}

func (cps *cassandraProductStore) DeleteVm(provider, service, region string) {
	panic("implement me")
}

func (cps *cassandraProductStore) StoreImage(provider, service, regionId string, val []cloudinfo.Image) {
	panic("implement me")
}

func (cps *cassandraProductStore) GetImage(provider, service, regionId string) ([]cloudinfo.Image, bool) {
	panic("implement me")
}

func (cps *cassandraProductStore) DeleteImage(provider, service, regionId string) {
	panic("implement me")
}

func (cps *cassandraProductStore) StoreVersion(provider, service, region string, val []cloudinfo.LocationVersion) {
	panic("implement me")
}

func (cps *cassandraProductStore) GetVersion(provider, service, region string) ([]cloudinfo.LocationVersion, bool) {
	panic("implement me")
}

func (cps *cassandraProductStore) DeleteVersion(provider, service, region string) {
	panic("implement me")
}

func (cps *cassandraProductStore) StoreStatus(provider string, val string) {
	panic("implement me")
}

func (cps *cassandraProductStore) GetStatus(provider string) (string, bool) {
	panic("implement me")
}

func (cps *cassandraProductStore) StoreServices(provider string, services []cloudinfo.Service) {
	panic("implement me")
}

func (cps *cassandraProductStore) GetServices(provider string) ([]cloudinfo.Service, bool) {
	panic("implement me")
}

func (cps *cassandraProductStore) Export(w io.Writer) error {
	panic("implement me")
}

func (cps *cassandraProductStore) Import(r io.Reader) error {
	panic("implement me")
}

func NewCassandraProductStore(config cassandra.Config, logger logur.Logger) cloudinfo.CloudInfoStore {
	// todo initialize cassandra if necessary, simply connect to it otherwise
	clusterCfg := cassandra.NewCluster(config)
	return &cassandraProductStore{
		log:        logur.WithFields(logger, map[string]interface{}{"cistore": "cassandra"}),
		clusterCfg: clusterCfg,
	}
}
