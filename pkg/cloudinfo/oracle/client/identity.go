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
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/pkg/errors"
)

// Identity is for managing Identity related calls of OCI
type Identity struct {
	oci    *OCI
	client *identity.IdentityClient
}

// NewIdentityClient creates a new Identity
func (oci *OCI) NewIdentityClient() (client *Identity, err error) {

	client = &Identity{}

	oClient, err := identity.NewIdentityClientWithConfigurationProvider(oci.config)
	if err != nil {
		return client, err
	}

	client.client = &oClient
	client.oci = oci

	return client, nil
}

// GetAvailabilityDomains gets all Availability Domains within the region
func (i *Identity) GetAvailabilityDomains() (domains []identity.AvailabilityDomain, err error) {

	r, err := i.client.ListAvailabilityDomains(context.Background(), identity.ListAvailabilityDomainsRequest{
		CompartmentId: i.oci.Tenancy.Id,
	})

	return r.Items, err
}

// GetTenancy gets an identity.Tenancy by id
func (i *Identity) GetTenancy(id string) (t identity.Tenancy, err error) {

	r, err := i.client.GetTenancy(context.Background(), identity.GetTenancyRequest{
		TenancyId: common.String(id),
	})

	return r.Tenancy, err
}

// IsRegionAvailable check whether the given region is available
func (i *Identity) IsRegionAvailable(name string) error {

	availableRegions, err := i.GetSubscribedRegionNames()
	if err != nil {
		return err
	}

	if availableRegions[name] == name {
		return nil
	}

	return errors.Wrap(errors.New(name), "region is not available")
}

// GetSubscribedRegionNames gives back an array of subscribed regions' names
func (i *Identity) GetSubscribedRegionNames() (regions map[string]string, err error) {

	response, err := i.client.ListRegionSubscriptions(context.Background(), identity.ListRegionSubscriptionsRequest{
		TenancyId: i.oci.Tenancy.Id,
	})

	regions = make(map[string]string)
	for _, item := range response.Items {
		regions[*item.RegionName] = *item.RegionName
	}

	return regions, err
}
