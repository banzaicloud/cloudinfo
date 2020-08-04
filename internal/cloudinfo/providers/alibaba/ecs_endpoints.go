// Copyright © 2020 Banzai Cloud
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

// List of endpoints for the ECS service.
// This is a very quick, very ugly fix for a missing region problem.
// See https://github.com/aliyun/alibaba-cloud-sdk-go/issues/440
// nolint: gochecknoglobals
var endpoints = map[string]string{
	// From the API
	"ap-northeast-1": "ecs.ap-northeast-1.aliyuncs.com",
	"ap-south-1":     "ecs.ap-south-1.aliyuncs.com",
	"ap-southeast-1": "ecs.aliyuncs.com",
	"ap-southeast-2": "ecs.ap-southeast-2.aliyuncs.com",
	"ap-southeast-3": "ecs.ap-southeast-3.aliyuncs.com",
	"ap-southeast-5": "ecs.ap-southeast-5.aliyuncs.com",
	"cn-beijing":     "ecs.aliyuncs.com",
	"cn-chengdu":     "ecs.cn-chengdu.aliyuncs.com",
	"cn-hangzhou":    "ecs.aliyuncs.com",
	"cn-heyuan":      "ecs.cn-heyuan.aliyuncs.com",
	"cn-hongkong":    "ecs.aliyuncs.com",
	"cn-huhehaote":   "ecs.cn-huhehaote.aliyuncs.com",
	"cn-qingdao":     "ecs.aliyuncs.com",
	"cn-shanghai":    "ecs.aliyuncs.com",
	"cn-shenzhen":    "ecs.aliyuncs.com",
	"cn-wulanchabu":  "ecs.cn-wulanchabu.aliyuncs.com",
	"cn-zhangjiakou": "ecs.cn-zhangjiakou.aliyuncs.com",
	"eu-central-1":   "ecs.eu-central-1.aliyuncs.com",
	"eu-west-1":      "ecs.eu-west-1.aliyuncs.com",
	"me-east-1":      "ecs.me-east-1.aliyuncs.com",
	"us-east-1":      "ecs.aliyuncs.com",
	"us-west-1":      "ecs.aliyuncs.com",

	// From https://github.com/aliyun/alibaba-cloud-sdk-go/blob/6bfd679d0979b7108c3212daa010e2c9dab830c5/services/ecs/endpoint.go
	"cn-shanghai-internal-test-1": "ecs-cn-hangzhou.aliyuncs.com",
	"cn-beijing-gov-1":            "ecs.aliyuncs.com",
	"cn-shenzhen-su18-b01":        "ecs-cn-hangzhou.aliyuncs.com",
	// "cn-beijing":                  "ecs-cn-hangzhou.aliyuncs.com",
	"cn-shanghai-inner":           "ecs.aliyuncs.com",
	"cn-shenzhen-st4-d01":         "ecs-cn-hangzhou.aliyuncs.com",
	"cn-haidian-cm12-c01":         "ecs-cn-hangzhou.aliyuncs.com",
	"cn-hangzhou-internal-prod-1": "ecs-cn-hangzhou.aliyuncs.com",
	"cn-north-2-gov-1":            "ecs.aliyuncs.com",
	"cn-yushanfang":               "ecs.aliyuncs.com",
	// "cn-qingdao":                  "ecs-cn-hangzhou.aliyuncs.com",
	"cn-hongkong-finance-pop": "ecs.aliyuncs.com",
	// "cn-shanghai":                 "ecs-cn-hangzhou.aliyuncs.com",
	"cn-shanghai-finance-1": "ecs-cn-hangzhou.aliyuncs.com",
	// "cn-hongkong":                 "ecs-cn-hangzhou.aliyuncs.com",
	"cn-beijing-finance-pop": "ecs.aliyuncs.com",
	"cn-wuhan":               "ecs.aliyuncs.com",
	// "us-west-1":                   "ecs-cn-hangzhou.aliyuncs.com",
	// "cn-shenzhen":                 "ecs-cn-hangzhou.aliyuncs.com",
	"cn-zhengzhou-nebula-1":       "ecs.cn-qingdao-nebula.aliyuncs.com",
	"rus-west-1-pop":              "ecs.ap-northeast-1.aliyuncs.com",
	"cn-shanghai-et15-b01":        "ecs-cn-hangzhou.aliyuncs.com",
	"cn-hangzhou-bj-b01":          "ecs-cn-hangzhou.aliyuncs.com",
	"cn-hangzhou-internal-test-1": "ecs-cn-hangzhou.aliyuncs.com",
	"eu-west-1-oxs":               "ecs.cn-shenzhen-cloudstone.aliyuncs.com",
	"cn-zhangbei-na61-b01":        "ecs-cn-hangzhou.aliyuncs.com",
	"cn-beijing-finance-1":        "ecs.aliyuncs.com",
	"cn-hangzhou-internal-test-3": "ecs-cn-hangzhou.aliyuncs.com",
	"cn-shenzhen-finance-1":       "ecs-cn-hangzhou.aliyuncs.com",
	"cn-hangzhou-internal-test-2": "ecs-cn-hangzhou.aliyuncs.com",
	"cn-hangzhou-test-306":        "ecs-cn-hangzhou.aliyuncs.com",
	"cn-shanghai-et2-b01":         "ecs-cn-hangzhou.aliyuncs.com",
	"cn-hangzhou-finance":         "ecs.aliyuncs.com",
	// "ap-southeast-1":              "ecs-cn-hangzhou.aliyuncs.com",
	"cn-beijing-nu16-b01": "ecs-cn-hangzhou.aliyuncs.com",
	"cn-edge-1":           "ecs.cn-qingdao-nebula.aliyuncs.com",
	// "us-east-1":                   "ecs-cn-hangzhou.aliyuncs.com",
	"cn-fujian":               "ecs-cn-hangzhou.aliyuncs.com",
	"ap-northeast-2-pop":      "ecs.ap-northeast-1.aliyuncs.com",
	"cn-shenzhen-inner":       "ecs.aliyuncs.com",
	"cn-zhangjiakou-na62-a01": "ecs.cn-zhangjiakou.aliyuncs.com",
	// "cn-hangzhou":                 "ecs-cn-hangzhou.aliyuncs.com",
}
