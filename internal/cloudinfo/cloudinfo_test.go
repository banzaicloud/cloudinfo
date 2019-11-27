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
	"testing"

	"github.com/goph/logur"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

// DummyCloudInfoStore type implements the CloudInfoStore interface for mockig of external calls
// the struct is to be extended according to the needs of test cases
type DummyCloudInfoStore struct {
	TcId string
	// implement the interface
	CloudInfoStore
}

const notCached = "error"

func (dcis *DummyCloudInfoStore) GetRegions(provider, service string) (map[string]string, bool) {
	switch dcis.TcId {
	case notCached:
		return nil, false
	default:
		return map[string]string{
				"US West (Oregon)": "us-west-2",
				"EU (Frankfurt)":   "eu-central-1",
				"EU (Ireland)":     "eu-west-1",
			},
			true
	}
}

func (dcis *DummyCloudInfoStore) GetZones(provider, service, region string) ([]string, bool) {
	switch dcis.TcId {
	case notCached:
		return nil, false
	default:
		return []string{
				"eu-central-1a",
				"eu-central-1b",
			},
			true
	}
}

func (dcis *DummyCloudInfoStore) GetImage(provider, service, regionId string) ([]types.Image, bool) {
	switch dcis.TcId {
	case notCached:
		return nil, false
	default:
		return []types.Image{
				{
					Name:         "ami-12345676",
					GpuAvailable: false,
					Version:      "1.10",
				},
				{
					Name:         "ami-3246433",
					GpuAvailable: true,
					Version:      "1.11",
				},
			},
			true
	}
}

func (dcis *DummyCloudInfoStore) GetVersion(provider, service, region string) ([]types.LocationVersion, bool) {
	switch dcis.TcId {
	case notCached:
		return nil, false
	default:
		return []types.LocationVersion{
				{
					Versions: []string{
						"1.10",
						"1.11",
					},
				},
			},
			true
	}
}

func (dcis *DummyCloudInfoStore) GetStatus(provider string) (string, bool) {
	switch dcis.TcId {
	case notCached:
		return "", false
	default:
		return "dummyStatus", true
	}
}

func (dcis *DummyCloudInfoStore) GetServices(provider string) ([]types.Service, bool) {
	switch dcis.TcId {
	case notCached:
		return nil, false
	default:
		return []types.Service{
				{
					Service:  "dummy1",
					IsStatic: false,
				},
				{
					Service:  "dummy2",
					IsStatic: false,
				},
			},
			true
	}
}

func TestNewCachingCloudInfo(t *testing.T) {
	tests := []struct {
		Name        string
		CloudInfoer []string
		checker     func(info *cloudInfo, err error)
	}{
		{
			Name:        "product info successfully created",
			CloudInfoer: []string{""},
			checker: func(info *cloudInfo, err error) {
				assert.Nil(t, err, "should not get error")
				assert.NotNil(t, info, "the product info should not be nil")
			},
		},
		{
			Name:        "validation should fail nil values",
			CloudInfoer: nil,
			checker: func(info *cloudInfo, err error) {
				assert.Nil(t, info, "the cloudinfo should be nil in case of error")
				assert.EqualError(t, err, "could not create product infoer")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test.checker(NewCloudInfo(test.CloudInfoer, &DummyCloudInfoStore{}, logur.NewTestLogger()))
		})
	}

}

