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
	"time"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/platform/cassandra"
	"github.com/banzaicloud/cloudinfo/pkg/platform/redis"
)

// CloudInfoStore configuration
type Config struct {
	Redis     redis.Config
	GoCache   GoCacheConfig
	Cassandra cassandra.Config
}

// GoCacheConfig configuration
type GoCacheConfig struct {
	expiration      time.Duration
	cleanupInterval time.Duration
}

// NewCloudInfoStore builds a new cloudinfo store based on the passed in configuration
// This method is in charge to create the appropriate store instance eventually to implement a fallback mechanism to the default store
func NewCloudInfoStore(conf Config, log cloudinfo.Logger) cloudinfo.CloudInfoStore {

	// use redis if enabled
	if conf.Redis.Enabled {
		log.Info("using Redis as product store")
		return NewRedisProductStore(conf.Redis, log)
	}

	if conf.Cassandra.Enabled {
		log.Info("using Cassandra as product store")
		return NewCassandraProductStore(conf.Cassandra, log)
	}

	// fallback to the "initial" implementation
	log.Info("using in-mem cache as product store")
	return NewCacheProductStore(conf.GoCache.expiration, conf.GoCache.cleanupInterval, log)
}
