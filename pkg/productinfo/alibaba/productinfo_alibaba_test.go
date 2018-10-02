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
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"github.com/stretchr/testify/assert"
	"testing"
)

//testStruct helps to mock external calls
type testStruct struct {
	TcId string
}

const (
	GetRegionsError          = "could not get regions"
	GetVmsError              = "could not get virtual machines"
	GetZonesError            = "could not get zones"
	GetUrlError              = "could not get url"
	GetSpotPriceHistoryError = "could not get spot price"
)

func (dps *testStruct) DescribeInstanceTypes(request *ecs.DescribeInstanceTypesRequest) (response *ecs.DescribeInstanceTypesResponse, err error) {
	switch dps.TcId {
	case GetVmsError:
		return &ecs.DescribeInstanceTypesResponse{}, fmt.Errorf(GetVmsError)
	default:
		return &ecs.DescribeInstanceTypesResponse{
			InstanceTypes: ecs.InstanceTypesInDescribeInstanceTypes{
				InstanceType: []ecs.InstanceType{
					{
						InstanceTypeId:      "ecs.g5.2xlarge",
						CpuCoreCount:        8,
						MemorySize:          32.0,
						InstanceBandwidthRx: 25000,
					},
					{
						InstanceTypeId:      "ecs.g5.8xlarge",
						CpuCoreCount:        32,
						MemorySize:          128.0,
						InstanceBandwidthRx: 10000,
					},
					{
						InstanceTypeId:      "ecs.sn2ne.8xlarge",
						CpuCoreCount:        32,
						MemorySize:          128.0,
						InstanceBandwidthRx: 25000,
					},
					{
						InstanceTypeId:      "ecs.ic5.3xlarge",
						CpuCoreCount:        12,
						MemorySize:          12.0,
						InstanceBandwidthRx: 4000,
					},
				},
			},
		}, nil
	}
}

func (dps *testStruct) DescribeSpotPriceHistory(request *ecs.DescribeSpotPriceHistoryRequest) (response *ecs.DescribeSpotPriceHistoryResponse, err error) {
	switch dps.TcId {
	case GetSpotPriceHistoryError:
		return &ecs.DescribeSpotPriceHistoryResponse{}, fmt.Errorf(GetSpotPriceHistoryError)
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
	case GetZonesError:
		return &ecs.DescribeZonesResponse{}, fmt.Errorf(GetZonesError)
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
	case GetRegionsError:
		return &ecs.DescribeRegionsResponse{}, fmt.Errorf(GetRegionsError)
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
	case GetUrlError:
		return OnDemandPrice{}, fmt.Errorf(GetUrlError)
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
			client: &testStruct{GetRegionsError},
			check: func(regions map[string]string, err error) {
				assert.Nil(t, regions, "the regions should be nil")
				assert.EqualError(t, err, GetRegionsError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			productInfoer.ecsClient = test.client
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetRegions(context.TODO(), "dummyService"))
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
			client: &testStruct{GetZonesError},
			region: "eu-central-1",
			check: func(zones []string, err error) {
				assert.Nil(t, zones, "the regions should be nil")
				assert.EqualError(t, err, GetZonesError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			productInfoer.ecsClient = test.client
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetZones(context.TODO(), test.region))
		})
	}
}

func TestAlibabaInfoer_GetProducts(t *testing.T) {
	tests := []struct {
		name           string
		ecsClient      EcsSource
		priceRetriever PriceRetriever
		region         string
		check          func(vms []productinfo.VmInfo, err error)
	}{
		{
			name:           "success",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{},
			region:         "us-east-1",
			check: func(vms []productinfo.VmInfo, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 2, len(vms))
			},
		},
		{
			name:           "could not retrieve virtual machines",
			ecsClient:      &testStruct{GetVmsError},
			priceRetriever: &testStruct{},
			region:         "us-east-1",
			check: func(vms []productinfo.VmInfo, err error) {
				assert.Nil(t, vms, "the vms should be nil")
				assert.EqualError(t, err, GetVmsError)
			},
		},
		{
			name:           "could not retrieve url",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{GetUrlError},
			region:         "us-east-1",
			check: func(vms []productinfo.VmInfo, err error) {
				assert.Nil(t, vms, "the vms should be nil")
				assert.EqualError(t, err, GetUrlError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			productInfoer.ecsClient = test.ecsClient
			productInfoer.priceRetriever = test.priceRetriever
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetProducts(context.TODO(), "compute", test.region))
		})
	}
}

