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
	"strconv"
	"time"

	"emperror.dev/emperror"
	"github.com/spf13/viper"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/messaging"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

// CloudInfoLoader operations for loading cloud information into the cloud info application
// Add specialized implementations for different information sources
type CloudInfoLoader interface {
	LoadRegions()
	LoadZones(provider string, service string, region Region)
	LoadVersions(provider string, service string, region Region)
	LoadImages(provider string, service string, region Region)
	LoadVms(provider string, service string, region Region)
	Load()
}

// defaultCloudInfoLoader component is in charge for loading service related information into the Cloud Information Store
// from yaml files in the filesystem
type defaultCloudInfoLoader struct {
	// stores the data parsed from the data file
	serviceData ServiceData

	// the destination of the loaded data
	store cloudinfo.CloudInfoStore

	// component logger
	log cloudinfo.Logger
}

func (dl *defaultCloudInfoLoader) Load() {
	dl.LoadRegions()
}

// loadRegions loads regions in the cloud info store
func (dl *defaultCloudInfoLoader) LoadRegions() {
	dl.log.Debug("loading region data...")

	regionMap := make(map[string]string)
	for _, region := range dl.serviceData.Regions {
		regionMap[region.Id] = region.Name

		dl.LoadZones(dl.serviceData.Provider, dl.serviceData.Name, region)

		dl.LoadVersions(dl.serviceData.Provider, dl.serviceData.Name, region)

		dl.LoadImages(dl.serviceData.Provider, dl.serviceData.Name, region)

		dl.LoadVms(dl.serviceData.Provider, dl.serviceData.Name, region)
	}

	dl.store.StoreRegions(dl.serviceData.Provider, dl.serviceData.Name, regionMap)
	dl.log.Debug("regions loaded")

	// set the status
	dl.store.StoreStatus(dl.serviceData.Provider, strconv.Itoa(int(time.Now().UnixNano()/1e6)))
	dl.log.Debug("status updated")

}

// loadZones loads zones for a given region in the store
func (dl *defaultCloudInfoLoader) LoadZones(provider, service string, region Region) {
	log := dl.log.WithFields(map[string]interface{}{"provider": provider, "service": service, "region": region.Id})
	log.Debug("loading zones...")
	defer log.Debug("loading zones... DONE.")

	dl.store.StoreZones(provider, service, region.Id, region.Data.Zones.Data)
}

// loadVersions loads versions for a given region into the store
func (dl *defaultCloudInfoLoader) LoadVersions(provider string, service string, region Region) {
	log := dl.log.WithFields(map[string]interface{}{"provider": provider, "service": service, "region": region.Id})
	log.Debug("loading versions...")
	defer log.Debug("loading versions... DONE.")

	dl.store.StoreVersion(provider, service, region.Id, region.Data.Versions.Data)

}

// loadImages loads images for a given region into the store
func (dl *defaultCloudInfoLoader) LoadImages(provider string, service string, region Region) {
	log := dl.log.WithFields(map[string]interface{}{"provider": provider, "service": service, "region": region.Id})
	log.Debug("loading images...")
	defer log.Debug("loading images... DONE.")
	if service == "pke" {
		return
	}
	dl.store.StoreImage(provider, service, region.Id, region.Data.Images.Data)
}

// loadVms loads vms for a given region into the store
func (dl *defaultCloudInfoLoader) LoadVms(provider string, service string, region Region) {
	log := dl.log.WithFields(map[string]interface{}{"provider": provider, "service": service, "region": region.Id})
	log.Debug("loading VMs...")
	defer log.Debug("loading VMs... DONE.")

	dl.store.StoreVm(provider, service, region.Id, region.Data.Vms.Data)
}

// loader implementation that populates service related data based on a source service available in the info store
type storeCloudInfoLoader struct {
	store       cloudinfo.CloudInfoStore
	log         cloudinfo.Logger
	serviceData ServiceData
	eventBus    messaging.EventBus
}

func (sl *storeCloudInfoLoader) Load() {
	sl.eventBus.SubscribeScrapingComplete(sl.serviceData.Provider, sl.LoadRegions)
}

