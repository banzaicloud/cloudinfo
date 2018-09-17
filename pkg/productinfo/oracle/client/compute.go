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

package client

import (
	"context"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/core"
)

// Compute is for managing Compute related calls of OCI
type Compute struct {
	oci    *OCI
	client *core.ComputeClient
}

// NewComputeClient creates a new Compute
func (oci *OCI) NewComputeClient() (client *Compute, err error) {

	client = &Compute{}

	oClient, err := core.NewComputeClientWithConfigurationProvider(oci.config)
	if err != nil {
		return client, err
	}

	client.client = &oClient
	client.oci = oci

	return client, nil
}

// GetShapes gets all available Shapes within the Tenancy
func (c *Compute) GetShapes() (shapes []core.Shape, err error) {

	request := core.ListShapesRequest{
		CompartmentId: c.oci.Tenancy.Id,
	}
	request.Limit = common.Int(20)

	listFunc := func(request core.ListShapesRequest) (core.ListShapesResponse, error) {
		return c.client.ListShapes(context.Background(), request)
	}

	for response, err := listFunc(request); ; response, err = listFunc(request) {
		if err != nil {
			return shapes, err
		}

		for _, item := range response.Items {
			shapes = append(shapes, item)
		}

		if response.OpcNextPage != nil {
			// if there are more items in next page, fetch items from next page
			request.Page = response.OpcNextPage
		} else {
			// no more result, break the loop
			break
		}
	}

	return shapes, err
}
