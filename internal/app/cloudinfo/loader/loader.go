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

package loader

import (
	"context"
	"strconv"
	"time"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/spf13/viper"
)

// CloudInfoLoader operations for loading cloud information into the cloud info application
// Add specialized implementations for different information sources
type CloudInfoLoader interface {
	LoadRegions(ctx context.Context, sd ServiceData)
	LoadZones(ctx context.Context, provider string, service string, region Region)
	LoadVersions(ctx context.Context, provider string, service string, region Region)
	LoadImages(ctx context.Context, provider string, service string, region Region)
	LoadVms(ctx context.Context, provider string, service string, region Region)
	Load(ctx context.Context)
}

// defaultCloudInfoLoader component is in charge for loading service related information into the Cloud Information Store
// from yaml files in the filesystem
type defaultCloudInfoLoader struct {
	// stores the data parsed from the data file
	serviceData ServiceData

	// the destination of the loaded data
	store cloudinfo.CloudInfoStore

	// component logger
	log logur.Logger
}

func (sl *defaultCloudInfoLoader) Load(ctx context.Context) {
	sl.LoadRegions(ctx, sl.serviceData)
}

// loadRegions loads regions in the cloud info store
func (sl *defaultCloudInfoLoader) LoadRegions(ctx context.Context, sd ServiceData) {
	sl.log.Debug("loading region data...")

	regionMap := make(map[string]string)
	for _, region := range sd.Regions {
		regionMap[region.Id] = region.Name

		sl.LoadZones(ctx, sd.Provider, sd.Name, region)

		sl.LoadVersions(ctx, sd.Provider, sd.Name, region)

		sl.LoadImages(ctx, sd.Provider, sd.Name, region)

		sl.LoadVms(ctx, sd.Provider, sd.Name, region)
	}

	sl.store.StoreRegions(sd.Provider, sd.Name, regionMap)
	sl.log.Debug("regions loaded")

	// set the status
	sl.store.StoreStatus(sd.Provider, strconv.Itoa(int(time.Now().UnixNano()/1e6)))
	sl.log.Debug("status updated")

}

// loadZones loads zones for a given region in the store
func (sl *defaultCloudInfoLoader) LoadZones(ctx context.Context, provider string, service string, region Region) {
	sl.log.Debug("loading zones...")
	sl.store.StoreZones(provider, service, region.Id, region.Data.Zones.Data)
	sl.log.Debug("zones loaded")
}

// loadVersions loads versions for a given region into the store
func (sl *defaultCloudInfoLoader) LoadVersions(ctx context.Context, provider string, service string, region Region) {
	sl.log.Debug("loading versions...")
	sl.store.StoreVersion(provider, service, region.Id, region.Data.Versions.Data)
	sl.log.Debug("versions loaded")
}

// loadImages loads images for a given region into the store
func (sl *defaultCloudInfoLoader) LoadImages(ctx context.Context, provider string, service string, region Region) {
	sl.log.Debug("loading images...")
	sl.store.StoreImage(provider, service, region.Id, region.Data.Images.Data)
	sl.log.Debug("images loaded")
}

// loadVms loads vms for a given region into the store
func (sl *defaultCloudInfoLoader) LoadVms(ctx context.Context, provider string, service string, region Region) {
	sl.log.Debug("loading vms...")
	sl.store.StoreVm(provider, service, region.Id, region.Data.Vms.Data)
	sl.log.Debug("vms loaded")
}

type storeCloudInfoLoader struct {
	store       cloudinfo.CloudInfoStore
	log         logur.Logger
	serviceData ServiceData
}

func (scil *storeCloudInfoLoader) Load(ctx context.Context) {
	scil.LoadRegions(ctx, scil.serviceData)
}

// loadRegions loads regions in the cloud info store
func (scil *storeCloudInfoLoader) LoadRegions(ctx context.Context, sd ServiceData) {
	scil.log.Debug("loading region data...")

	regionMap := make(map[string]string)
	for _, region := range sd.Regions {
		regionMap[region.Id] = region.Name

		scil.LoadZones(ctx, sd.Provider, sd.Name, region)

		scil.LoadVersions(ctx, sd.Provider, sd.Name, region)

		scil.LoadImages(ctx, sd.Provider, sd.Name, region)

		scil.LoadVms(ctx, sd.Provider, sd.Name, region)
	}

	scil.store.StoreRegions(sd.Provider, sd.Name, regionMap)
	scil.log.Debug("regions loaded")

	// set the status
	scil.store.StoreStatus(sd.Provider, strconv.Itoa(int(time.Now().UnixNano()/1e6)))
	scil.log.Debug("status updated")

}

func (scil *storeCloudInfoLoader) LoadZones(ctx context.Context, provider string, service string, region Region) {

	scil.log.Debug("copying zones...")
	if zones, ok := scil.store.GetZones(provider, scil.serviceData.Source, region.Id); ok {
		scil.store.StoreZones(provider, service, region.Id, zones)
	}
	scil.log.Debug("zones copied")

}

func (scil *storeCloudInfoLoader) LoadVersions(ctx context.Context, provider string, service string, region Region) {
	scil.log.Debug("copying versions...")
	if versions, ok := scil.store.GetVersion(provider, scil.serviceData.Source, region.Id); ok {
		scil.store.StoreVersion(provider, service, region.Id, versions)
	}
	scil.log.Debug("versions copied")
}

func (scil *storeCloudInfoLoader) LoadImages(ctx context.Context, provider string, service string, region Region) {
	scil.log.Debug("copying images...")
	if images, ok := scil.store.GetImage(provider, scil.serviceData.Source, region.Id); ok {
		scil.store.StoreImage(provider, service, region.Id, images)
	}
	scil.log.Debug("images copied")
}

func (scil *storeCloudInfoLoader) LoadVms(ctx context.Context, provider string, service string, region Region) {
	scil.log.Debug("copying vms...")
	if vms, ok := scil.store.GetVm(provider, scil.serviceData.Source, region.Id); ok {
		scil.store.StoreVm(provider, service, region.Id, vms)
	}
	scil.log.Debug("vms copied")
}

func NewCloudInfoLoader(datapath, datafile, datatype string, store cloudinfo.CloudInfoStore, log logur.Logger) CloudInfoLoader {

	dataViper := viper.New()
	dataViper.SetConfigName(datafile)
	dataViper.SetConfigType(datatype)
	dataViper.AddConfigPath(datapath)

	if err := dataViper.ReadInConfig(); err != nil { // Find and read the config file
		// Handle errors
		emperror.Panic(err)
	}

	var serviceData ServiceData
	if err := dataViper.Unmarshal(&serviceData); err != nil {
		emperror.Panic(err)
	}

	if serviceData.Source != "" {
		// serviceloader implementation that uses another service as source
		return &storeCloudInfoLoader{
			log:         logur.WithFields(log, map[string]interface{}{"component": "service-loader"}),
			store:       store,
			serviceData: serviceData,
		}
	}

	return &defaultCloudInfoLoader{
		serviceData: serviceData,
		store:       store,
		log:         logur.WithFields(log, map[string]interface{}{"component": "service-loader"}),
	}
}
