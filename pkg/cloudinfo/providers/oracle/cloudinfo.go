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

package oracle

import (
	"fmt"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/oracle/oci-go-sdk/common"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/providers/oracle/client"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/types"
	"github.com/banzaicloud/cloudinfo/pkg/platform/log"
)

const svcOke = "oke"

// Infoer encapsulates the data and operations needed to access external resources
type Infoer struct {
	client         *client.OCI
	shapeSpecs     map[string]ShapeSpecs
	cloudInfoCache map[string]ITRACloudInfo
	log            cloudinfo.Logger
}

// ShapeSpecs representation the specs of a certain type of virtual machine
type ShapeSpecs struct {
	PartNumber string
	Cpus       float64 `json:"cpusPerVm"`
	Mem        float64 `json:"memPerVm"`
	NtwPerf    string  `json:"NtwPerf"`
}

var regionNames = map[string]string{
	"uk-london-1":    "EU (London)",
	"eu-frankfurt-1": "EU (Frankfurt)",
	"us-ashburn-1":   "US East (Ashburn)",
	"us-phoenix-1":   "US West (Phoenix)",
}

var shapeSpecs = map[string]ShapeSpecs{
	"VM.Standard.E2.1": {PartNumber: "B90425", Mem: 8, Cpus: 1, NtwPerf: "0.7 Gbps"},
	"VM.Standard.E2.2": {PartNumber: "B90425", Mem: 16, Cpus: 2, NtwPerf: "1.4 Gbps"},
	"VM.Standard1.1":   {PartNumber: "B88317", Mem: 7, Cpus: 1, NtwPerf: "0.6 Gbps"},
	"VM.Standard2.1":   {PartNumber: "B88514", Mem: 15, Cpus: 1, NtwPerf: "1 Gbps"},
	"VM.Standard1.2":   {PartNumber: "B88317", Mem: 14, Cpus: 2, NtwPerf: "1.2 Gbps"},
	"VM.Standard2.2":   {PartNumber: "B88514", Mem: 30, Cpus: 2, NtwPerf: "2 Gbps"},
	"VM.Standard1.4":   {PartNumber: "B88317", Mem: 28, Cpus: 4, NtwPerf: "1.2 Gbps"},
	"VM.Standard2.4":   {PartNumber: "B88514", Mem: 60, Cpus: 4, NtwPerf: "4.1 Gbps"},
	"VM.Standard1.8":   {PartNumber: "B88317", Mem: 56, Cpus: 8, NtwPerf: "2.4 Gbps"},
	"VM.Standard2.8":   {PartNumber: "B88514", Mem: 120, Cpus: 8, NtwPerf: "8.2 Gbps"},
	"VM.Standard1.16":  {PartNumber: "B88317", Mem: 112, Cpus: 16, NtwPerf: "4.8 Gbps"},
	"VM.Standard2.16":  {PartNumber: "B88514", Mem: 240, Cpus: 16, NtwPerf: "16.4 Gbps"},
	"VM.Standard2.24":  {PartNumber: "B88514", Mem: 320, Cpus: 24, NtwPerf: "24.6 Gbps"},
	"VM.DenseIO1.4":    {PartNumber: "B88316", Mem: 60, Cpus: 4, NtwPerf: "1.2 Gbps"},
	"VM.DenseIO1.8":    {PartNumber: "B88316", Mem: 60, Cpus: 8, NtwPerf: "2.4 Gbps"},
	"VM.DenseIO2.8":    {PartNumber: "B88516", Mem: 120, Cpus: 8, NtwPerf: "8.2 Gbps"},
	"VM.DenseIO1.16":   {PartNumber: "B88316", Mem: 120, Cpus: 16, NtwPerf: "4.8 Gbps"},
	"VM.DenseIO2.16":   {PartNumber: "B88516", Mem: 240, Cpus: 16, NtwPerf: "16.4 Gbps"},
	"VM.DenseIO2.24":   {PartNumber: "B88516", Mem: 320, Cpus: 24, NtwPerf: "24.6 Gbps"},
}