// loadRegions loads regions in the cloud info store
func (sl *storeCloudInfoLoader) LoadRegions() {
	sl.log.Debug("loading region data...")

	regionMap := make(map[string]string)
	for _, region := range sl.serviceData.Regions {
		regionMap[region.Id] = region.Name

		sl.LoadZones(sl.serviceData.Provider, sl.serviceData.Name, region)

		sl.LoadVersions(sl.serviceData.Provider, sl.serviceData.Name, region)

		sl.LoadImages(sl.serviceData.Provider, sl.serviceData.Name, region)

		sl.LoadVms(sl.serviceData.Provider, sl.serviceData.Name, region)
	}

	sl.store.StoreRegions(sl.serviceData.Provider, sl.serviceData.Name, regionMap)
	sl.log.Debug("regions loaded")

	// set the status
	sl.store.StoreStatus(sl.serviceData.Provider, strconv.Itoa(int(time.Now().UnixNano()/1e6)))
	sl.log.Debug("status updated")

}

func (sl *storeCloudInfoLoader) LoadZones(provider string, service string, region Region) {
	// add method context to the logger
	log := sl.log.WithFields(map[string]interface{}{"provider": provider, "service": service, "region": region.Id})
	log.Debug("loading zones...")
	defer log.Debug("loading zones... DONE.")

	switch region.Data.Zones.Strategy {
	case exact:
		sl.store.StoreZones(provider, service, region.Id, region.Data.Zones.Data)

	case exclude:
		zones, ok := sl.store.GetZones(provider, sl.serviceData.Source, region.Id)
		if !ok {
			log.Warn("source service zones not yet cached", map[string]interface{}{"source": sl.serviceData.Source})
			return
		}

		filteredZones := make([]string, 0, len(zones))
		for _, zone := range zones {
			if cloudinfo.Contains(region.Data.Zones.Data, zone) {
				continue
			}
			filteredZones = append(filteredZones, zone)
		}

		sl.store.StoreZones(provider, service, region.Id, filteredZones)

	case include:
		zones, ok := sl.store.GetZones(provider, sl.serviceData.Source, region.Id)
		if !ok {
			log.Warn("source service zones not yet cached", map[string]interface{}{"source": sl.serviceData.Source})
			return
		}

		filteredZones := make([]string, 0, len(zones))
		for _, zone := range region.Data.Zones.Data {
			if cloudinfo.Contains(zones, zone) {
				filteredZones = append(filteredZones, zone)
			}
		}

		sl.store.StoreZones(provider, service, region.Id, filteredZones)

	default:
		log.Error("unsupported strategy for loading zones", map[string]interface{}{"strategy": region.Data.Zones.Strategy})
	}
}

func (sl *storeCloudInfoLoader) LoadVersions(provider string, service string, region Region) {
	// add method context to the logger
	log := sl.log.WithFields(map[string]interface{}{"provider": provider, "service": service, "region": region.Id})
	log.Debug("loading versions...")
	defer log.Debug("loading versions... DONE.")

	switch region.Data.Versions.Strategy {
	case exact:
		sl.store.StoreVersion(provider, service, region.Id, region.Data.Versions.Data)

	case exclude:
		sourceVersions, ok := sl.store.GetVersion(provider, sl.serviceData.Source, region.Id)
		if !ok {
			log.Warn("source service versions not yet cached", map[string]interface{}{"source": sl.serviceData.Source})
			return
		}

		filteredVersions := make([]types.LocationVersion, 0, len(sourceVersions))
		for _, version := range sourceVersions {
			for _, versionData := range region.Data.Versions.Data {
				if versionData.Location != version.Location {
					continue
				}

				for _, sourceVersion := range version.Versions {
					if cloudinfo.Contains(versionData.Versions, sourceVersion) {
						continue
					}

					filteredVersions = append(filteredVersions, version)
				}
			}
		}
		sl.store.StoreVersion(provider, service, region.Id, filteredVersions)

	case include:
		sourceVersions, ok := sl.store.GetVersion(provider, sl.serviceData.Source, region.Id)
		if !ok {
			log.Warn("source service versions not yet cached", map[string]interface{}{"source": sl.serviceData.Source})
			return
		}

		filteredVersions := make([]types.LocationVersion, 0, len(sourceVersions))
		for _, sourceVersion := range sourceVersions {
			for _, versionData := range region.Data.Versions.Data {

				if versionData.Location != sourceVersion.Location {
					continue
				}

				for _, sourceVersionValue := range sourceVersion.Versions {
					if cloudinfo.Contains(versionData.Versions, sourceVersionValue) {
						filteredVersions = append(filteredVersions, sourceVersion)
					}
				}
			}
		}

		sl.store.StoreVersion(provider, service, region.Id, filteredVersions)

	default:
		log.Error("unsupported strategy for loading versions", map[string]interface{}{"strategy": region.Data.Zones.Strategy})
	}
}

