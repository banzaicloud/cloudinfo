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

package cloudinfo

import (
	"fmt"
	"github.com/patrickmn/go-cache"
	"time"
)

const (
	// VmKeyTemplate format for generating vm cache keys
	VmKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/vms"

	// AttrKeyTemplate format for generating attribute cache keys
	AttrKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/attrValues/%s"

	// PriceKeyTemplate format for generating price cache keys
	PriceKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/regions/%s/prices/%s"

	// ZoneKeyTemplate format for generating zone cache keys
	ZoneKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/regions/%s/zones/"

	// RegionKeyTemplate format for generating region cache keys
	RegionKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/"

	// StatusKeyTemplate format for generating status cache keys
	StatusKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/status/"

	// ImageKeyTemplate format for generating image cache keys
	ImageKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/images"

	// VersionKeyTemplate format for generating kubernetes version cache keys
	VersionKeyTemplate = "/banzaicloud.com/cloudinfo/providers/%s/services/%s/regions/%s/versions"
)

// ProductStorer contract for storing and retrieving cloud product information
type ProductStorer interface {
	Get(k string) (interface{}, bool)
	Set(k string, x interface{}, d time.Duration)
}

type CacheKeyProvider interface {
	getZonesKey(provider string, region string) string

}

// CacheProductStorer in memory cloud product information storer
type CacheProductStorer struct {
	*cache.Cache
}

// NewCacheProductStorer creates a new ProductStorer instance backef by in memory cache imlementation
func NewCacheProductStorer(cleanupInterval time.Duration) ProductStorer {
	return CacheProductStorer{
		cache.New(cache.NoExpiration, cleanupInterval),
	}
}


func (cpi *CacheProductStorer) getZonesKey(provider string, region string) string {
	return fmt.Sprintf(ZoneKeyTemplate, provider, region)
}

