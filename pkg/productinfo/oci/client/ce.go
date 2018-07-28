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
