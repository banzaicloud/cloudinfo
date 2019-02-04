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

package cloudinfo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/tracing"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/stretchr/testify/assert"
)

// DummyCloudInfoer type implements the CloudInfoer interface for mockig of external calls
// the struct is to be extended according to the needs of test cases
type DummyCloudInfoer struct {
	AttrValues AttrValues
	TcId       string
	//dummyNetworkMapper NetworkPerfMapper
	// implement the interface
	CloudInfoer
}

func newDummyNetworkMapper() dummyNetworkMapper {
	return dummyNetworkMapper{}
}

type dummyNetworkMapper struct {
}

func (nm *dummyNetworkMapper) MapNetworkPerf(ntwPerf string) (string, error) {
	return "high", nil
}

const (
	GetRegionsError         = "could not get regions"
	GetCurrentPricesError   = "could not get current prices"
	GetAttributeValuesError = "could not get attribute values"
	GetProductsError        = "could not get products"
	InitializeError         = "initialization failed"
	GetZonesError           = "could not get zones"
)

func (dpi *DummyCloudInfoer) Initialize(ctx context.Context) (map[string]map[string]Price, error) {
	switch dpi.TcId {
	case InitializeError:
		return nil, errors.New(InitializeError)
	default:
		return map[string]map[string]Price{
			"c1.xlarge":  {"dummy": {OnDemandPrice: 0.52, SpotPrice: SpotPriceInfo{"dummyZone1": 0.164}}},
			"c4.2xlarge": {"dummy": {OnDemandPrice: 0.4, SpotPrice: SpotPriceInfo{"dummyZone2": 0.12}}},
			"c3.large":   {"dummy": {OnDemandPrice: 0.11, SpotPrice: SpotPriceInfo{"dummyZone1": 0.053}}},
		}, nil
	}
}

func (dpi *DummyCloudInfoer) GetAttributeValues(ctx context.Context, service, attribute string) (AttrValues, error) {
	switch dpi.TcId {
	case GetAttributeValuesError:
		return nil, errors.New(GetAttributeValuesError)
	}
	return dpi.AttrValues, nil
}

func (dpi *DummyCloudInfoer) GetProducts(ctx context.Context, service, regionId string) ([]VmInfo, error) {
	switch dpi.TcId {
	case GetProductsError:
		return nil, errors.New(GetProductsError)
	default:
		return []VmInfo{
			{Cpus: float64(2),
				Mem:           float64(32),
				OnDemandPrice: float64(0.32)},
		}, nil
	}
}

func (dpi *DummyCloudInfoer) GetZones(ctx context.Context, region string) ([]string, error) {
	switch dpi.TcId {
	case GetZonesError:
		return nil, errors.New(GetZonesError)
	default:
		return []string{"dummyZone1", "dummyZone2"}, nil
	}
}

func (dpi *DummyCloudInfoer) GetRegion(id string) *endpoints.Region {
	return nil
}

func (dpi *DummyCloudInfoer) GetRegions(ctx context.Context, service string) (map[string]string, error) {
	switch dpi.TcId {
	case GetRegionsError:
		return nil, errors.New(GetRegionsError)
	default:
		return map[string]string{
			"EU (Frankfurt)":   "eu-central-1",
			"EU (Ireland)":     "eu-west-1",
			"US West (Oregon)": "us-west-2",
		}, nil
	}
}

func (dpi *DummyCloudInfoer) HasShortLivedPriceInfo() bool {
	return true
}

func (dpi *DummyCloudInfoer) GetCurrentPrices(ctx context.Context, region string) (map[string]Price, error) {
	switch dpi.TcId {
	case GetCurrentPricesError:
		return nil, errors.New(GetCurrentPricesError)
	default:
		return map[string]Price{
			"c1.xlarge":  {OnDemandPrice: 0.52, SpotPrice: SpotPriceInfo{"dummyZone1": 0.164}},
			"c4.2xlarge": {OnDemandPrice: 0.4, SpotPrice: SpotPriceInfo{"dummyZone2": 0.12}},
			"c3.large":   {OnDemandPrice: 0.11, SpotPrice: SpotPriceInfo{"dummyZone1": 0.053}},
		}, nil
	}

}

func (dpi *DummyCloudInfoer) GetMemoryAttrName() string {
	return "memory"
}

