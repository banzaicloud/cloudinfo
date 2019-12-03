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
	dl.log.Debug("loading zones...", map[string]interface{}{"service": service, "region": region.Id})
	dl.store.StoreZones(provider, service, region.Id, region.Data.Zones.Data)
	dl.log.Debug("zones loaded", map[string]interface{}{"service": service, "region": region.Id})
}

// loadVersions loads versions for a given region into the store
func (dl *defaultCloudInfoLoader) LoadVersions(provider string, service string, region Region) {
	dl.log.Debug("loading versions...", map[string]interface{}{"service": service, "region": region.Id})
	dl.store.StoreVersion(provider, service, region.Id, region.Data.Versions.Data)
	dl.log.Debug("versions loaded", map[string]interface{}{"service": service, "region": region.Id})
}

// loadImages loads images for a given region into the store
func (dl *defaultCloudInfoLoader) LoadImages(provider string, service string, region Region) {
	dl.log.Debug("loading images...", map[string]interface{}{"service": service, "region": region.Id})
	dl.store.StoreImage(provider, service, region.Id, region.Data.Images.Data)
	dl.log.Debug("images loaded", map[string]interface{}{"service": service, "region": region.Id})
}

// loadVms loads vms for a given region into the store
func (dl *defaultCloudInfoLoader) LoadVms(provider string, service string, region Region) {
	dl.log.Debug("loading vms...", map[string]interface{}{"service": service, "region": region.Id})
	dl.store.StoreVm(provider, service, region.Id, region.Data.Vms.Data)
	dl.log.Debug("vms loaded", map[string]interface{}{"service": service, "region": region.Id})
}

type storeCloudInfoLoader struct {
	store       cloudinfo.CloudInfoStore
	log         cloudinfo.Logger
	serviceData ServiceData
	eventBus    messaging.EventBus
}

func (scil *storeCloudInfoLoader) Load() {
	scil.eventBus.SubscribeScrapingComplete(scil.serviceData.Provider, scil.LoadRegions)
}

// loadRegions loads regions in the cloud info store
func (scil *storeCloudInfoLoader) LoadRegions() {
	scil.log.Debug("loading region data...")

	regionMap := make(map[string]string)
	for _, region := range scil.serviceData.Regions {
		regionMap[region.Id] = region.Name

		scil.LoadZones(scil.serviceData.Provider, scil.serviceData.Name, region)

		scil.LoadVersions(scil.serviceData.Provider, scil.serviceData.Name, region)

		scil.LoadImages(scil.serviceData.Provider, scil.serviceData.Name, region)

		scil.LoadVms(scil.serviceData.Provider, scil.serviceData.Name, region)
	}

	scil.store.StoreRegions(scil.serviceData.Provider, scil.serviceData.Name, regionMap)
	scil.log.Debug("regions loaded")

	// set the status
	scil.store.StoreStatus(scil.serviceData.Provider, strconv.Itoa(int(time.Now().UnixNano()/1e6)))
	scil.log.Debug("status updated")

}

func (scil *storeCloudInfoLoader) LoadZones(provider string, service string, region Region) {
	switch region.Data.Zones.Strategy {
	case exact:
		scil.log.Debug("loading zones...")
		scil.store.StoreZones(provider, service, region.Id, region.Data.Zones.Data)
		scil.log.Debug("zones loaded")
	case exclude:
		var (
			zones interface{}
			ok    bool
		)
		if zones, ok = scil.store.GetZones(provider, scil.serviceData.Source, region.Id); !ok {
			scil.log.Error("zones not yet cached",
				map[string]interface{}{"provider": provider, "service": scil.serviceData.Source, "region": region.Id})
		} else {
			var availableZones []string
			for _, zone := range zones.([]string) {
				if !cloudinfo.Contains(region.Data.Zones.Data, string(zone)) {
					availableZones = append(availableZones, string(zone))
				}
			}
			scil.store.StoreZones(provider, service, region.Id, availableZones)
		}
	case include:
		var (
			zones interface{}
			ok    bool
		)
		if zones, ok = scil.store.GetZones(provider, scil.serviceData.Source, region.Id); !ok {
			scil.log.Error("zones not yet cached",
				map[string]interface{}{"provider": provider, "service": scil.serviceData.Source, "region": region.Id})
		} else {
			var availableZones []string
			for _, zone := range region.Data.Zones.Data {
				if cloudinfo.Contains(zones.([]string), zone) {
					availableZones = append(availableZones, zone)
				}
			}
			scil.store.StoreZones(provider, service, region.Id, availableZones)
		}
	default:
		scil.log.Error("invalid zone strategy",
			map[string]interface{}{"provider": provider, "service": service, "region": region.Id, "strategy": region.Data.Zones.Strategy})
	}
}

