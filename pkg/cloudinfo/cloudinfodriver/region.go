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

package cloudinfodriver

import (
	"context"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
)

const (
	OperationRegionListRegions = "cloudinfo.Region.ListRegions"
	OperationRegionListZones   = "cloudinfo.Region.ListZones"
)

// RegionService provides access to regions supported by a service.
type RegionService interface {
	// ListRegions returns a list of regions supported by a service.
	ListRegions(ctx context.Context, provider string, service string) ([]cloudinfo.Region, error)

	// ListZones returns a list of zones within a region.
	ListZones(ctx context.Context, provider string, service string, region string) ([]cloudinfo.Zone, error)
}
