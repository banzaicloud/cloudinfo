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

package amazon

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/stretchr/testify/assert"
	"logur.dev/logur"

	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/cloudinfoadapter"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/types"
)

// testStruct helps to mock external calls
type testStruct struct {
	TcId int
}

func (dps *testStruct) GetPriceList(input *pricing.GetProductsInput) ([]aws.JSONValue, error) {
	switch dps.TcId {
	case 4:
		return []aws.JSONValue{
			{
				"product": map[string]interface{}{
					"attributes": map[string]interface{}{
						"instanceType":       ec2.InstanceTypeT2Small,
						"vcpu":               "1",
						types.Memory:         "2",
						"networkPerformance": "Low to Moderate",
					}},
				"terms": map[string]interface{}{
					"OnDemand": map[string]interface{}{
						"randomNumber": map[string]interface{}{
							"priceDimensions": map[string]interface{}{
								"randomNumber": map[string]interface{}{
									"pricePerUnit": map[string]interface{}{
										"USD": "5",
									}}}}}}},
		}, nil
	case 5:
		return nil, errors.New("failed to retrieve values")
	case 6:
		return []aws.JSONValue{
			{
				"product": map[string]interface{}{
					"attributes": map[string]interface{}{
						"instanceType": ec2.InstanceTypeT2Small,
						"vcpu":         "1",
						types.Memory:   "2",
					}},
				"terms": map[string]interface{}{
					"OnDemand": map[string]interface{}{
						"randomNumber": map[string]interface{}{
							"priceDimensions": map[string]interface{}{
								"randomNumber": map[string]interface{}{
									"pricePerUnit": map[string]interface{}{},
								}}}}}},
		}, nil
	case 7:
		return []aws.JSONValue{
			{
				"product": map[string]interface{}{
					"attributes": map[string]interface{}{
						"instanceType": ec2.InstanceTypeT2Small,
						"vcpu":         "1",
					}}},
		}, nil
	case 8:
		return []aws.JSONValue{
			{
				"product": map[string]interface{}{
					"attributes": map[string]interface{}{
						"instanceType": ec2.InstanceTypeT2Small,
					}}},
		}, nil
	case 9:
		return []aws.JSONValue{
			{
				"product": map[string]interface{}{
					"attributes": map[string]interface{}{},
				}},
		}, nil
	}
	return nil, nil
}

func (dps *testStruct) DescribeAvailabilityZones(input *ec2.DescribeAvailabilityZonesInput) (*ec2.DescribeAvailabilityZonesOutput, error) {
	if dps.TcId == 10 {
		return nil, errors.New("could not get information about zones")
	}
	return &ec2.DescribeAvailabilityZonesOutput{
		AvailabilityZones: []*ec2.AvailabilityZone{
			{
				State:      aws.String(ec2.AvailabilityZoneStateAvailable),
				RegionName: aws.String("eu-central-1"),
				ZoneName:   aws.String("eu-central-1a"),
			},
			{
				State:      aws.String("available"),
				RegionName: aws.String("eu-central-1"),
				ZoneName:   aws.String("eu-central-1b"),
			},
		},
	}, nil
}

func (dps *testStruct) DescribeImages(*ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
	return nil, nil
}

func (dps *testStruct) DescribeSpotPriceHistoryPages(input *ec2.DescribeSpotPriceHistoryInput, fn func(*ec2.DescribeSpotPriceHistoryOutput, bool) bool) error {
	if dps.TcId == 11 {
		return errors.New("invalid")
	}
	return nil
}

