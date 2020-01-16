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

package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/banzaicloud/cloudinfo/.gen/cloudinfo-client"
)

const (
	// todo get these from configuration
	host      = "alpha.dev.banzaicloud.com"
	userAgent = "cloudinfo-client/testing"
	basePath  = "cloudinfo/api/v1"
)

func getCloudinfoClient() *cloudinfo.APIClient {

	return cloudinfo.NewAPIClient(&cloudinfo.Configuration{
		BasePath:  basePath,
		Host:      host,
		Scheme:    "https",
		UserAgent: userAgent,
	})
}

func TestImagesApi(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		service  string
		region   string
		options  cloudinfo.GetImagesOpts
		asserter func(*testing.T, []cloudinfo.Image, *http.Response, error)
	}{
		{
			name:     "images successfully retrieved",
			provider: "amazon",
			service:  "pke",
			region:   "eu-west-2",
			options:  cloudinfo.GetImagesOpts{},
			asserter: func(t *testing.T, images []cloudinfo.Image, response *http.Response, err error) {
				assert.Nil(t, err)
				assert.True(t, response.StatusCode == http.StatusOK)
				assert.NotNil(t, images)
			},
		},
	}

	apiClient := getCloudinfoClient()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, r, err := apiClient.ImagesApi.GetImages(context.Background(), test.provider, test.service, test.region, &test.options)
			test.asserter(t, payload, r, err)
		})
	}
}
