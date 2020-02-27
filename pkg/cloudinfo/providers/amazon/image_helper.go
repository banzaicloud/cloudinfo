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

package amazon

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	eksGPUImageNamePrefix = "amazon-eks-gpu"
	eksImageNamePrefix    = "amazon-eks"
	eksImageNameFormat    = "%s-node-%s-*"

	tagPKEVersion = "pke-version"
	tagK8SVersion = "k8s-version"
)

func getEKSDescribeImagesInput(kubernetesVersion string, GPUs bool) *ec2.DescribeImagesInput {
	nameFilter := fmt.Sprintf(eksImageNameFormat, eksImageNamePrefix, kubernetesVersion)
	if GPUs {
		nameFilter = fmt.Sprintf(eksImageNameFormat, eksGPUImageNamePrefix, kubernetesVersion)
	}

	describeImagesInput := ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []*string{aws.String(nameFilter)},
			},
			{
				Name:   aws.String("is-public"),
				Values: []*string{aws.String("true")},
			},
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
			{
				Name:   aws.String("owner-alias"),
				Values: []*string{aws.String("amazon")},
			},
		},
	}
	return &describeImagesInput
}

func getPKEDescribeImagesInput() *ec2.DescribeImagesInput {
	describeImagesInput := ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", tagPKEVersion)),
				Values: []*string{aws.String("*")},
			},
		},
	}
	return &describeImagesInput
}

func tagsFormImage(image *ec2.Image) map[string]string {
	tags := make(map[string]string)
	for _, tagPtr := range image.Tags {
		tags[*tagPtr.Key] = *tagPtr.Value
	}
	return tags
}