func (scil *storeCloudInfoLoader) LoadVersions(provider string, service string, region Region) {
	switch region.Data.Versions.Strategy {
	case exact:
		scil.log.Debug("loading versions...", map[string]interface{}{"service": service, "region": region.Id})
		scil.store.StoreVersion(provider, service, region.Id, region.Data.Versions.Data)
		scil.log.Debug("versions loaded", map[string]interface{}{"service": service, "region": region.Id})
	case exclude:
		var (
			versions interface{}
			ok       bool
		)
		if versions, ok = scil.store.GetVersion(provider, scil.serviceData.Source, region.Id); !ok {
			scil.log.Error("versions not yet cached",
				map[string]interface{}{"provider": provider, "service": scil.serviceData.Source, "region": region.Id})
		} else {
			var availableVersions []types.LocationVersion
			for _, version := range versions.([]types.LocationVersion) {
				for _, data := range region.Data.Versions.Data {
					if data.Location == version.Location {
						for _, v := range version.Versions {
							keep := true
							for _, excludeVersion := range data.Versions {
								if v == excludeVersion {
									keep = false
									break
								}
							}
							if keep {
								availableVersions = append(availableVersions, version)
							}
						}
					}

				}
			}
			scil.store.StoreVersion(provider, service, region.Id, availableVersions)
		}
	case include:
		var (
			versions interface{}
			ok       bool
		)
		if versions, ok = scil.store.GetVersion(provider, scil.serviceData.Source, region.Id); !ok {
			scil.log.Error("versions not yet cached",
				map[string]interface{}{"provider": provider, "service": scil.serviceData.Source, "region": region.Id})
		} else {
			var availableVersions []types.LocationVersion
			for _, data := range region.Data.Versions.Data {
				for _, version := range versions.([]types.LocationVersion) {
					if data.Location == version.Location {
						for _, includeVersion := range data.Versions {
							for _, v := range version.Versions {
								if includeVersion == v {
									availableVersions = append(availableVersions, version)
								}
							}
						}
					}
				}
			}
			scil.store.StoreVersion(provider, service, region.Id, availableVersions)
		}
	default:
		scil.log.Error("invalid version strategy",
			map[string]interface{}{"provider": provider, "service": service, "region": region.Id, "strategy": region.Data.Versions.Strategy})
	}
}

func (scil *storeCloudInfoLoader) LoadImages(provider string, service string, region Region) {
	switch region.Data.Images.Strategy {
	case exact:
		scil.log.Debug("loading images...")
		scil.store.StoreImage(provider, service, region.Id, region.Data.Images.Data)
		scil.log.Debug("images loaded")
	case exclude:
		var (
			images interface{}
			ok     bool
		)
		if images, ok = scil.store.GetImage(provider, scil.serviceData.Source, region.Id); !ok {
			scil.log.Error("images not yet cached",
				map[string]interface{}{"provider": provider, "service": scil.serviceData.Source, "region": region.Id})
		} else {
			var availableImages []types.Image
			for _, image := range images.([]types.Image) {
				keep := true
				for _, excludeImage := range region.Data.Images.Data {
					if excludeImage.Name == image.Name {
						keep = false
						break
					}
				}
				if keep {
					availableImages = append(availableImages, image)
				}
			}

			scil.store.StoreImage(provider, service, region.Id, availableImages)
		}
	case include:
		var (
			images interface{}
			ok     bool
		)
		if images, ok = scil.store.GetImage(provider, scil.serviceData.Source, region.Id); !ok {
			scil.log.Error("images not yet cached",
				map[string]interface{}{"provider": provider, "service": scil.serviceData.Source, "region": region.Id})
		} else {
			var availableImages []types.Image
			for _, image := range region.Data.Images.Data {
				for _, img := range images.([]types.Image) {
					if image.Name == img.Name {
						availableImages = append(availableImages, img)
					}
				}
			}
			scil.store.StoreImage(provider, service, region.Id, availableImages)
		}
	default:
		scil.log.Error("invalid image strategy",
			map[string]interface{}{"provider": provider, "service": service, "region": region.Id, "strategy": region.Data.Images.Strategy})
	}
}

func (scil *storeCloudInfoLoader) LoadVms(provider string, service string, region Region) {
	switch region.Data.Vms.Strategy {
	case exact:
		scil.log.Debug("loading vms...")
		scil.store.StoreVm(provider, service, region.Id, region.Data.Vms.Data)
		scil.log.Debug("vms loaded")
	case exclude:
		var (
			vms interface{}
			ok  bool
		)
		if vms, ok = scil.store.GetVm(provider, scil.serviceData.Source, region.Id); !ok {
			scil.log.Error("vms not yet cached",
				map[string]interface{}{"provider": provider, "service": scil.serviceData.Source, "region": region.Id})
		} else {
			var availableVms []types.VMInfo
			for _, vm := range vms.([]types.VMInfo) {
				keep := true
				for _, excludeVm := range region.Data.Vms.Data {
					if excludeVm.Type == vm.Type {
						keep = false
						break
					}
				}
				if keep {
					availableVms = append(availableVms, vm)
				}
			}

			scil.store.StoreVm(provider, service, region.Id, availableVms)
		}
	case include:
		var (
			vms interface{}
			ok  bool
		)
		if vms, ok = scil.store.GetVm(provider, scil.serviceData.Source, region.Id); !ok {
			scil.log.Error("vms not yet cached",
				map[string]interface{}{"provider": provider, "service": scil.serviceData.Source, "region": region.Id})
		} else {
			var availableVms []types.VMInfo
			for _, _vm := range region.Data.Vms.Data {
				for _, vm := range vms.([]types.VMInfo) {
					if _vm.Type == vm.Type {
						availableVms = append(availableVms, vm)
					}
				}
			}
			scil.store.StoreVm(provider, service, region.Id, availableVms)
		}
	default:
		scil.log.Error("invalid vm strategy",
			map[string]interface{}{"provider": provider, "service": service, "region": region.Id, "strategy": region.Data.Vms.Strategy})
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
		// servicloader implementation that uses another service as source
		return &storeCloudInfoLoader{
			log:         logur.WithFields(log, map[string]interface{}{"component": "service-loader"}),
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
