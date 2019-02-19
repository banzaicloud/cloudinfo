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
				if !cloudinfo.Contains(region.Data.Zones.Data, zone) {
					availableZones = append(availableZones, zone)
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

func (scil *storeCloudInfoLoader) LoadVersions(ctx context.Context, provider string, service string, region Region) {
	switch region.Data.Versions.Strategy {
	case exact:
		scil.log.Debug("loading versions...")
		scil.store.StoreVersion(provider, service, region.Id, region.Data.Versions.Data)
		scil.log.Debug("versions loaded")
	case exclude:
		var (
			versions interface{}
			ok       bool
		)
		if versions, ok = scil.store.GetVersion(provider, scil.serviceData.Source, region.Id); !ok {
			scil.log.Error("versions not yet cached",
				map[string]interface{}{"provider": provider, "service": scil.serviceData.Source, "region": region.Id})
		} else {
			var availableVersions []string
			for _, version := range versions.([]string) {
				if !cloudinfo.Contains(region.Data.Versions.Data, version) {
					availableVersions = append(availableVersions, version)
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
			var availableVersions []string
			for _, version := range region.Data.Versions.Data {
				if cloudinfo.Contains(versions.([]string), version) {
					availableVersions = append(availableVersions, version)
				}
			}
			scil.store.StoreVersion(provider, service, region.Id, availableVersions)
		}
	default:
		scil.log.Error("invalid version strategy",
			map[string]interface{}{"provider": provider, "service": service, "region": region.Id, "strategy": region.Data.Versions.Strategy})
	}
}

func (scil *storeCloudInfoLoader) LoadImages(ctx context.Context, provider string, service string, region Region) {
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
			var availableImages []cloudinfo.Image
			var unavailableImages []string
			for _, image := range images.([]cloudinfo.Image) {
				for _, img := range region.Data.Images.Data {
					if img.Name == image.Name {
						unavailableImages = append(unavailableImages, image.Name)
						break
					}
				}
			}

			for _, image := range images.([]cloudinfo.Image) {
				if !cloudinfo.Contains(unavailableImages, image.Name) {
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
			var availableImages []cloudinfo.Image
			for _, image := range region.Data.Images.Data {
				for _, img := range images.([]cloudinfo.Image) {
					if image.Name == img.Name {
						availableImages = append(availableImages, img)
					}
				}
			}
			scil.store.StoreVersion(provider, service, region.Id, availableImages)
		}
	default:
		scil.log.Error("invalid image strategy",
			map[string]interface{}{"provider": provider, "service": service, "region": region.Id, "strategy": region.Data.Images.Strategy})
	}
}

func (scil *storeCloudInfoLoader) LoadVms(ctx context.Context, provider string, service string, region Region) {
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
			var availableVms []cloudinfo.VmInfo
			var unavailableVms []string
			for _, vm := range vms.([]cloudinfo.VmInfo) {
				for _, _vm := range region.Data.Vms.Data {
					if _vm.Type == vm.Type {
						unavailableVms = append(unavailableVms, vm.Type)
						break
					}
				}
			}

			for _, vm := range vms.([]cloudinfo.VmInfo) {
				if !cloudinfo.Contains(unavailableVms, vm.Type) {
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
			var availableVms []cloudinfo.VmInfo
			for _, _vm := range region.Data.Vms.Data {
				for _, vm := range vms.([]cloudinfo.VmInfo) {
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

	return &storeCloudInfoLoader{
		log:         logur.WithFields(log, map[string]interface{}{"component": "service-loader"}),
		store:       store,
		serviceData: serviceData,
	}
}
