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

	"github.com/oracle/oci-go-sdk/containerengine"
)

// ContainerEngine is for managing OKE related calls of OCI
type ContainerEngine struct {
	CompartmentOCID string

	oci    *OCI
	client *containerengine.ContainerEngineClient
}

// NewContainerEngineClient creates a new ContainerEngine
func (oci *OCI) NewContainerEngineClient() (client *ContainerEngine, err error) {

	client = &ContainerEngine{}

	oClient, err := containerengine.NewContainerEngineClientWithConfigurationProvider(oci.config)
	if err != nil {
		return client, err
	}

	client.client = &oClient
	client.oci = oci
	client.CompartmentOCID = *oci.Tenancy.Id

	return client, nil
}

// GetDefaultNodePoolOptions gets default node pool options
func (ce *ContainerEngine) GetDefaultNodePoolOptions() (options NodePoolOptions, err error) {

	return ce.GetNodePoolOptions("all")
}

// GetNodePoolOptions gets available node pool options for a specified cluster OCID
func (ce *ContainerEngine) GetNodePoolOptions(clusterID string) (options NodePoolOptions, err error) {

	request := containerengine.GetNodePoolOptionsRequest{
		NodePoolOptionId: &clusterID,
	}

	r, err := ce.client.GetNodePoolOptions(context.Background(), request)

	return NodePoolOptions{
		Images:             Strings{strings: r.Images},
		KubernetesVersions: Strings{strings: r.KubernetesVersions},
		Shapes:             Strings{strings: r.Shapes},
	}, err
}