func (dpi *DummyCloudInfoer) GetCpuAttrName() string {
	return "vcpu"
}

func (dpi *DummyCloudInfoer) GetNetworkPerformanceMapper() (NetworkPerfMapper, error) {
	nm := newDummyNetworkMapper()
	return &nm, nil
}

func (dpi *DummyCloudInfoer) MapNetworkPerf(vm VmInfo) (string, error) {
	return "high", nil
}

func TestNewCachingCloudInfo(t *testing.T) {
	tests := []struct {
		Name        string
		CloudInfoer map[string]CloudInfoer
		checker     func(info *cachingCloudInfo, err error)
	}{
		{
			Name: "product info successfully created",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{},
			},
			checker: func(info *cachingCloudInfo, err error) {
				assert.Nil(t, err, "should not get error")
				assert.NotNil(t, info, "the product info should not be nil")
			},
		},
		{
			Name:        "validation should fail nil values",
			CloudInfoer: nil,
			checker: func(info *cachingCloudInfo, err error) {
				assert.Nil(t, info, "the cloudinfo should be nil in case of error")
				assert.EqualError(t, err, "could not create product infoer")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test.checker(NewCachingCloudInfo(NewCacheProductStore(10*time.Minute, 5*time.Minute, logur.NewTestLogger()), test.CloudInfoer, metrics.NewNoOpMetricsReporter(), tracing.NewNoOpTracer()))
		})
	}

}

func TestCachingCloudInfo_GetAttrValues(t *testing.T) {
	dummyAttrValues := AttrValues{
		AttrValue{Value: 15},
		AttrValue{Value: 16},
		AttrValue{Value: 17},
	}
	tests := []struct {
		name        string
		CloudInfoer map[string]CloudInfoer
		Attribute   string
		checker     func(value []float64, err error)
	}{
		{
			name: "successfully returned the attribute values for cpu",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{AttrValues: dummyAttrValues}},
			Attribute: Cpu,
			checker: func(value []float64, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, []float64{15, 16, 17}, value)
			},
		},
		{
			name: "successfully returned the attribute values for memory",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{AttrValues: dummyAttrValues}},
			Attribute: Memory,
			checker: func(value []float64, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, []float64{15, 16, 17}, value)
			},
		},
		{
			name: "the specified attribute is not supported",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{AttrValues: dummyAttrValues}},
			Attribute: "invalidAttribute",
			checker: func(value []float64, err error) {
				assert.Equal(t, emperror.Context(err)[0], "provider", "unexpected context")
				assert.Equal(t, emperror.Context(err)[1], "dummy", "unexpected context")
				assert.EqualError(t, err, "failed to retrieve attribute values: unsupported attribute")
				assert.Nil(t, value, "the retrieved values should be nil")
			},
		},
		{
			name: "could not retrieve attribute values",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{TcId: GetAttributeValuesError, AttrValues: dummyAttrValues}},
			Attribute: Cpu,
			checker: func(value []float64, err error) {
				assert.EqualError(t, err, "failed to retrieve attribute values: "+GetAttributeValuesError)
				assert.Nil(t, value, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger.Init(logur.NewTestLogger())
			cloudInfo, _ := NewCachingCloudInfo(NewCacheProductStore(5*time.Minute, 10*time.Minute, logur.NewTestLogger()), test.CloudInfoer, metrics.NewNoOpMetricsReporter(), tracing.NewNoOpTracer())
			test.checker(cloudInfo.GetAttrValues(context.Background(), "dummy", "dummyService", test.Attribute))
		})
	}
}

func TestCachingCloudInfo_Initialize(t *testing.T) {
	tests := []struct {
		name        string
		CloudInfoer map[string]CloudInfoer
		checker     func(price map[string]map[string]Price, err error)
	}{
		{
			name: "successful - store the result in cache",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{},
			},
			checker: func(price map[string]map[string]Price, err error) {
				assert.Equal(t, 3, len(price))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "could not get the output of the Infoer's Initialize function",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{TcId: InitializeError},
			},
			checker: func(price map[string]map[string]Price, err error) {
				assert.Nil(t, price, "the price should be nil")
				assert.EqualError(t, err, InitializeError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloudInfo, _ := NewCachingCloudInfo(NewCacheProductStore(5*time.Minute, 10*time.Minute, logur.NewTestLogger()), test.CloudInfoer, metrics.NewNoOpMetricsReporter(), tracing.NewNoOpTracer())
			test.checker(cloudInfo.Initialize(context.Background(), "dummy"))
		})
	}
}