// NewOracleInfoer creates a new instance of the Oracle infoer.
func NewOracleInfoer(config Config, logger cloudinfo.Logger) (*Infoer, error) {
	var privateKeyPassphrase string
	if config.PrivateKeyPassphrase != nil {
		privateKeyPassphrase = *config.PrivateKeyPassphrase
	}

	providers := []common.ConfigurationProvider{
		common.NewRawConfigurationProvider(
			config.Tenancy,
			config.User,
			config.Region,
			config.Fingerprint,
			config.PrivateKey,
			config.PrivateKeyPassphrase,
		),
		common.ConfigurationProviderEnvironmentVariables("oracle", privateKeyPassphrase),
	}

	if config.ConfigFilePath != "" && config.Profile != "" {
		provider, _ := common.ConfigurationProviderFromFileWithProfile(
			config.ConfigFilePath,
			config.Profile,
			privateKeyPassphrase,
		)
		providers = append(providers, provider)
	} else if config.ConfigFilePath != "" {
		provider, _ := common.ConfigurationProviderFromFile(
			config.ConfigFilePath,
			privateKeyPassphrase,
		)
		providers = append(providers, provider)
	}

	provider, _ := common.ComposingConfigurationProvider(providers)

	oci, err := client.NewOCI(provider)
	if err != nil {
		return nil, err
	}

	return &Infoer{
		client:     oci,
		shapeSpecs: shapeSpecs,
		log:        logger,
	}, nil
}

// Initialize downloads and parses the SKU list of the Compute Engine service
func (i *Infoer) Initialize() (map[string]map[string]types.Price, error) {
	return nil, nil
}

// GetCurrentPrices retrieves all the spot prices in a region
func (i *Infoer) GetCurrentPrices(region string) (map[string]types.Price, error) {
	return nil, errors.New("oracle prices cannot be queried on the fly")
}

// GetProductPrices gets prices for available shapes from ITRA
func (i *Infoer) GetProductPrice(specs ShapeSpecs) (float64, error) {
	info, err := i.GetCloudInfoFromITRA(specs.PartNumber)
	if err != nil {
		return 0, err
	}

	return info.GetPrice("PAY_AS_YOU_GO") * specs.Cpus, nil

}

func (i *Infoer) GetVirtualMachines(region string) ([]types.VMInfo, error) {
	logger := log.WithFields(i.log, map[string]interface{}{"region": region})

	err := i.client.ChangeRegion(region)
	if err != nil {
		return nil, err
	}

	shapes, err := i.client.GetSupportedShapesInARegion(region, "compute")
	if err != nil {
		return nil, err
	}

	zones, err := i.GetZones(region)
	if err != nil {
		return nil, err
	}

	products := make([]types.VMInfo, 0, len(shapes))
	for _, shape := range shapes {
		s, ok := i.shapeSpecs[shape]
		if !ok {
			logger.Debug("unsupported shape", map[string]interface{}{"shape": shape})
			// skipping further processing
			continue
		}
		ntwMapper := newNetworkMapper()
		ntwPerfCat, err := ntwMapper.MapNetworkPerf(s.NtwPerf)
		if err != nil {
			logger.Debug(emperror.Wrap(err, "failed to get network performance category").Error(),
				map[string]interface{}{"instanceType": shape})
		}

		price, err := i.GetProductPrice(s)
		if err != nil {
			return nil, err
		}

		products = append(products, types.VMInfo{
			Category:      types.CategoryMemory,
			Type:          shape,
			OnDemandPrice: price,
			NtwPerf:       s.NtwPerf,
			NtwPerfCat:    ntwPerfCat,
			Cpus:          s.Cpus,
			Mem:           s.Mem,
			Zones:         zones,
			Attributes:    cloudinfo.Attributes(fmt.Sprint(s.Cpus), fmt.Sprint(s.Mem), ntwPerfCat, types.CategoryMemory),
		})
	}

	return products, nil
}

