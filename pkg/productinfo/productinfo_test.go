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

package productinfo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
)

// DummyProductInfoer type implements the ProductInfoer interface for mockig of external calls
// the struct is to be extended according to the needs of test cases
type DummyProductInfoer struct {
	AttrValues AttrValues
	Vms        []VmInfo
	TcId       string
	//dummyNetworkMapper NetworkPerfMapper
	// implement the interface
	ProductInfoer
	ProductStorer
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
	ProductDetailsOK        = "successfully get product details"
	GetProductDetail        = "returns a product detail"
)

func (dpi *DummyProductInfoer) Initialize(ctx context.Context) (map[string]map[string]Price, error) {
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

func (dpi *DummyProductInfoer) GetAttributeValues(ctx context.Context, service, attribute string) (AttrValues, error) {
	switch dpi.TcId {
	case GetAttributeValuesError:
		return nil, errors.New(GetAttributeValuesError)
	}
	return dpi.AttrValues, nil
}

func (dpi *DummyProductInfoer) GetProducts(ctx context.Context, service, regionId string) ([]VmInfo, error) {
	switch dpi.TcId {
	case GetProductsError:
		return nil, errors.New(GetProductsError)
	default:
		return dpi.Vms, nil
	}
}

func (dpi *DummyProductInfoer) GetZones(ctx context.Context, region string) ([]string, error) {
	switch dpi.TcId {
	case GetZonesError:
		return nil, errors.New(GetZonesError)
	default:
		return []string{"dummyZone1", "dummyZone2"}, nil
	}
}

func (dpi *DummyProductInfoer) GetRegion(id string) *endpoints.Region {
	return nil
}

func (dpi *DummyProductInfoer) GetRegions(ctx context.Context, service string) (map[string]string, error) {
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

func (dpi *DummyProductInfoer) HasShortLivedPriceInfo() bool {
	return true
}

func (dpi *DummyProductInfoer) GetCurrentPrices(ctx context.Context, region string) (map[string]Price, error) {
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

func (dpi *DummyProductInfoer) GetMemoryAttrName() string {
	return "memory"
}

func (dpi *DummyProductInfoer) GetCpuAttrName() string {
	return "vcpu"
}

func (dpi *DummyProductInfoer) Get(k string) (interface{}, bool) {
	switch dpi.TcId {
	case ProductDetailsOK:
		switch k {
		case "/banzaicloud.com/productinfo/providers/dummy/services/dummyService/regions/dummyRegion/vms":
			return []VmInfo{
				{
					Type:          "type-1",
					OnDemandPrice: 0.021,
					Cpus:          1,
					Mem:           2,
					NtwPerfCat:    "high",
					SpotPrice:     SpotPriceInfo{"dummy": 0.006},
				},
				{
					Type:       "type-2",
					Cpus:       2,
					Mem:        4,
					NtwPerfCat: "high",
				},
				{
					Type:       "type-3",
					Cpus:       2,
					Mem:        4,
					NtwPerfCat: "high",
				},
			}, true
		case "/banzaicloud.com/productinfo/providers/dummy/regions/dummyRegion/prices/type-1":
			return Price{
				OnDemandPrice: 0.023,
				SpotPrice:     SpotPriceInfo{"dummyZone": 0.0069},
			}, true
		case "/banzaicloud.com/productinfo/providers/dummy/regions/dummyRegion/prices/type-2":
			return Price{
				OnDemandPrice: 0.043,
				SpotPrice:     SpotPriceInfo{"dummyZone": 0.0087},
			}, true
		default:
			return nil, false
		}
	case GetProductDetail:
		switch k {
		case "/banzaicloud.com/productinfo/providers/dummy/services/dummyService/regions/dummyRegion/vms":
			return []VmInfo{
				{
					Type:          "type-1",
					OnDemandPrice: 0.021,
					Cpus:          1,
					Mem:           2,
					NtwPerfCat:    "high",
				},
			}, true
		case "/banzaicloud.com/productinfo/providers/dummy/regions/dummyRegion/prices/type-1":
			return Price{
				OnDemandPrice: 0.023,
				SpotPrice:     SpotPriceInfo{"dummyZone": 0.0069},
			}, true
		default:
			return nil, false
		}
	default:
		return nil, false
	}
}

func (dpi *DummyProductInfoer) Set(k string, x interface{}, d time.Duration) {
}

func (dpi *DummyProductInfoer) GetNetworkPerformanceMapper() (NetworkPerfMapper, error) {
	nm := newDummyNetworkMapper()
	return &nm, nil
}

func (dpi *DummyProductInfoer) MapNetworkPerf(vm VmInfo) (string, error) {
	return "high", nil
}

func TestNewCachingProductInfo(t *testing.T) {
	tests := []struct {
		Name          string
		ProductInfoer map[string]ProductInfoer
		checker       func(info *CachingProductInfo, err error)
	}{
		{
			Name: "product info successfully created",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(info *CachingProductInfo, err error) {
				assert.Nil(t, err, "should not get error")
				assert.NotNil(t, info, "the product info should not be nil")
			},
		},
		{
			Name:          "validation should fail nil values",
			ProductInfoer: nil,
			checker: func(info *CachingProductInfo, err error) {
				assert.Nil(t, info, "the productinfo should be nil in case of error")
				assert.EqualError(t, err, "could not create product infoer")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			test.checker(NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer))
		})
	}

}

func TestCachingProductInfo_renewVms(t *testing.T) {
	tests := []struct {
		name          string
		attrValue     AttrValue
		ProductInfoer map[string]ProductInfoer
		Cache         *cache.Cache
		checker       func(cache *cache.Cache, vms []VmInfo, err error)
	}{
		{
			name:      "vm successfully renewed",
			attrValue: AttrValue{Value: float64(2), StrValue: Cpu},
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					Vms: []VmInfo{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
				},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			checker: func(cache *cache.Cache, vms []VmInfo, err error) {
				assert.Nil(t, err, "should not get error on vm renewal")
				assert.Equal(t, 1, len(vms), "there should be a single entry in values")
				vals, _ := cache.Get("/banzaicloud.com/productinfo/providers/dummy/services/dummyService/regions/dummyRegion/vms")

				for _, val := range vals.([]VmInfo) {
					assert.Equal(t, float64(32), val.Mem, "the value in the cache is not as expected")
				}

			},
		},
		{
			name:      "could not retrieve virtual machines",
			attrValue: AttrValue{Value: float64(2), StrValue: Cpu},
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{
					TcId: GetProductsError,
					Vms:  []VmInfo{{Cpus: float64(2), Mem: float64(32), OnDemandPrice: float64(0.32)}},
				},
			},
			Cache: cache.New(5*time.Minute, 10*time.Minute),
			checker: func(cache *cache.Cache, vms []VmInfo, err error) {
				assert.EqualError(t, err, GetProductsError)
				assert.Nil(t, vms, "no vms expected")

			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.Cache, test.ProductInfoer)
			values, err := productInfo.renewVms(context.Background(), "dummy", "dummyService", "dummyRegion")
			test.checker(test.Cache, values, err)
		})
	}
}