func TestAlibabaInfoer_GetAttributeValues(t *testing.T) {
	tests := []struct {
		name           string
		ecsClient      EcsSource
		priceRetriever PriceRetriever
		attribute      string
		check          func(attrVal productinfo.AttrValues, err error)
	}{
		{
			name:           "success - memory",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{},
			attribute:      productinfo.Memory,
			check: func(attrVals productinfo.AttrValues, err error) {
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
			attribute:      productinfo.Cpu,
			check: func(attrVals productinfo.AttrValues, err error) {
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
			ecsClient:      &testStruct{GetRegionsError},
			priceRetriever: &testStruct{},
			attribute:      productinfo.Cpu,
			check: func(attrVal productinfo.AttrValues, err error) {
				assert.Nil(t, attrVal, "the attribute should be nil")
				assert.EqualError(t, err, GetRegionsError)
			},
		},
		{
			name:           "could not retrieve virtual machines",
			ecsClient:      &testStruct{GetVmsError},
			priceRetriever: &testStruct{},
			attribute:      productinfo.Cpu,
			check: func(attrVal productinfo.AttrValues, err error) {
				assert.Nil(t, attrVal, "the attribute should be nil")
				assert.EqualError(t, err, GetVmsError)
			},
		},
		{
			name:           "could not retrieve url",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{GetUrlError},
			attribute:      productinfo.Cpu,
			check: func(attrVal productinfo.AttrValues, err error) {
				assert.Nil(t, attrVal, "the attribute should be nil")
				assert.EqualError(t, err, GetUrlError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			productInfoer.ecsClient = test.ecsClient
			productInfoer.priceRetriever = test.priceRetriever
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetAttributeValues(context.TODO(), "compute", test.attribute))
		})
	}
}

func TestAlibabaInfoer_GetCurrentPrices(t *testing.T) {
	tests := []struct {
		name           string
		ecsClient      EcsSource
		priceRetriever PriceRetriever
		spotClient     func(region string) EcsSource
		region         string
		check          func(prices map[string]productinfo.Price, err error)
	}{
		{
			name:           "success",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{},
			spotClient: func(region string) EcsSource {
				return &testStruct{}
			},
			check: func(prices map[string]productinfo.Price, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 2, len(prices))
			},
		},
		{
			name:           "could not retrieve zones",
			ecsClient:      &testStruct{GetZonesError},
			priceRetriever: &testStruct{},
			spotClient: func(region string) EcsSource {
				return &testStruct{}
			},
			check: func(prices map[string]productinfo.Price, err error) {
				assert.Nil(t, prices, "the prices should be nil")
				assert.EqualError(t, err, GetZonesError)
			},
		},
		{
			name:           "could not retrieve url",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{GetUrlError},
			spotClient: func(region string) EcsSource {
				return &testStruct{}
			},
			check: func(prices map[string]productinfo.Price, err error) {
				assert.Nil(t, prices, "the prices should be nil")
				assert.EqualError(t, err, GetUrlError)
			},
		},
		{
			name:           "could not retrieve spot price history",
			ecsClient:      &testStruct{},
			priceRetriever: &testStruct{},
			spotClient: func(region string) EcsSource {
				return &testStruct{GetSpotPriceHistoryError}
			},
			check: func(prices map[string]productinfo.Price, err error) {
				assert.Nil(t, err, "the err should be nil")
				assert.Equal(t, map[string]productinfo.Price(map[string]productinfo.Price{}), prices)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfoer, err := NewAlibabaInfoer("", "", "")
			// override pricingSvc
			productInfoer.ecsClient = test.ecsClient
			productInfoer.priceRetriever = test.priceRetriever
			productInfoer.spotClient = test.spotClient
			if err != nil {
				t.Fatalf("failed to create productinfoer; [%s]", err.Error())
			}

			test.check(productInfoer.GetCurrentPrices(context.TODO(), "us-east-1"))
		})
	}
}