// GetProducts retrieves the available virtual machines types in a region
func (i *Infoer) GetProducts(vms []types.VMInfo, service, regionId string) ([]types.VMInfo, error) {
	logger := log.WithFields(i.log, map[string]interface{}{"service": service, "region": regionId})

	err := i.client.ChangeRegion(regionId)
	if err != nil {
		return nil, err
	}

	shapes, err := i.client.GetSupportedShapesInARegion(regionId, service)
	if err != nil {
		return nil, err
	}

	zones, err := i.GetZones(regionId)
	if err != nil {
		return nil, err
	}

	products := make([]types.VMInfo, 0, len(shapes))
	ntwMapper := newNetworkMapper()
	for _, shape := range shapes {
		s, ok := i.shapeSpecs[shape]
		if !ok {
			logger.Warn(fmt.Sprintf("unsupported shape: %s", shape))
			continue
		}

		ntwPerfCat, err := ntwMapper.MapNetworkPerf(s.NtwPerf)
		if err != nil {
			logger.Warn("failed to get network performance category", map[string]interface{}{"shape": shape})
		}

		price, err := i.GetProductPrice(s)
		if err != nil {
			logger.Warn("failed to get product price", map[string]interface{}{"shape": shape})
			continue
		}

		products = append(products, types.VMInfo{
			Category:      types.CategoryMemory,
			Type:          shape,
			OnDemandPrice: price,
			NtwPerf:       s.NtwPerf,
			NtwPerfCat:    ntwPerfCat,
			Cpus:          s.Cpus,
			Mem:           s.Mem,
			Zones:         zones,
			Attributes:    cloudinfo.Attributes(fmt.Sprint(s.Cpus), fmt.Sprint(s.Mem), ntwPerfCat, types.CategoryMemory),
		})
	}

	return products, nil
}

// GetRegions returns a map with available regions
func (i *Infoer) GetRegions(service string) (map[string]string, error) {
	logger := log.WithFields(i.log, map[string]interface{}{"service": service})
	logger.Debug("getting regions")

	c, err := i.client.NewIdentityClient()
	if err != nil {
		return nil, err
	}

	subscribedRegionNames, err := c.GetSubscribedRegionNames()
	if err != nil {
		return nil, err
	}

	regions := make(map[string]string)
	for _, region := range subscribedRegionNames {
		description := region
		if displayName, ok := regionNames[region]; ok {
			description = displayName
		}
		regions[region] = description
	}

	logger.Debug("found regions", map[string]interface{}{"numberOfRegions": len(regions)})
	return regions, nil
}

// GetZones returns the availability zones in a region
func (i *Infoer) GetZones(region string) ([]string, error) {
	err := i.client.ChangeRegion(region)
	if err != nil {
		return nil, err
	}

	c, err := i.client.NewIdentityClient()
	if err != nil {
		return nil, err
	}

	availabilityDomains, err := c.GetAvailabilityDomains()
	if err != nil {
		return nil, err
	}

	zones := make([]string, 0, len(availabilityDomains))
	for _, ad := range availabilityDomains {
		zones = append(zones, *ad.Name)
	}

	return zones, nil
}

// HasShortLivedPriceInfo - Oracle doesn't have preemptible instances
func (i *Infoer) HasShortLivedPriceInfo() bool {
	return false
}

// HasImages - Oracle support images
func (i *Infoer) HasImages() bool {
	return true
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (i *Infoer) GetServiceImages(service, region string) ([]types.Image, error) {
	imageNames, err := i.client.GetSupportedImagesInARegion(service, region)
	if err != nil {
		return nil, err
	}

	var images = make([]types.Image, 0, len(imageNames))
	for _, imageName := range imageNames {
		images = append(images, types.NewImage(imageName, "", false))
	}

	return images, nil
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (i *Infoer) GetServiceProducts(region, service string) ([]types.ProductDetails, error) {
	return nil, errors.New("GetServiceProducts - not yet implemented")
}

// GetVersions retrieves the kubernetes versions supported by the given service in the given region
func (i *Infoer) GetVersions(service, region string) ([]types.LocationVersion, error) {
	switch service {
	case svcOke:
		err := i.client.ChangeRegion(region)
		if err != nil {
			return nil, err
		}

		ce, err := i.client.NewContainerEngineClient()
		if err != nil {
			return nil, err
		}

		options, err := ce.GetDefaultNodePoolOptions()
		if err != nil {
			return nil, err
		}

		versions := options.KubernetesVersions.Get()

		return []types.LocationVersion{types.NewLocationVersion(region, versions, "")}, nil
	default:
		return []types.LocationVersion{}, nil
	}
}