func TestCachingProductInfo_GetAttrValues(t *testing.T) {
	dummyAttrValues := AttrValues{
		AttrValue{Value: 15},
		AttrValue{Value: 16},
		AttrValue{Value: 17},
	}
	tests := []struct {
		name          string
		ProductInfoer map[string]ProductInfoer
		Attribute     string
		checker       func(value []float64, err error)
	}{
		{
			name: "successfully returned the attribute values for cpu",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{AttrValues: dummyAttrValues}},
			Attribute: Cpu,
			checker: func(value []float64, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, []float64{15, 16, 17}, value)
			},
		},
		{
			name: "successfully returned the attribute values for memory",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{AttrValues: dummyAttrValues}},
			Attribute: Memory,
			checker: func(value []float64, err error) {
				assert.Nil(t, err, "the returned error must be nil")
				assert.Equal(t, []float64{15, 16, 17}, value)
			},
		},
		{
			name: "the specified attribute is not supported",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{AttrValues: dummyAttrValues}},
			Attribute: "invalidAttribute",
			checker: func(value []float64, err error) {
				assert.EqualError(t, err, "unsupported attribute: invalidAttribute")
				assert.Nil(t, value, "the retrieved values should be nil")
			},
		},
		{
			name: "could not retrieve attribute values",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: GetAttributeValuesError, AttrValues: dummyAttrValues}},
			Attribute: Cpu,
			checker: func(value []float64, err error) {
				assert.EqualError(t, err, GetAttributeValuesError)
				assert.Nil(t, value, "the retrieved values should be nil")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			test.checker(productInfo.GetAttrValues(context.Background(), "dummy", "dummyService", test.Attribute))
		})
	}
}

func TestCachingProductInfo_GetZones(t *testing.T) {
	tests := []struct {
		name          string
		ProductInfoer map[string]ProductInfoer
		checker       func(cpi *CachingProductInfo, zones []string, err error)
	}{
		{
			name: "zones retrieved and cached",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(cpi *CachingProductInfo, zones []string, err error) {
				assert.Equal(t, []string{"dummyZone1", "dummyZone2"}, zones)
				assert.Nil(t, err, "the error should be nil")

				// get the values from the cache
				cachedZones, _ := cpi.vmAttrStore.Get(cpi.getZonesKey("dummy", "dummyRegion"))
				assert.EqualValues(t, []string{"dummyZone1", "dummyZone2"}, cachedZones, "zones not cached")
			},
		},
		{
			name: "could not retrieve zones",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: GetZonesError},
			},
			checker: func(cpi *CachingProductInfo, zones []string, err error) {
				assert.Nil(t, zones, "the error should be nil")
				assert.EqualError(t, err, GetZonesError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values, err := productInfo.GetZones(context.Background(), "dummy", "dummyRegion")
			test.checker(productInfo, values, err)
		})
	}
}

