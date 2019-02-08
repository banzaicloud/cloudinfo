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

// Loader abstracts the loading logic
// I'ts purpose is to provide the possibility to load information from various sources (but scraping)
type Loader interface {
	LoadServiceData(ctx context.Context)

	LoadServices(ctx context.Context)
}

// defaultServiceLoader component is in charge for loading service related information into the Cloud Information Store
// from yaml files in the filesystem
type defaultServiceLoader struct {
	// used to locate anf parse the data file
	viper *viper.Viper

	// the destination of the loaded data
	store cloudinfo.CloudInfoStore
	log   logur.Logger
}

func (sl *defaultServiceLoader) LoadServices(ctx context.Context) {
	sl.log.Info("initializing services for providers...s")
	sl.directViper(ctx, "services")
	var (
		sds map[string][]Service
	)

	if err := sl.viper.Unmarshal(&sds); err != nil {
		sl.log.Error("failed to load service data")
		emperror.Panic(err)
	}

	for p, psvcs := range sds {
		var svcs []cloudinfo.Service
		for _, psvc := range psvcs {
			svcs = append(svcs, cloudinfo.Service{Service: psvc.Name, IsStatic: psvc.IsStatic})
		}
		sl.store.StoreServices(p, svcs)
	}

	sl.log.Info("services initialized")
}

// Load entry point to the service loading logic
func (sl *defaultServiceLoader) LoadServiceData(ctx context.Context) {

	sl.log.Info("loading service information...")

	sl.directViper(ctx, "service-definition")
	var sds []ServiceData

	if err := sl.viper.Unmarshal(&sds); err != nil {
		sl.log.Error("failed to load service data")
		emperror.Panic(err)
	}

	for _, sd := range sds {
		if err := sl.loadRegions(ctx, sd); err != nil {
			sl.log.Error("failed to load regions", map[string]interface{}{"svc": sd.Name})
		}
	}

	sl.log.Info("service information loaded")
}

// loadRegions loads regions in the cloud info store
func (sl *defaultServiceLoader) loadRegions(ctx context.Context, sd ServiceData) error {
	sl.log.Debug("loading regions...")

	regionMap := make(map[string]string)
	for _, rd := range sd.Regions {
		regionMap[rd.RegionId] = rd.Region

		sl.loadZones(ctx, sd.Provider, rd)

		sl.loadAttributes(ctx, sd.Provider, sd.Name, rd)

		sl.loadVersions(ctx, sd.Provider, sd.Name, rd)

		sl.loadImages(ctx, sd.Provider, sd.Name, rd)

		sl.loadVms(ctx, sd.Provider, sd.Name, rd)

		sl.loadPrices(ctx, sd.Provider, sd.Name, rd)
	}

	sl.store.StoreRegions(sd.Provider, sd.Name, regionMap)
	sl.log.Debug("regions loaded")

	// set the status
	sl.store.StoreStatus(sd.Provider, strconv.Itoa(int(time.Now().UnixNano()/1e6)))
	sl.log.Debug("status updated")

	return nil
}

// loadZones loads zones for a given region in the store
func (sl *defaultServiceLoader) loadZones(ctx context.Context, provider string, rd RegionData) {
	sl.log.Debug("loading zones...")
	sl.store.StoreZones(provider, rd.RegionId, rd.Zones)
	sl.log.Debug("zones loaded")
}

// loadAttributes loads attribute values for a given region into the store
func (sl *defaultServiceLoader) loadAttributes(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading attributes...")
	for _, attr := range rd.Attributes {
		sl.store.StoreAttribute(provider, service, attr.Name, attr.Values)
	}
	sl.log.Debug("attributes loaded")
}

// loadVersions loads versions for a given region into the store
func (sl *defaultServiceLoader) loadVersions(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading versions...")
	sl.store.StoreVersion(provider, service, rd.RegionId, rd.Versions)
	sl.log.Debug("versions loaded")
}

// loadImages loads images for a given region into the store
func (sl *defaultServiceLoader) loadImages(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading images...")
	sl.store.StoreImage(provider, service, rd.RegionId, rd.Images)
	sl.log.Debug("images loaded")
}

// loadVms loads vms for a given region into the store
func (sl *defaultServiceLoader) loadVms(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading vms...")
	sl.store.StoreVm(provider, service, rd.RegionId, rd.Vms)
	sl.log.Debug("vms loaded")
}

// loadVms loads vms for a given region into the store
func (sl *defaultServiceLoader) loadPrices(ctx context.Context, provider string, service string, rd RegionData) {
	sl.log.Debug("loading prices...")

	for _, priceData := range rd.Prices {
		sl.store.StorePrice(provider, rd.RegionId, priceData.Instancetype, priceData.Price)
	}
	sl.log.Debug("prices loaded")
}

// loadVms loads vms for a given region into the store
func (sl *defaultServiceLoader) directViper(ctx context.Context, file string) {
	sl.viper.SetConfigName(file)

	if err := sl.viper.ReadInConfig(); err != nil { // Find and read the config file
		// Handle errors reading the config file
		emperror.Panic(err)
	}
}

// NewDefaultServiceLoader sets up a new serviceloader
func NewDefaultServiceLoader(config Config, store cloudinfo.CloudInfoStore, log logur.Logger) Loader {
	// using a viper instance for loading data
	vp := viper.New()

	vp.AddConfigPath(config.SvcDataLocation)        // path to look for the config file in
	vp.AddConfigPath(config.SvcDefinitionsLocation) // path to look for the config file in
	vp.SetConfigType(config.Format)

	return &defaultServiceLoader{
		viper: vp,
		store: store,
		log:   logur.WithFields(log, map[string]interface{}{"service": "svcloader"}),
	}
}