func (sl *storeCloudInfoLoader) LoadImages(provider string, service string, region Region) {

	// add method context to the logger
	log := sl.log.WithFields(map[string]interface{}{"provider": provider, "service": service, "region": region.Id})

	if service == "pke" {
		log.Debug("skip loading images - special case pke")
		return
	}

	log.Debug("loading images...")
	defer log.Debug("loading images... DONE.")

	switch region.Data.Images.Strategy {
	case exact:
		sl.store.StoreImage(provider, service, region.Id, region.Data.Images.Data)

	case exclude:
		sourceImages, ok := sl.store.GetImage(provider, sl.serviceData.Source, region.Id)
		if !ok {
			log.Warn("source service images not yet cached", map[string]interface{}{"source": sl.serviceData.Source})
			return
		}

		filteredImages := make([]types.Image, 0, len(sourceImages))
		for _, sourceImage := range sourceImages {
			for _, excludeImage := range region.Data.Images.Data {

				if excludeImage.Name != sourceImage.Name {
					filteredImages = append(filteredImages, sourceImage)
					break
				}
			}
		}

		sl.store.StoreImage(provider, service, region.Id, filteredImages)

	case include:
		sourceImages, ok := sl.store.GetImage(provider, sl.serviceData.Source, region.Id)
		if !ok {
			log.Warn("source service images not yet cached", map[string]interface{}{"source": sl.serviceData.Source})
			return
		}

		filteredImages := make([]types.Image, 0, len(sourceImages))
		for _, sourceImage := range sourceImages {
			for _, excludeImage := range region.Data.Images.Data {

				if excludeImage.Name == sourceImage.Name {
					filteredImages = append(filteredImages, sourceImage)
					break
				}
			}
		}

		sl.store.StoreImage(provider, service, region.Id, filteredImages)

	default:
		log.Error("unsupported strategy for loading images", map[string]interface{}{"strategy": region.Data.Zones.Strategy})
	}
}

func (sl *storeCloudInfoLoader) LoadVms(provider string, service string, region Region) {
	// add method context to the logger
	log := sl.log.WithFields(map[string]interface{}{"provider": provider, "service": service, "region": region.Id})
	log.Debug("loading VMs...")
	defer log.Debug("loading VMs... DONE.")

	switch region.Data.Vms.Strategy {
	case exact:
		sl.store.StoreVm(provider, service, region.Id, region.Data.Vms.Data)

	case exclude:
		sourceVMs, ok := sl.store.GetVm(provider, sl.serviceData.Source, region.Id)
		if !ok {
			log.Warn("source service VMs not yet cached", map[string]interface{}{"source": sl.serviceData.Source})
			return
		}

		// there are no exclusions
		if len(region.Data.Vms.Data) == 0 {
			sl.store.StoreVm(provider, service, region.Id, sourceVMs)
			return
		}

		filteredVMs := make([]types.VMInfo, 0, len(sourceVMs))
		for _, sourceVM := range sourceVMs {

			for _, excludeVM := range region.Data.Vms.Data {
				if excludeVM.Type != sourceVM.Type {
					filteredVMs = append(filteredVMs, sourceVM)
				}
			}
		}

		sl.store.StoreVm(provider, service, region.Id, filteredVMs)

	case include:
		sourceVMs, ok := sl.store.GetVm(provider, sl.serviceData.Source, region.Id)
		if !ok {
			log.Warn("source service VMs not yet cached", map[string]interface{}{"source": sl.serviceData.Source})
			return
		}

		// there are no inclusions
		if len(region.Data.Vms.Data) == 0 {
			sl.store.StoreVm(provider, service, region.Id, sourceVMs)
			return
		}

		filteredVMs := make([]types.VMInfo, 0, len(sourceVMs))
		for _, sourceVM := range sourceVMs {

			for _, excludeVM := range region.Data.Vms.Data {
				if excludeVM.Type == sourceVM.Type {
					filteredVMs = append(filteredVMs, sourceVM)
				}
			}
		}

		sl.store.StoreVm(provider, service, region.Id, filteredVMs)

	default:
		log.Error("unsupported strategy for loading VMs", map[string]interface{}{"strategy": region.Data.Zones.Strategy})
	}
}

func NewCloudInfoLoader(datapath, datafile, datatype string, store cloudinfo.CloudInfoStore, log cloudinfo.Logger,
	eventBus messaging.EventBus) CloudInfoLoader {

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
			log:         log.WithFields(map[string]interface{}{"component": "service-loader"}),
			store:       store,
			serviceData: serviceData,
			eventBus:    eventBus,
		}
	}

	return &defaultCloudInfoLoader{
		serviceData: serviceData,
		store:       store,
		log:         log.WithFields(map[string]interface{}{"component": "service-loader"}),
	}
}