func TestCachingProductInfo_Initialize(t *testing.T) {
	tests := []struct {
		name          string
		ProductInfoer map[string]ProductInfoer
		checker       func(price map[string]map[string]Price, err error)
	}{
		{
			name: "successful - store the result in cache",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(price map[string]map[string]Price, err error) {
				assert.Equal(t, 3, len(price))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "could not get the output of the Infoer's Initialize function",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: InitializeError},
			},
			checker: func(price map[string]map[string]Price, err error) {
				assert.Nil(t, price, "the price should be nil")
				assert.EqualError(t, err, InitializeError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			test.checker(productInfo.Initialize(context.Background(), "dummy"))
		})
	}
}

func TestCachingProductInfo_renewShortLivedInfo(t *testing.T) {
	tests := []struct {
		name          string
		ProductInfoer map[string]ProductInfoer
		checker       func(price map[string]Price, err error)
	}{
		{
			name: "successful - retrieve attribute values from the cloud provider",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(price map[string]Price, err error) {
				assert.Equal(t, 3, len(price))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "could not retrieve current prices",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: GetCurrentPricesError},
			},
			checker: func(price map[string]Price, err error) {
				assert.Nil(t, price, "the price should be nil")
				assert.EqualError(t, err, GetCurrentPricesError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			test.checker(productInfo.renewShortLivedInfo(context.Background(), "dummy", "dummyRegion"))
		})
	}
}

func TestCachingProductInfo_GetPrice(t *testing.T) {
	tests := []struct {
		name          string
		zones         []string
		ProductInfoer map[string]ProductInfoer
		checker       func(i float64, f float64, err error)
	}{
		{
			name:  "return on demand price and average spot price with 1 zone",
			zones: []string{"dummyZone1"},
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
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
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
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
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
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
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: GetCurrentPricesError},
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
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			values, value, err := productInfo.GetPrice(context.Background(), "dummy", "dummyRegion", "c3.large", test.zones)
			test.checker(values, value, err)
		})
	}
}

func TestCachingProductInfo_GetRegions(t *testing.T) {
	tests := []struct {
		name          string
		ProductInfoer map[string]ProductInfoer
		checker       func(regions map[string]string, err error)
	}{
		{
			name: "successfully retrieved the regions",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			checker: func(regions map[string]string, err error) {
				assert.Equal(t, map[string]string(map[string]string{"US West (Oregon)": "us-west-2", "EU (Frankfurt)": "eu-central-1", "EU (Ireland)": "eu-west-1"}), regions)
				assert.Equal(t, 3, len(regions))
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name: "could not retrieve regions",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{TcId: GetRegionsError},
			},
			checker: func(regions map[string]string, err error) {
				assert.Nil(t, regions, "the error should be nil")
				assert.EqualError(t, err, GetRegionsError)
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, cache.New(5*time.Minute, 10*time.Minute), test.ProductInfoer)
			test.checker(productInfo.GetRegions(context.Background(), "dummy", "compute"))
		})
	}
}

func TestCachingProductInfo_GetProductDetails(t *testing.T) {
	tests := []struct {
		name          string
		ProductInfoer map[string]ProductInfoer
		cache         ProductStorer
		checker       func(details []ProductDetails, err error)
	}{
		{
			name: "successfully retrieved product details",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			cache: &DummyProductInfoer{TcId: ProductDetailsOK},
			checker: func(details []ProductDetails, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 2, len(details))
			},
		},
		{
			name: "successfully retrieved one product detail",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			cache: &DummyProductInfoer{TcId: GetProductDetail},
			checker: func(details []ProductDetails, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 1, len(details))
				for _, info := range details {
					assert.Equal(t, "type-1", info.Type)
					assert.Equal(t, float64(1), info.Cpus)
					assert.Equal(t, 0.023, info.OnDemandPrice)
					assert.Equal(t, float64(2), info.Mem)
				}
			},
		},
		{
			name: "vms not yet cached, we need to get an error",
			ProductInfoer: map[string]ProductInfoer{
				"dummy": &DummyProductInfoer{},
			},
			cache: &DummyProductInfoer{},
			checker: func(details []ProductDetails, err error) {
				assert.Nil(t, details, "the details should be nil")
				assert.EqualError(t, err, "vms not yet cached for the key: /banzaicloud.com/productinfo/providers/dummy/services/dummyService/regions/dummyRegion/vms")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			productInfo, _ := NewCachingProductInfo(10*time.Second, test.cache, test.ProductInfoer)
			test.checker(productInfo.GetProductDetails(context.Background(), "dummy", "dummyService", "dummyRegion"))
		})
	}
}
