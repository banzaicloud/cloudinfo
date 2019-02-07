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
	"fmt"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"

	"github.com/goph/emperror"

	"github.com/goph/logur"
	"github.com/spf13/viper"
)

// Loader abstracts the loading logic
// I'ts purpose is to provide the possibility to load information from various sources (but scraping)
type Loader interface {
	Load(ctx context.Context)
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

// Load entry point to the service loading logic
func (sl *defaultServiceLoader) Load(ctx context.Context) {
	sl.log.Info("loading service information...")
	var sd []ServiceData

	if err := sl.viper.Unmarshal(&sd); err != nil {
		sl.log.Error("failed to load service data")
		emperror.Panic(err)
	}

	sl.log.Info("service information loaded")

}

func NewDefaultServiceLoader(config Config, log logur.Logger) Loader {
	// using a viper instance for loading data
	vp := viper.New()
	vp.SetConfigName(config.Name)     // name of config file (without extension)
	vp.AddConfigPath(config.Location) // path to look for the config file in
	vp.SetConfigType(config.Format)
	err := vp.ReadInConfig() // Find and read the config file
	if err != nil {          // Handle errors reading the config file
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}

	return &defaultServiceLoader{
		viper: vp,
		log:   logur.WithFields(log, map[string]interface{}{"service": "svcloader"}),
	}
}