func TestNewEc2Infoer(t *testing.T) {
	tests := []struct {
		name  string
		prom  string
		check func(info *Ec2Infoer, err error)
	}{
		{
			name: "create Ec2Infoer - Prometheus API address is not set",
			prom: "",
			check: func(info *Ec2Infoer, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.NotNil(t, info, "the Ec2Infoer should not be nil")
			},
		},
		{
			name: "create Ec2Infoer - Prometheus API address is set",
			prom: "PromAPIAddress",
			check: func(info *Ec2Infoer, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.NotNil(t, info, "the Ec2Infoer should not be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := Config{PrometheusAddress: test.prom}

			test.check(NewAmazonInfoer(c, cloudinfoadapter.NewLogger(&logur.TestLogger{})))
		})
	}
}

func TestEc2Infoer_GetRegion(t *testing.T) {
	tests := []struct {
		name           string
		id             string
		pricingService PricingSource
		check          func(region *endpoints.Region)
	}{
		{
			name:           "returns data of a known region",
			id:             "eu-west-3",
			pricingService: &testStruct{},
			check: func(region *endpoints.Region) {
				assert.Equal(t, "Europe (Paris)", region.Description())
				assert.Equal(t, "eu-west-3", region.ID())
			},
		},
		{
			name:           "get an unknown region",
			id:             "unknownRegion",
			pricingService: &testStruct{},
			check: func(region *endpoints.Region) {
				assert.Nil(t, region, "the region should be nil")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloudInfoer, err := NewAmazonInfoer(Config{}, cloudinfoadapter.NewLogger(&logur.TestLogger{}))
			if err != nil {
				t.Fatalf("failed to create cloudinfoer; [%s]", err.Error())
			}

			test.check(cloudInfoer.GetRegion(test.id))
		})
	}
}

func TestEc2Infoer_getCurrentSpotPrices(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		ec2CliMock func(region string) Ec2Describer
		check      func(data map[string]types.SpotPriceInfo, err error)
	}{
		{
			name:   "successful - get current spot prices",
			region: "dummyRegion",
			ec2CliMock: func(region string) Ec2Describer {
				return &testStruct{}
			},
			check: func(data map[string]types.SpotPriceInfo, err error) {
				assert.Equal(t, map[string]types.SpotPriceInfo{}, data)
				assert.Nil(t, err, "the error should be nil")
			},
		},
		{
			name:   "error - could not get spot price history pages",
			region: "dummyRegion",
			ec2CliMock: func(region string) Ec2Describer {
				return &testStruct{TcId: 11}
			},
			check: func(data map[string]types.SpotPriceInfo, err error) {
				assert.Nil(t, data, "the data should be nil")
				assert.EqualError(t, err, "invalid")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cloudInfoer, err := NewAmazonInfoer(Config{}, cloudinfoadapter.NewLogger(&logur.TestLogger{}))
			if err != nil {
				t.Fatalf("failed to create cloudinfoer; [%s]", err.Error())
			}
			// override ec2cli
			cloudInfoer.ec2Describer = test.ec2CliMock

			test.check(cloudInfoer.getCurrentSpotPrices(test.region))
		})
	}
}

func TestEc2Infoer_GetCurrentPrices(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		ec2CliMock func(region string) Ec2Describer
		check      func(price map[string]types.Price, err error)
	}{
		{
			name:   "success - known region",
			region: "eu-central-1",
			ec2CliMock: func(region string) Ec2Describer {
				return &testStruct{}
			},
			check: func(price map[string]types.Price, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, 0, len(price))
			},
		},
		{
			name:   "error - unknown region",
			region: "dummyRegion",
			ec2CliMock: func(region string) Ec2Describer {
				return &testStruct{TcId: 11}
			},
			check: func(price map[string]types.Price, err error) {
				assert.Nil(t, price, "the zones should be nil")
				assert.EqualError(t, err, "invalid")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := Config{PrometheusAddress: "PromAPIAddress"}
			cloudInfoer, err := NewAmazonInfoer(c, cloudinfoadapter.NewLogger(&logur.TestLogger{}))
			if err != nil {
				t.Fatalf("failed to create cloudinfoer; [%s]", err.Error())
			}
			// override ec2cli
			cloudInfoer.ec2Describer = test.ec2CliMock

			test.check(cloudInfoer.GetCurrentPrices(test.region))
		})
	}
}

