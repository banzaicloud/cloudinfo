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
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/messaging"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/spf13/viper"
)

// ServiceManager abstracts the operations related to cloud info services
type ServiceManager interface {
	// ConfigureServices parses the service configuration file and registers supported services
	ConfigureServices(providers []string)

	// LoadServiceInformation triggers importing cloud information based on the available service information
	LoadServiceInformation(providers []string)
}

// defaultServiceManager default implementation for the service manager
type defaultServiceManager struct {
	// (all) services parsed from the configuration
	services map[string][]ServiceData

	// the destination of the loaded data
	store cloudinfo.CloudInfoStore

	// component logger instance
	log logur.Logger

	// component eventbus instance
	eventBus messaging.EventBus
}

func (sm *defaultServiceManager) LoadServiceInformation(providers []string) {
	sm.log.Info("triggering cloud information importing ...")
	for _, provider := range providers {

		for _, service := range sm.services[provider] {
			if !service.IsStatic {
				sm.log.Debug("skip loading static cloud information, dynamic service", map[string]interface{}{"service": service.Name})
				continue
			}

			cloudInfoLoader := NewCloudInfoLoader(service.DataLocation, service.DataFile, service.DataType, sm.store, sm.log, sm.eventBus)

			cloudInfoLoader.Load()
		}

	}
	sm.log.Info("triggering cloud information importing ...")
}

func (sm *defaultServiceManager) ConfigureServices(providers []string) {
	sm.log.Info("initializing services for providers...")
	for p, psvcs := range sm.services {
		if !cloudinfo.Contains(providers, p) {
			sm.log.Debug("skip loading services for provider", map[string]interface{}{"provider": p})
			continue
		}
		var svcs []cloudinfo.Service
		for _, psvc := range psvcs {
			svcs = append(svcs, cloudinfo.Service{Service: psvc.Name, IsStatic: psvc.IsStatic})
		}
		sm.store.StoreServices(p, svcs)
	}
	sm.log.Info("services initialized")
}

func NewDefaultServiceManager(config Config, store cloudinfo.CloudInfoStore, log logur.Logger, eventBus messaging.EventBus) ServiceManager {
	// using a viper instance for loading data
	vp := viper.New()
	vp.AddConfigPath(config.ServiceConfigLocation)
	vp.SetConfigName(config.ServiceConfigName)
	vp.SetConfigType(config.Format)

	if err := vp.ReadInConfig(); err != nil { // Find and read the config file
		// Handle errors reading the config file
		emperror.Panic(err)
	}

	var (
		sds map[string][]ServiceData
	)

	if err := vp.Unmarshal(&sds); err != nil {
		emperror.Panic(err)
	}

	return &defaultServiceManager{
		store:    store,
		log:      logur.WithFields(log, map[string]interface{}{"component": "service-manager"}),
		services: sds,
		eventBus: eventBus,
	}
}