func TestCachingCloudInfo_renewShortLivedInfo(t *testing.T) {
	tests := []struct {
		name        string
		CloudInfoer map[string]CloudInfoer
		checker     func(price map[string]Price, err error)
	}{
		{
			name: "successful - retrieve attribute values from the cloud provider",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{},
			},
			checker: func(price map[string]Price, err error) {
				assert.Equal(t, 3, len(price))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "could not retrieve current prices",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{TcId: GetCurrentPricesError},
			},
			checker: func(price map[string]Price, err error) {
				assert.Nil(t, price, "the price should be nil")
				assert.EqualError(t, err, GetCurrentPricesError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, _ := NewCachingCloudInfo(NewCacheProductStore(5*time.Minute, 10*time.Minute, logur.NewTestLogger()), test.CloudInfoer, metrics.NewNoOpMetricsReporter(), tracing.NewNoOpTracer())
			test.checker(info.renewShortLivedInfo(context.Background(), "dummy", "dummyRegion"))
		})
	}
}

func TestCachingCloudInfo_GetPrice(t *testing.T) {
	tests := []struct {
		name        string
		zones       []string
		CloudInfoer map[string]CloudInfoer
		checker     func(i float64, f float64, err error)
	}{
		{
			name:  "return on demand price and average spot price with 1 zone",
			zones: []string{"dummyZone1"},
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{},
			},
			checker: func(ondemand float64, avg float64, err error) {
				assert.Equal(t, float64(0.11), ondemand)
				assert.Equal(t, float64(0.053), avg)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:  "return on demand price and average spot price with 4 zones",
			zones: []string{"dummyZone1", "dummyZone2", "dummyZone3", "dummyZone4"},
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{},
			},
			checker: func(ondemand float64, avg float64, err error) {
				assert.Equal(t, float64(0.11), ondemand)
				assert.Equal(t, float64(0.01325), avg)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:  "return on demand price and average spot price without expected zone",
			zones: []string{"dummyZone2", "dummyZone3", "dummyZone4"},
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{},
			},
			checker: func(ondemand float64, avg float64, err error) {
				assert.Equal(t, float64(0.11), ondemand)
				assert.Equal(t, float64(0), avg)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:  "could not retrieve current prices",
			zones: []string{"dummyZone1"},
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{TcId: GetCurrentPricesError},
			},
			checker: func(i float64, f float64, err error) {
				assert.Equal(t, float64(0), i)
				assert.Equal(t, float64(0), f)
				assert.EqualError(t, err, GetCurrentPricesError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, _ := NewCachingCloudInfo(NewCacheProductStore(5*time.Minute, 10*time.Minute, logur.NewTestLogger()), test.CloudInfoer, metrics.NewNoOpMetricsReporter(), tracing.NewNoOpTracer())
			values, value, err := info.GetPrice(context.Background(), "dummy", "dummyRegion", "c3.large", test.zones)
			test.checker(values, value, err)
		})
	}
}

func TestCachingCloudInfo_GetRegions(t *testing.T) {
	tests := []struct {
		name        string
		CloudInfoer map[string]CloudInfoer
		checker     func(regions map[string]string, err error)
	}{
		{
			name: "successfully retrieved the regions",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{},
			},
			checker: func(regions map[string]string, err error) {
				assert.Equal(t, map[string]string(map[string]string{"US West (Oregon)": "us-west-2", "EU (Frankfurt)": "eu-central-1", "EU (Ireland)": "eu-west-1"}), regions)
				assert.Equal(t, 3, len(regions))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "could not retrieve regions",
			CloudInfoer: map[string]CloudInfoer{
				"dummy": &DummyCloudInfoer{TcId: GetRegionsError},
			},
			checker: func(regions map[string]string, err error) {
				assert.Nil(t, regions, "the error should be nil")
				assert.EqualError(t, err, GetRegionsError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, _ := NewCachingCloudInfo(NewCacheProductStore(5*time.Minute, 10*time.Minute, logur.NewTestLogger()), test.CloudInfoer, metrics.NewNoOpMetricsReporter(), tracing.NewNoOpTracer())
			test.checker(info.GetRegions(context.Background(), "dummy", "compute"))
		})
	}
}
