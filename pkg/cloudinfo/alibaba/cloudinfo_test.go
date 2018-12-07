// Copyright © 2018 Banzai Cloud
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

package alibaba

import (
	"context"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

//testStruct helps to mock external calls
type testStruct struct {
	TcId string
}

const (
	externalApiError         = "external API error"
	getRegionsError          = "failed to get regions"
	getVmsError              = "failed to get virtual machines"
	getZonesError            = "failed to get zones"
	getPriceError            = "could not retrieve current prices"
	getSpotPriceHistoryError = "failed to get spot price"
)

func (dps *testStruct) DescribeInstanceTypes(request *ecs.DescribeInstanceTypesRequest) (response *ecs.DescribeInstanceTypesResponse, err error) {
	switch dps.TcId {
	case getVmsError:
		return &ecs.DescribeInstanceTypesResponse{}, errors.New(externalApiError)
	default:
		return &ecs.DescribeInstanceTypesResponse{
			InstanceTypes: ecs.InstanceTypesInDescribeInstanceTypes{
				InstanceType: []ecs.InstanceType{
					{
						InstanceTypeId:      "ecs.g5.2xlarge",
						CpuCoreCount:        8,
						MemorySize:          32.0,
						InstanceBandwidthRx: 1024000,
					},
					{
						InstanceTypeId:      "ecs.g5.8xlarge",
						CpuCoreCount:        32,
						MemorySize:          128.0,
						InstanceBandwidthRx: 1024000,
					},
					{
						InstanceTypeId:      "ecs.sn2ne.8xlarge",
						CpuCoreCount:        32,
						MemorySize:          128.0,
						InstanceBandwidthRx: 1024000,
					},
					{
						InstanceTypeId:      "ecs.ic5.3xlarge",
						CpuCoreCount:        12,
						MemorySize:          12.0,
						InstanceBandwidthRx: 1024000,
					},
				},
			},
		}, nil
	}
}

func (dps *testStruct) DescribeSpotPriceHistory(request *ecs.DescribeSpotPriceHistoryRequest) (response *ecs.DescribeSpotPriceHistoryResponse, err error) {
	switch dps.TcId {
	case getSpotPriceHistoryError:
		return &ecs.DescribeSpotPriceHistoryResponse{}, errors.New(externalApiError)
	default:
		return &ecs.DescribeSpotPriceHistoryResponse{
			Currency: "USD",
			SpotPrices: ecs.SpotPrices{
				SpotPriceType: []ecs.SpotPriceType{
					{
						InstanceType: "ecs.g5.2xlarge",
						ZoneId:       "dummyZone",
						NetworkType:  "vpc",
						OriginPrice:  1.435,
						SpotPrice:    0.652,
					},
					{
						InstanceType: "ecs.sn2ne.8xlarge",
						ZoneId:       "us-east-1b",
						NetworkType:  "vpc",
						OriginPrice:  2.243,
						SpotPrice:    1.021,
					},
				},
			},
		}, nil
	}
}

func (dps *testStruct) DescribeZones(request *ecs.DescribeZonesRequest) (response *ecs.DescribeZonesResponse, err error) {
	switch dps.TcId {
	case getZonesError:
		return &ecs.DescribeZonesResponse{}, errors.New(externalApiError)
	default:
		return &ecs.DescribeZonesResponse{
			RequestId: "dummyRequestId",
			Zones: ecs.ZonesInDescribeZones{
				Zone: []ecs.Zone{
					{
						ZoneId: "us-east-1a",
					},
					{
						ZoneId: "us-east-1b",
					},
					{
						ZoneId: "eu-central-1c",
					},
				},
			},
		}, nil

	}
}

func (dps *testStruct) DescribeRegions(request *ecs.DescribeRegionsRequest) (response *ecs.DescribeRegionsResponse, err error) {
	switch dps.TcId {
	case getRegionsError:
		return &ecs.DescribeRegionsResponse{}, errors.New(externalApiError)
	default:
		return &ecs.DescribeRegionsResponse{
			RequestId: "dummyRequestId",
			Regions: ecs.Regions{
				Region: []ecs.Region{
					{
						RegionId:  "cn-qingdao",
						LocalName: "China (Qingdao)",
					},
					{
						RegionId:  "us-east-1",
						LocalName: "US (Virginia)",
					},
					{
						RegionId:  "ap-southeast-5",
						LocalName: "Indonesia (Jakarta)",
					},
				},
			},
		}, nil
	}
}

func (dps *testStruct) getOnDemandPrice(url string) (OnDemandPrice, error) {
	switch dps.TcId {
	case getPriceError:
		return OnDemandPrice{}, errors.New(externalApiError)
	default:
		return OnDemandPrice{
			Currency: "USD",
			PricingInfo: map[string]TimeUnitPrice{
				"us-east-1::ecs.g5.2xlarge::vpc::linux::optimized": {
					Hours: []Price{
						{
							Price:  "0.336",
							Period: "1",
						},
					},
				},
				"us-east-1::ecs.sn2ne.8xlarge::vpc::linux::optimized": {
					Hours: []Price{
						{
							Price:  "0.011",
							Period: "1",
						},
					},
				},
				"ap-southeast-2::ecs.n4.large::vpc::windows::optimized": {
					Hours: []Price{
						{
							Price:  "0.082",
							Period: "1",
						},
					},
				},
			},
		}, nil
	}
}

func TestAlibabaInfoer_GetRegions(t *testing.T) {
	tests := []struct {
		name   string
		client EcsSource
		check  func(regions map[string]string, err error)
	}{
		{
			name:   "receive all regions",
			client: &testStruct{},
			check: func(regions map[string]string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, map[string]string(map[string]string{"cn-qingdao": "China (Qingdao)", "us-east-1": "US (Virginia)", "ap-southeast-5": "Indonesia (Jakarta)"}), regions)
			},
		},
		{
			name:   "could not retrieve regions",
			client: &testStruct{getRegionsError},
			check: func(regions map[string]string, err error) {
				assert.Nil(t, regions, "the regions should be nil")
				assert.EqualError(t, err, externalApiError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloudInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			cloudInfoer.ecsClient = test.client
			if err != nil {
				t.Fatalf("failed to create cloudinfoer; [%s]", err.Error())
			}

			test.check(cloudInfoer.GetRegions(context.TODO(), "dummyService"))
		})
	}
}

func TestAlibabaInfoer_GetZones(t *testing.T) {
	tests := []struct {
		name   string
		client EcsSource
		region string
		check  func(zones []string, err error)
	}{
		{
			name:   "receive all zones",
			client: &testStruct{},
			region: "eu-central-1",
			check: func(zones []string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, []string([]string{"us-east-1a", "us-east-1b", "eu-central-1c"}), zones)
			},
		},
		{
			name:   "could not retrieve zones",
			client: &testStruct{getZonesError},
			region: "eu-central-1",
			check: func(zones []string, err error) {
				assert.Nil(t, zones, "the regions should be nil")
				assert.EqualError(t, err, externalApiError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloudInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			cloudInfoer.ecsClient = test.client
			if err != nil {
				t.Fatalf("failed to create cloudinfoer; [%s]", err.Error())
			}

			test.check(cloudInfoer.GetZones(context.TODO(), test.region))
		})
	}
}

func TestAlibabaInfoer_GetProducts(t *testing.T) {
	tests := []struct {
		name           string
		ecsClient      EcsSource
		priceRetriever PriceRetriever
		region         string
		check          func(vms []cloudinfo.VmInfo, err error)
	}{
		{
			name:           "success",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{},
			region:         "us-east-1",
			check: func(vms []cloudinfo.VmInfo, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 2, len(vms))
			},
		},
		{
			name:           "could not retrieve virtual machines",
			ecsClient:      &testStruct{getVmsError},
			priceRetriever: &testStruct{},
			region:         "us-east-1",
			check: func(vms []cloudinfo.VmInfo, err error) {
				assert.Nil(t, vms, "the vms should be nil")
				assert.EqualError(t, err, externalApiError)
			},
		},
		{
			name:           "could not retrieve url",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{getPriceError},
			region:         "us-east-1",
			check: func(vms []cloudinfo.VmInfo, err error) {
				assert.Nil(t, vms, "the vms should be nil")
				assert.EqualError(t, err, externalApiError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloudInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			cloudInfoer.ecsClient = test.ecsClient
			cloudInfoer.priceRetriever = test.priceRetriever
			if err != nil {
				t.Fatalf("failed to create cloudinfoer; [%s]", err.Error())
			}

			test.check(cloudInfoer.GetProducts(context.TODO(), "compute", test.region))
		})
	}
}

func TestAlibabaInfoer_GetAttributeValues(t *testing.T) {
	tests := []struct {
		name           string
		ecsClient      EcsSource
		priceRetriever PriceRetriever
		attribute      string
		check          func(attrVal cloudinfo.AttrValues, err error)
	}{
		{
			name:           "success - memory",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{},
			attribute:      cloudinfo.Memory,
			check: func(attrVals cloudinfo.AttrValues, err error) {
				assert.Nil(t, err, "the error should be nil")
				var attr []float64

				for _, attrVal := range attrVals {
					attr = append(attr, attrVal.Value)
				}
				assert.ElementsMatch(t, attr, []float64{32, 128})
			},
		},
		{
			name:           "success - cpu",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{},
			attribute:      cloudinfo.Cpu,
			check: func(attrVals cloudinfo.AttrValues, err error) {
				assert.Nil(t, err, "the error should be nil")
				var attr []float64

				for _, attrVal := range attrVals {
					attr = append(attr, attrVal.Value)
				}
				assert.ElementsMatch(t, attr, []float64{8, 32})
			},
		},
		{
			name:           "could not retrieve regions",
			ecsClient:      &testStruct{getRegionsError},
			priceRetriever: &testStruct{},
			attribute:      cloudinfo.Cpu,
			check: func(attrVal cloudinfo.AttrValues, err error) {
				assert.Nil(t, attrVal, "the attribute should be nil")
				assert.EqualError(t, err, getRegionsError+": "+externalApiError)
			},
		},
		{
			name:           "could not retrieve virtual machines",
			ecsClient:      &testStruct{getVmsError},
			priceRetriever: &testStruct{},
			attribute:      cloudinfo.Cpu,
			check: func(attrVal cloudinfo.AttrValues, err error) {
				assert.Nil(t, attrVal, "the attribute should be nil")
				assert.EqualError(t, err, externalApiError)
			},
		},
		{
			name:           "could not retrieve url",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{getPriceError},
			attribute:      cloudinfo.Cpu,
			check: func(attrVal cloudinfo.AttrValues, err error) {
				assert.Nil(t, attrVal, "the attribute should be nil")
				assert.EqualError(t, err, externalApiError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloudInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			cloudInfoer.ecsClient = test.ecsClient
			cloudInfoer.priceRetriever = test.priceRetriever
			if err != nil {
				t.Fatalf("failed to create cloudinfoer; [%s]", err.Error())
			}

			test.check(cloudInfoer.GetAttributeValues(context.TODO(), "compute", test.attribute))
		})
	}
}

func TestAlibabaInfoer_GetCurrentPrices(t *testing.T) {
	tests := []struct {
		name           string
		ecsClient      EcsSource
		priceRetriever PriceRetriever
		spotClient     func(region string) EcsSource
		check          func(prices map[string]cloudinfo.Price, err error)
	}{
		{
			name:           "success",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{},
			spotClient: func(region string) EcsSource {
				return &testStruct{}
			},
			check: func(prices map[string]cloudinfo.Price, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 2, len(prices))
			},
		},
		{
			name:           "could not retrieve zones",
			ecsClient:      &testStruct{getZonesError},
			priceRetriever: &testStruct{},
			spotClient: func(region string) EcsSource {
				return &testStruct{}
			},
			check: func(prices map[string]cloudinfo.Price, err error) {
				assert.Nil(t, prices, "the prices should be nil")
				assert.EqualError(t, err, getZonesError+": "+externalApiError)
			},
		},
		{
			name:           "could not retrieve url",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{getPriceError},
			spotClient: func(region string) EcsSource {
				return &testStruct{}
			},
			check: func(prices map[string]cloudinfo.Price, err error) {
				assert.Nil(t, prices, "the prices should be nil")
				assert.EqualError(t, err, getPriceError+": "+externalApiError)
			},
		},
		{
			name:           "could not retrieve spot price history",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{},
			spotClient: func(region string) EcsSource {
				return &testStruct{getSpotPriceHistoryError}
			},
			check: func(prices map[string]cloudinfo.Price, err error) {
				assert.Nil(t, err, "the err should be nil")
				assert.Equal(t, map[string]cloudinfo.Price(map[string]cloudinfo.Price{}), prices)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloudInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			cloudInfoer.ecsClient = test.ecsClient
			cloudInfoer.priceRetriever = test.priceRetriever
			cloudInfoer.spotClient = test.spotClient
			if err != nil {
				t.Fatalf("failed to create cloudinfoer; [%s]", err.Error())
			}

			test.check(cloudInfoer.GetCurrentPrices(context.TODO(), "us-east-1"))
		})
	}
}
