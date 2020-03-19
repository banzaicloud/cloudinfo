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

package digitalocean

import (
	"context"
	"fmt"
	"strings"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/types"
	"github.com/banzaicloud/cloudinfo/pkg/platform/log"
)

// DigitaloceanInfoer encapsulates the data and operations needed to access external DigitalOcean resources.
type DigitaloceanInfoer struct {
	client *godo.Client

	logger cloudinfo.Logger
}

// NewDigitaloceanInfoer creates a new instance of the Digitalocean infoer.
func NewDigitaloceanInfoer(config Config, logger cloudinfo.Logger) (*DigitaloceanInfoer, error) {
	tokenSource := oauth2.StaticTokenSource(&oauth2.Token{
		AccessToken: config.AccessToken,
	})
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := godo.NewClient(oauthClient)

	return &DigitaloceanInfoer{
		client: client,
		logger: logger,
	}, nil
}

func (i *DigitaloceanInfoer) getSizes() ([]godo.Size, error) {
	var sizeList []godo.Size

	opt := &godo.ListOptions{}
	for {
		sizes, resp, err := i.client.Sizes.List(context.Background(), opt)
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list droplet sizes")
		}

		for _, s := range sizes {
			sizeList = append(sizeList, s)
		}

		if resp.Links == nil || resp.Links.IsLastPage() {
			break
		}

		page, err := resp.Links.CurrentPage()
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list droplet sizes")
		}

		opt.Page = page + 1
	}

	return sizeList, nil
}

func (i *DigitaloceanInfoer) Initialize() (map[string]map[string]types.Price, error) {
	i.logger.Debug("initializing price info")
	allPrices := make(map[string]map[string]types.Price)

	sizes, err := i.getSizes()
	if err != nil {
		return nil, err
	}

	for _, size := range sizes {
		if !size.Available {
			continue
		}

		for _, r := range size.Regions {
			if allPrices[r] == nil {
				allPrices[r] = make(map[string]types.Price)
			}

			allPrices[r][size.Slug] = types.Price{
				OnDemandPrice: size.PriceHourly,
			}
		}
	}

	i.logger.Debug("finished initializing price info")
	return allPrices, nil
}

func contains(s string, ss []string) bool {
	for _, v := range ss {
		if s == v {
			return true
		}
	}

	return false
}

func getCategory(name string) string {
	switch {
	case strings.HasPrefix(name, "s-"):
		// TODO: Standard and general are not the same at DO
		return types.CategoryGeneral
	case strings.HasPrefix(name, "c-"):
		return types.CategoryCompute
	default:
		return types.CategoryGeneral
	}
}

func (i *DigitaloceanInfoer) GetVirtualMachines(region string) ([]types.VMInfo, error) {
	logger := log.WithFields(i.logger, map[string]interface{}{"region": region})
	logger.Debug("getting product info")

	sizes, err := i.getSizes()
	if err != nil {
		return nil, err
	}

	var virtualMachines []types.VMInfo

	for _, size := range sizes {
		if !size.Available || !contains(region, size.Regions) {
			continue
		}

		virtualMachines = append(virtualMachines, types.VMInfo{
			Category:      getCategory(size.Slug),
			Type:          size.Slug,
			OnDemandPrice: size.PriceHourly,
			Mem:           float64(size.Memory) / 1024,
			Cpus:          float64(size.Vcpus),
			NtwPerf:       "300 Mbit/s",
			NtwPerfCat:    types.NtwLow,
			Zones:         []string{},
			Attributes:    cloudinfo.Attributes(fmt.Sprint(size.Vcpus), fmt.Sprint(size.Memory), types.NtwLow, getCategory(size.Slug)),
		})
	}

	return virtualMachines, nil
}

func (i *DigitaloceanInfoer) GetProducts(vms []types.VMInfo, service, regionId string) ([]types.VMInfo, error) {
	switch service {
	case "dok":
		options, _, err := i.client.Kubernetes.GetOptions(context.Background())
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list regions")
		}

		sizes := make([]string, len(options.Sizes))
		for i, size := range options.Sizes {
			sizes[i] = size.Slug
		}

		var virtualMachines []types.VMInfo

		for _, vm := range vms {
			if !contains(vm.Type, sizes) {
				continue
			}

			virtualMachines = append(virtualMachines, vm)
		}

		return virtualMachines, nil

	default:
		return nil, emperror.With(errors.New("unknown service"), "service", service)
	}
}

func (*DigitaloceanInfoer) GetZones(region string) ([]string, error) {
	return []string{}, nil
}

func (i *DigitaloceanInfoer) GetRegions(service string) (map[string]string, error) {
	switch service {
	case "compute":
		regions, _, err := i.client.Regions.List(context.Background(), &godo.ListOptions{Page: 1, PerPage: 200})
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list regions")
		}

		regionMap := make(map[string]string, len(regions))

		for _, region := range regions {
			if !region.Available {
				continue
			}

			regionMap[region.Slug] = region.Name
		}

		return regionMap, nil

	case "dok":
		options, _, err := i.client.Kubernetes.GetOptions(context.Background())
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list regions")
		}

		regionMap := make(map[string]string, len(options.Regions))

		for _, region := range options.Regions {
			regionMap[region.Slug] = region.Name
		}

		return regionMap, nil

	default:
		return nil, emperror.With(errors.New("unknown service"), "service", service)
	}
}

func (*DigitaloceanInfoer) HasShortLivedPriceInfo() bool {
	return false
}

func (*DigitaloceanInfoer) GetCurrentPrices(region string) (map[string]types.Price, error) {
	return nil, errors.New("GetCurrentPrices - not yet implemented")
}

func (*DigitaloceanInfoer) HasImages() bool {
	return false
}

func (*DigitaloceanInfoer) GetServiceImages(service, region string) ([]types.Image, error) {
	return nil, errors.New("GetServiceImages - not yet implemented")
}

func (i *DigitaloceanInfoer) GetVersions(service, region string) ([]types.LocationVersion, error) {
	switch service {
	case "dok":
		options, _, err := i.client.Kubernetes.GetOptions(context.Background())
		if err != nil {
			return nil, emperror.Wrap(err, "failed to list regions")
		}

		var defaultVersion string
		if len(options.Versions) > 0 {
			defaultVersion = options.Versions[0].Slug
		}

		versions := make([]string, len(options.Versions))

		for i, version := range options.Versions {
			versions[i] = version.Slug
		}

		return []types.LocationVersion{types.NewLocationVersion(region, versions, defaultVersion)}, nil
	default:
		return []types.LocationVersion{}, nil
	}
}

func (*DigitaloceanInfoer) GetServiceProducts(region, service string) ([]types.ProductDetails, error) {
	return nil, errors.New("GetServiceProducts - not yet implemented")
}
