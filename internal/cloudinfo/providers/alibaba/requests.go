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

package alibaba

import (
	"strconv"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	cloudinfo2 "github.com/banzaicloud/cloudinfo/internal/cloudinfo"
)

// CommonDescriber interface for all Alibaba API calls
type CommonDescriber interface {
	ProcessCommonRequest(request *requests.CommonRequest) (*responses.CommonResponse, error)
}

func (a *AlibabaInfoer) describeSpotPriceHistoryRequest(region, instanceType string) *requests.CommonRequest {
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.ApiName = "DescribeSpotPriceHistory"
	request.Domain = "ecs.aliyuncs.com"
	request.Version = "2014-05-26"
	request.QueryParams["RegionId"] = region
	request.QueryParams["NetworkType"] = "vpc"
	request.QueryParams["OSType"] = "linux"
	request.QueryParams["StartTime"] = time.Now().Round(1 * time.Hour).UTC().Format(time.RFC3339)
	request.QueryParams["InstanceType"] = instanceType

	return request
}

func (a *AlibabaInfoer) describeInstanceTypesRequest() *requests.CommonRequest {
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.ApiName = "DescribeInstanceTypes"
	request.Domain = "ecs.aliyuncs.com"
	request.Version = "2014-05-26"

	return request
}

func (a *AlibabaInfoer) describeZonesRequest(region string) *requests.CommonRequest {
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.ApiName = "DescribeZones"
	request.Domain = "ecs.aliyuncs.com"
	request.Version = "2014-05-26"
	request.QueryParams["RegionId"] = region

	return request
}

func (a *AlibabaInfoer) getPayAsYouGoPriceRequest(region string, instanceTypes []string) *requests.CommonRequest {
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.ApiName = "GetPayAsYouGoPrice"
	request.Scheme = "https"
	request.Domain = "business.ap-southeast-1.aliyuncs.com"
	request.Version = "2017-12-14"
	request.QueryParams["RegionId"] = region
	request.QueryParams["ProductCode"] = "ecs"
	request.QueryParams["SubscriptionType"] = "PayAsYouGo"
	for i, instanceType := range instanceTypes {
		request.QueryParams[cloudinfo2.CreateString("ModuleList.", strconv.Itoa(i+1), ".ModuleCode")] = "InstanceType"
		request.QueryParams[cloudinfo2.CreateString("ModuleList.", strconv.Itoa(i+1), ".Config")] =
			cloudinfo2.CreateString("InstanceType:", instanceType, ",IoOptimized:IoOptimized,ImageOs:linux")
		request.QueryParams[cloudinfo2.CreateString("ModuleList.", strconv.Itoa(i+1), ".PriceType")] = "Hour"
	}

	return request
}

func (a *AlibabaInfoer) describeRegionsRequest() *requests.CommonRequest {
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.ApiName = "DescribeRegions"
	request.Domain = "ecs.aliyuncs.com"
	request.Version = "2014-05-26"
	request.QueryParams["AcceptLanguage"] = "en-US"

	return request
}

func (a *AlibabaInfoer) describeImagesRequest(region string) *requests.CommonRequest {
	request := requests.NewCommonRequest()
	request.Method = "POST"
	request.ApiName = "DescribeImages"
	request.Domain = "ecs.aliyuncs.com"
	request.Version = "2014-05-26"
	request.QueryParams["OSType"] = "linux"
	request.QueryParams["RegionId"] = region

	return request
}