func TestEc2Infoer_GetZones(t *testing.T) {
	tests := []struct {
		name       string
		region     string
		ec2CliMock func(region string) Ec2Describer
		check      func(zones []string, err error)
	}{
		{
			name:   "success - known region",
			region: "eu-central-1",
			ec2CliMock: func(region string) Ec2Describer {
				return &testStruct{}
			},
			check: func(zones []string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, []string{"eu-central-1a", "eu-central-1b"}, zones)
				assert.Equal(t, 2, len(zones))
			},
		},
		{
			name:   "error - unknown region",
			region: "dummyRegion",
			ec2CliMock: func(region string) Ec2Describer {
				return &testStruct{TcId: 10}
			},
			check: func(zones []string, err error) {
				assert.Nil(t, zones, "the zones should be nil")
				assert.EqualError(t, err, "could not get information about zones")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c := Config{PrometheusAddress: "PromAPIAddress"}
			cloudInfoer, err := NewAmazonInfoer(c, cloudinfoadapter.NewLogger(&logur.TestLogger{}))
			if err != nil {
				t.Fatalf("failed to create cloudinfoer; [%s]", err.Error())
			}
			// override ec2cli
			cloudInfoer.ec2Describer = test.ec2CliMock

			test.check(cloudInfoer.GetZones(test.region))
		})
	}
}

func TestPriceData_getDataForKey(t *testing.T) {
	var missingData = priceData{
		awsData: aws.JSONValue{
			"product": map[string]interface{}{
				"attributes": map[string]interface{}{}}}}
	var wrongCast = priceData{
		awsData: aws.JSONValue{
			"product": map[string]interface{}{
				"attributes": map[string]interface{}{
					"instanceType": 0,
					"vcpu":         1,
					types.Memory:   2,
					"gpu":          3,
				}},
		},
	}
	var data = priceData{
		awsData: aws.JSONValue{
			"product": map[string]interface{}{
				"attributes": map[string]interface{}{
					"instanceType": ec2.InstanceTypeT2Small,
					"vcpu":         "1",
					types.Memory:   "2",
					"gpu":          "5",
				}},
			"terms": map[string]interface{}{
				"OnDemand": map[string]interface{}{
					"randomNumber": map[string]interface{}{
						"priceDimensions": map[string]interface{}{
							"randomNumber": map[string]interface{}{
								"pricePerUnit": map[string]interface{}{
									"USD": "5",
								}}}}}},
		},
	}
	tests := []struct {
		name  string
		attr  string
		price priceData
		check func(s string, err error)
	}{
		{
			name:  "successful - get instance type",
			attr:  "instanceType",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, "t2.small", s)
			},
		},
		{
			name:  "cast problem - get instance type",
			attr:  "instanceType",
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get instanceType or could not cast instanceType to string")
			},
		},
		{
			name:  "missing data - get instance type",
			attr:  "instanceType",
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get instanceType or could not cast instanceType to string")
			},
		},
		{
			name:  "successful - get cpu",
			attr:  "vcpu",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, "1", s)
			},
		},
		{
			name:  "cast problem - get cpu",
			attr:  "vcpu",
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get vcpu or could not cast vcpu to string")
			},
		},
		{
			name:  "missing data - get cpu",
			attr:  "vcpu",
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get vcpu or could not cast vcpu to string")
			},
		},
		{
			name:  "successful - get memory",
			attr:  types.Memory,
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, "2", s)
			},
		},
		{
			name:  "cast problem - get memory",
			attr:  types.Memory,
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get memory or could not cast memory to string")
			},
		},
		{
			name:  "missing data - get memory",
			attr:  types.Memory,
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get memory or could not cast memory to string")
			},
		},
		{
			name:  "successful - get gpu",
			attr:  "gpu",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, "5", s)
			},
		},
		{
			name:  "cast problem - get gpu",
			attr:  "gpu",
			price: wrongCast,
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get gpu or could not cast gpu to string")
			},
		},
		{
			name:  "missing data - get gpu",
			attr:  "gpu",
			price: missingData,
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get gpu or could not cast gpu to string")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pricedata, _ := newPriceData(test.price.awsData)
			test.check(pricedata.getDataForKey(test.attr))
		})
	}
}