func TestCachingCloudInfo_GetRegions(t *testing.T) {
	tests := []struct {
		name    string
		ciStore CloudInfoStore
		checker func(regions map[string]string, err error)
	}{
		{
			name:    "successfully retrieved the regions",
			ciStore: &DummyCloudInfoStore{},
			checker: func(regions map[string]string, err error) {
				assert.Equal(t, map[string]string{
					"US West (Oregon)": "us-west-2",
					"EU (Frankfurt)":   "eu-central-1",
					"EU (Ireland)":     "eu-west-1"}, regions)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:    "failed to retrieve regions",
			ciStore: &DummyCloudInfoStore{TcId: notCached},
			checker: func(regions map[string]string, err error) {
				assert.Nil(t, regions, "the regions should be nil")
				assert.EqualError(t, err, "regions not yet cached")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, _ := NewCloudInfo([]string{}, &DummyCloudInfoStore{}, logur.NewTestLogger())
			info.cloudInfoStore = test.ciStore
			test.checker(info.GetRegions("dummyProvider", "dummyService"))
		})
	}
}

func TestCachingCloudInfo_GetVersions(t *testing.T) {
	tests := []struct {
		name    string
		ciStore CloudInfoStore
		checker func(versions []types.LocationVersion, err error)
	}{
		{
			name:    "successfully retrieved the versions",
			ciStore: &DummyCloudInfoStore{},
			checker: func(versions []types.LocationVersion, err error) {
				assert.Equal(t, 1, len(versions))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:    "failed to retrieve versions",
			ciStore: &DummyCloudInfoStore{TcId: notCached},
			checker: func(versions []types.LocationVersion, err error) {
				assert.Nil(t, versions, "the versions should be nil")
				assert.EqualError(t, err, "versions not yet cached")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, _ := NewCloudInfo([]string{}, &DummyCloudInfoStore{}, logur.NewTestLogger())
			info.cloudInfoStore = test.ciStore
			test.checker(info.GetVersions("dummyProvider", "dummyService", "dummyRegion"))
		})
	}
}

func TestCachingCloudInfo_GetServiceImages(t *testing.T) {
	tests := []struct {
		name    string
		ciStore CloudInfoStore
		checker func(images []types.Image, err error)
	}{
		{
			name:    "successfully retrieved the images",
			ciStore: &DummyCloudInfoStore{},
			checker: func(images []types.Image, err error) {
				assert.Equal(t, 2, len(images))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:    "failed to retrieve images",
			ciStore: &DummyCloudInfoStore{TcId: notCached},
			checker: func(images []types.Image, err error) {
				assert.Nil(t, images, "the images should be nil")
				assert.EqualError(t, err, "images not yet cached")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, _ := NewCloudInfo([]string{}, &DummyCloudInfoStore{}, logur.NewTestLogger())
			info.cloudInfoStore = test.ciStore
			test.checker(info.GetServiceImages("dummyProvider", "dummyService", "dummyRegion"))
		})
	}
}

func TestCachingCloudInfo_GetZones(t *testing.T) {
	tests := []struct {
		name    string
		ciStore CloudInfoStore
		checker func(zones []string, err error)
	}{
		{
			name:    "successfully retrieved the zones",
			ciStore: &DummyCloudInfoStore{},
			checker: func(zones []string, err error) {
				assert.Equal(t, []string{"eu-central-1a", "eu-central-1b"}, zones)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:    "failed to retrieve zones",
			ciStore: &DummyCloudInfoStore{TcId: notCached},
			checker: func(zones []string, err error) {
				assert.Nil(t, zones, "the zones should be nil")
				assert.EqualError(t, err, "zones not yet cached")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, _ := NewCloudInfo([]string{}, &DummyCloudInfoStore{}, logur.NewTestLogger())
			info.cloudInfoStore = test.ciStore
			test.checker(info.GetZones("dummyProvider", "dummyService", "dummyRegion"))
		})
	}
}

func TestCachingCloudInfo_GetServices(t *testing.T) {
	tests := []struct {
		name    string
		ciStore CloudInfoStore
		checker func(services []types.Service, err error)
	}{
		{
			name:    "successfully retrieved the services",
			ciStore: &DummyCloudInfoStore{},
			checker: func(services []types.Service, err error) {
				assert.Equal(t, 2, len(services))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:    "failed to retrieve services",
			ciStore: &DummyCloudInfoStore{TcId: notCached},
			checker: func(services []types.Service, err error) {
				assert.Nil(t, services, "the services should be nil")
				assert.EqualError(t, err, "services not yet cached")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, _ := NewCloudInfo([]string{}, &DummyCloudInfoStore{}, logur.NewTestLogger())
			info.cloudInfoStore = test.ciStore
			test.checker(info.GetServices("dummyProvider"))
		})
	}
}

func TestCachingCloudInfo_GetStatus(t *testing.T) {
	tests := []struct {
		name    string
		ciStore CloudInfoStore
		checker func(status string, err error)
	}{
		{
			name:    "successfully retrieved the status",
			ciStore: &DummyCloudInfoStore{},
			checker: func(status string, err error) {
				assert.Equal(t, "dummyStatus", status)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:    "failed to retrieve status",
			ciStore: &DummyCloudInfoStore{TcId: notCached},
			checker: func(status string, err error) {
				assert.Equal(t, "", status)
				assert.EqualError(t, err, "status not yet cached")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, _ := NewCloudInfo([]string{}, &DummyCloudInfoStore{}, logur.NewTestLogger())
			info.cloudInfoStore = test.ciStore
			test.checker(info.GetStatus("dummyProvider"))
		})
	}
}
