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
	LoadZones(ctx context.Context, provider string, service string, rd RegionData)
	LoadAttributes(ctx context.Context, provider string, service string, rd RegionData)
	LoadVersions(ctx context.Context, provider string, service string, rd RegionData)
	LoadImages(ctx context.Context, provider string, service string, rd RegionData)
	LoadVms(ctx context.Context, provider string, service string, rd RegionData)
	LoadPrices(ctx context.Context, provider string, service string, rd RegionData)
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
	for _, rd := range sd.Regions {
		regionMap[rd.RegionId] = rd.Region

		sl.LoadZones(ctx, sd.Provider, sd.Name, rd)

		sl.LoadAttributes(ctx, sd.Provider, sd.Name, rd)

		sl.LoadVersions(ctx, sd.Provider, sd.Name, rd)

		sl.LoadImages(ctx, sd.Provider, sd.Name, rd)

		sl.LoadVms(ctx, sd.Provider, sd.Name, rd)

		sl.LoadPrices(ctx, sd.Provider, sd.Name, rd)
	}

	sl.store.StoreRegions(sd.Provider, sd.Name, regionMap)
	sl.log.Debug("regions loaded")

	// set the status
	sl.store.StoreStatus(sd.Provider, strconv.Itoa(int(time.Now().UnixNano()/1e6)))
	sl.log.Debug("status updated")

}

// loadZones loads zones for a given region in the store
func (sl *defaultCloudInfoLoader) LoadZones(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading zones...")
	sl.store.StoreZones(provider, service, rd.RegionId, rd.Zones)
	sl.log.Debug("zones loaded")
}

// loadAttributes loads attribute values for a given region into the store
func (sl *defaultCloudInfoLoader) LoadAttributes(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading attributes...")
	for _, attr := range rd.Attributes {
		sl.store.StoreAttribute(provider, service, attr.Name, attr.Values)
	}
	sl.log.Debug("attributes loaded")
}

// loadVersions loads versions for a given region into the store
func (sl *defaultCloudInfoLoader) LoadVersions(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading versions...")
	sl.store.StoreVersion(provider, service, rd.RegionId, rd.Versions)
	sl.log.Debug("versions loaded")
}

// loadImages loads images for a given region into the store
func (sl *defaultCloudInfoLoader) LoadImages(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading images...")
	sl.store.StoreImage(provider, service, rd.RegionId, rd.Images)
	sl.log.Debug("images loaded")
}

// loadVms loads vms for a given region into the store
func (sl *defaultCloudInfoLoader) LoadVms(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading vms...")
	sl.store.StoreVm(provider, service, rd.RegionId, rd.Vms)
	sl.log.Debug("vms loaded")
}

// loadVms loads vms for a given region into the store
func (sl *defaultCloudInfoLoader) LoadPrices(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading prices...")

	for _, priceData := range rd.Prices {
		sl.store.StorePrice(provider, rd.RegionId, priceData.Instancetype, priceData.Price)
	}
	sl.log.Debug("prices loaded")
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
	for _, rd := range sd.Regions {
		regionMap[rd.RegionId] = rd.Region

		scil.LoadZones(ctx, sd.Provider, sd.Name, rd)

		scil.LoadAttributes(ctx, sd.Provider, sd.Name, rd)

		scil.LoadVersions(ctx, sd.Provider, sd.Name, rd)

		scil.LoadImages(ctx, sd.Provider, sd.Name, rd)

		scil.LoadVms(ctx, sd.Provider, sd.Name, rd)

		scil.LoadPrices(ctx, sd.Provider, sd.Name, rd)
	}

	scil.store.StoreRegions(sd.Provider, sd.Name, regionMap)
	scil.log.Debug("regions loaded")

	// set the status
	scil.store.StoreStatus(sd.Provider, strconv.Itoa(int(time.Now().UnixNano()/1e6)))
	scil.log.Debug("status updated")

}

func (scil *storeCloudInfoLoader) LoadZones(ctx context.Context, provider string, service string, rd RegionData) {

	scil.log.Debug("copying zones...")
	if zones, ok := scil.store.GetZones(provider, scil.serviceData.Source, rd.RegionId); ok {
		scil.store.StoreZones(provider, service, rd.RegionId, zones)
	}
	scil.log.Debug("zones copied")

}

func (scil *storeCloudInfoLoader) LoadAttributes(ctx context.Context, provider string, service string, rd RegionData) {
	scil.log.Debug("loading attributes...")
	for _, attr := range rd.Attributes {
		if attrs, ok := scil.store.GetAttribute(provider, scil.serviceData.Source, attr.Name); ok {
			scil.log.Debug("copy attributes from source", map[string]interface{}{"attribute": attr.Name})
			scil.store.StoreAttribute(provider, service, attr.Name, attrs)
			continue
		}
		scil.log.Warn("couldn't copy attributes from source", map[string]interface{}{"attribute": attr.Name})
	}
	scil.log.Debug("attributes loaded")
}

func (scil *storeCloudInfoLoader) LoadVersions(ctx context.Context, provider string, service string, rd RegionData) {
	scil.log.Debug("copying versions...")
	if versions, ok := scil.store.GetVersion(provider, scil.serviceData.Source, rd.RegionId); ok {
		scil.store.StoreVersion(provider, service, rd.RegionId, versions)
	}
	scil.log.Debug("versions copied")
}

func (scil *storeCloudInfoLoader) LoadImages(ctx context.Context, provider string, service string, rd RegionData) {
	scil.log.Debug("copying images...")
	if images, ok := scil.store.GetImage(provider, scil.serviceData.Source, rd.RegionId); ok {
		scil.store.StoreImage(provider, service, rd.RegionId, images)
	}
	scil.log.Debug("images copied")
}

func (scil *storeCloudInfoLoader) LoadVms(ctx context.Context, provider string, service string, rd RegionData) {
	scil.log.Debug("copying vms...")
	if vms, ok := scil.store.GetVm(provider, scil.serviceData.Source, rd.RegionId); ok {
		scil.store.StoreVm(provider, service, rd.RegionId, vms)
	}
	scil.log.Debug("vms copied")
}

func (scil *storeCloudInfoLoader) LoadPrices(ctx context.Context, provider string, service string, rd RegionData) {
	scil.log.Debug("copying prices...")
	// todo prices are not bound to services!!
	scil.log.Debug("prices loaded")
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