func TestPriceData_GetOnDemandPrice(t *testing.T) {
	var data = priceData{
		awsData: aws.JSONValue{
			"product": map[string]interface{}{
				"attributes": map[string]interface{}{
					"instanceType": ec2.InstanceTypeT2Small,
					"vcpu":         "1",
					types.Memory:   "2",
					"gpu":          "5",
				}},
			"terms": map[string]interface{}{
				"OnDemand": map[string]interface{}{
					"randomNumber": map[string]interface{}{
						"priceDimensions": map[string]interface{}{
							"randomNumber": map[string]interface{}{
								"pricePerUnit": map[string]interface{}{
									"USD": "5",
								}}}}}},
		},
	}
	tests := []struct {
		name  string
		price priceData
		check func(s string, err error)
	}{
		{
			name:  "successful",
			price: data,
			check: func(s string, err error) {
				assert.Nil(t, err, "the error should be nil")
				assert.Equal(t, "5", s)
			},
		},
		{
			name: "cast problem",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"randomNumber": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"randomNumber": map[string]interface{}{
										"pricePerUnit": map[string]interface{}{
											"USD": 5,
										}}}}}}}},
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get on demand price or could not cast on demand price to string")
			},
		},
		{
			name: "missing data",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"randomNumber": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"randomNumber": map[string]interface{}{
										"pricePerUnit": map[string]interface{}{}}}}}}}},
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get on demand price or could not cast on demand price to string")
			},
		},
		{
			name: "could not get pricePerUnit",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"randomNumber": map[string]interface{}{
								"priceDimensions": map[string]interface{}{
									"randomNumber": map[string]interface{}{}}}}}}},
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get map for key")
			},
		},
		{
			name: "could not get priceDimensions",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{
						"OnDemand": map[string]interface{}{
							"randomNumber": map[string]interface{}{}}}}},
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get map for key")
			},
		},
		{
			name: "could not get OnDemand",
			price: priceData{
				awsData: aws.JSONValue{
					"terms": map[string]interface{}{}}},
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get map for key")
			},
		},
		{
			name: "could not get terms",
			price: priceData{
				awsData: aws.JSONValue{}},
			check: func(s string, err error) {
				assert.Equal(t, "", s)
				assert.EqualError(t, err, "could not get map for key")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(test.price.getOnDemandPrice())
		})
	}
}

func TestPriceData_newPriceData(t *testing.T) {
	tests := []struct {
		name   string
		prData aws.JSONValue
		check  func(data *priceData, err error)
	}{
		{
			name:   "successful",
			prData: aws.JSONValue{"product": map[string]interface{}{"attributes": map[string]interface{}{"dummy": "dummyInterface"}}},
			check: func(data *priceData, err error) {
				assert.Equal(t, map[string]interface{}{"dummy": "dummyInterface"}, data.attrMap)
				assert.Nil(t, err)
			},
		},
		{
			name:   "could not get map for key attributes",
			prData: aws.JSONValue{"product": map[string]interface{}{}},
			check: func(data *priceData, err error) {
				assert.Nil(t, data, "the data should be nil")
				assert.EqualError(t, err, "could not get map for key")
			},
		},
		{
			name:   "could not get map for key product",
			prData: aws.JSONValue{},
			check: func(data *priceData, err error) {
				assert.Nil(t, data, "the data should be nil")
				assert.EqualError(t, err, "could not get map for key")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.check(newPriceData(test.prData))
		})
	}
}
