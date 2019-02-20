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

	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/oracle/client"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
)

// Infoer encapsulates the data and operations needed to access external resources
type Infoer struct {
	client         *client.OCI
	shapeSpecs     map[string]ShapeSpecs
	cloudInfoCache map[string]ITRACloudInfo
	log            logur.Logger
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
	"VM.Standard1.1":  {PartNumber: "B88317", Mem: 7, Cpus: 1, NtwPerf: "0.6 Gbps"},
	"VM.Standard2.1":  {PartNumber: "B88514", Mem: 15, Cpus: 1, NtwPerf: "1 Gbps"},
	"VM.Standard1.2":  {PartNumber: "B88317", Mem: 14, Cpus: 2, NtwPerf: "1.2 Gbps"},
	"VM.Standard2.2":  {PartNumber: "B88514", Mem: 30, Cpus: 2, NtwPerf: "2 Gbps"},
	"VM.Standard1.4":  {PartNumber: "B88317", Mem: 28, Cpus: 4, NtwPerf: "1.2 Gbps"},
	"VM.Standard2.4":  {PartNumber: "B88514", Mem: 60, Cpus: 4, NtwPerf: "4.1 Gbps"},
	"VM.Standard1.8":  {PartNumber: "B88317", Mem: 56, Cpus: 8, NtwPerf: "2.4 Gbps"},
	"VM.Standard2.8":  {PartNumber: "B88514", Mem: 120, Cpus: 8, NtwPerf: "8.2 Gbps"},
	"VM.Standard1.16": {PartNumber: "B88317", Mem: 112, Cpus: 16, NtwPerf: "4.8 Gbps"},
	"VM.Standard2.16": {PartNumber: "B88514", Mem: 240, Cpus: 16, NtwPerf: "16.4 Gbps"},
	"VM.Standard2.24": {PartNumber: "B88514", Mem: 320, Cpus: 24, NtwPerf: "24.6 Gbps"},
	"VM.DenseIO1.4":   {PartNumber: "B88316", Mem: 60, Cpus: 4, NtwPerf: "1.2 Gbps"},
	"VM.DenseIO1.8":   {PartNumber: "B88316", Mem: 60, Cpus: 8, NtwPerf: "2.4 Gbps"},
	"VM.DenseIO2.8":   {PartNumber: "B88516", Mem: 120, Cpus: 8, NtwPerf: "8.2 Gbps"},
	"VM.DenseIO1.16":  {PartNumber: "B88316", Mem: 120, Cpus: 16, NtwPerf: "4.8 Gbps"},
	"VM.DenseIO2.16":  {PartNumber: "B88516", Mem: 240, Cpus: 16, NtwPerf: "16.4 Gbps"},
	"VM.DenseIO2.24":  {PartNumber: "B88516", Mem: 320, Cpus: 24, NtwPerf: "24.6 Gbps"},
}

// newInfoer creates a new instance of the infoer
func newInfoer(configFileLocation string, log logur.Logger) (*Infoer, error) {

	oci, err := client.NewOCI(configFileLocation)
	if err != nil {
		return nil, err
	}

	return &Infoer{
		client:     oci,
		shapeSpecs: shapeSpecs,
		log:        log,
	}, nil
}

func NewOracleInfoer(cfg Config, log logur.Logger) (*Infoer, error) {
	return newInfoer(cfg.ConfigLocation, log)
}

// Initialize downloads and parses the SKU list of the Compute Engine service
func (i *Infoer) Initialize() (prices map[string]map[string]cloudinfo.Price, err error) {
	return nil, nil
}

// GetCurrentPrices retrieves all the spot prices in a region
func (i *Infoer) GetCurrentPrices(region string) (prices map[string]cloudinfo.Price, err error) {
	return nil, errors.New("oracle prices cannot be queried on the fly")
}

// GetProductPrices gets prices for available shapes from ITRA
func (i *Infoer) GetProductPrice(specs ShapeSpecs) (price float64, err error) {
	info, err := i.GetCloudInfoFromITRA(specs.PartNumber)
	if err != nil {
		return 0, err
	}

	return info.GetPrice("PAY_AS_YOU_GO") * specs.Cpus, nil

}

func (i *Infoer) GetVirtualMachines(region string) (products []cloudinfo.VmInfo, err error) {
	log := log.WithFields(i.log, map[string]interface{}{"region": region})

	err = i.client.ChangeRegion(region)
	if err != nil {
		return
	}

	shapes, err := i.client.GetSupportedShapesInARegion(region, "compute")
	if err != nil {
		return
	}

	zones, err := i.GetZones(region)
	if err != nil {
		return
	}

	products = make([]cloudinfo.VmInfo, 0)
	for _, shape := range shapes {
		s := i.shapeSpecs[shape]
		ntwMapper := newNetworkMapper()
		ntwPerfCat, err := ntwMapper.MapNetworkPerf(fmt.Sprint(s.NtwPerf))
		if err != nil {
			log.Debug(emperror.Wrap(err, "failed to get network performance category").Error(),
				map[string]interface{}{"instanceType": shape})
		}

		price, err := i.GetProductPrice(s)
		if err != nil {
			return nil, err
		}

		products = append(products, cloudinfo.VmInfo{
			Type:          shape,
			OnDemandPrice: price,
			NtwPerf:       s.NtwPerf,
			NtwPerfCat:    ntwPerfCat,
			Cpus:          s.Cpus,
			Mem:           s.Mem,
			Zones:         zones,
			Attributes:    cloudinfo.Attributes(fmt.Sprint(s.Cpus), fmt.Sprint(s.Mem), ntwPerfCat),
		})
	}

	return
}

// GetProducts retrieves the available virtual machines types in a region
func (i *Infoer) GetProducts(vms []cloudinfo.VmInfo, service, regionId string) (products []cloudinfo.VmInfo, err error) {
	log := log.WithFields(i.log, map[string]interface{}{"service": service, "region": regionId})

	err = i.client.ChangeRegion(regionId)
	if err != nil {
		return
	}

	shapes, err := i.client.GetSupportedShapesInARegion(regionId, service)
	if err != nil {
		return
	}

	zones, err := i.GetZones(regionId)
	if err != nil {
		return
	}

	products = make([]cloudinfo.VmInfo, 0)
	for _, shape := range shapes {
		s := i.shapeSpecs[shape]
		ntwMapper := newNetworkMapper()
		ntwPerfCat, err := ntwMapper.MapNetworkPerf(fmt.Sprint(s.NtwPerf))
		if err != nil {
			log.Debug(emperror.Wrap(err, "failed to get network performance category").Error(),
				map[string]interface{}{"instanceType": shape})
		}

		price, err := i.GetProductPrice(s)
		if err != nil {
			return nil, err
		}

		products = append(products, cloudinfo.VmInfo{
			Type:          shape,
			OnDemandPrice: price,
			NtwPerf:       s.NtwPerf,
			NtwPerfCat:    ntwPerfCat,
			Cpus:          s.Cpus,
			Mem:           s.Mem,
			Zones:         zones,
			Attributes:    cloudinfo.Attributes(fmt.Sprint(s.Cpus), fmt.Sprint(s.Mem), ntwPerfCat),
		})
	}

	return
}

// GetRegions returns a map with available regions
func (i *Infoer) GetRegions(service string) (regions map[string]string, err error) {
	log := log.WithFields(i.log, map[string]interface{}{"service": service})
	log.Debug("getting regions")

	c, err := i.client.NewIdentityClient()
	if err != nil {
		return
	}

	_regions, err := c.GetSubscribedRegionNames()
	if err != nil {
		return
	}

	regions = make(map[string]string)
	for _, region := range _regions {
		description := region
		if displayName, ok := regionNames[region]; ok {
			description = displayName
		}
		regions[region] = description
	}

	log.Debug("found regions", map[string]interface{}{"numberOfRegions": len(regions)})
	return
}

// GetZones returns the availability zones in a region
func (i *Infoer) GetZones(region string) (zones []string, err error) {
	log := log.WithFields(i.log, map[string]interface{}{"region": region})
	log.Debug("getting zones")

	err = i.client.ChangeRegion(region)
	if err != nil {
		return
	}

	c, err := i.client.NewIdentityClient()
	if err != nil {
		return
	}

	ads, err := c.GetAvailabilityDomains()
	if err != nil {
		return
	}

	for _, ad := range ads {
		zones = append(zones, *ad.Name)
	}

	log.Debug("found zones", map[string]interface{}{"numberOfZones": len(zones)})
	return
}

// HasShortLivedPriceInfo - Oracle doesn't have preemptible instances
func (i *Infoer) HasShortLivedPriceInfo() bool {
	return false
}

// GetServices returns the available services on the  given region
func (i *Infoer) GetServices() ([]cloudinfo.Service, error) {
	services := []cloudinfo.Service{
		cloudinfo.NewService("compute"),
		cloudinfo.NewService("oke")}
	return services, nil
}

// HasImages - Oracle support images
func (i *Infoer) HasImages() bool {
	return true
}

// GetServiceImages retrieves the images supported by the given service in the given region
func (i *Infoer) GetServiceImages(service, region string) (images []cloudinfo.Image, err error) {

	_images, err := i.client.GetSupportedImagesInARegion(service, region)
	if err != nil {
		return images, err
	}

	for _, image := range _images {
		images = append(images, cloudinfo.NewImage(image, "", false))
	}

	return images, nil
}

// GetServiceProducts retrieves the products supported by the given service in the given region
func (i *Infoer) GetServiceProducts(region, service string) ([]cloudinfo.ProductDetails, error) {
	return nil, errors.New("GetServiceProducts - not yet implemented")
}

// GetServiceAttributes retrieves the attribute values supported by the given service in the given region for the given attribute
func (i *Infoer) GetServiceAttributes(region, service, attribute string) (cloudinfo.AttrValues, error) {
	return nil, errors.New("GetServiceAttributes - not yet implemented")
}

// GetVersions retrieves the kubernetes versions supported by the given service in the given region
func (i *Infoer) GetVersions(service, region string) ([]cloudinfo.ZoneVersion, error) {
	switch service {
	case "oke":
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

		return []cloudinfo.ZoneVersion{cloudinfo.NewZoneVersion(region, options.KubernetesVersions.Get())}, nil
	default:
		return []cloudinfo.ZoneVersion{}, nil
	}
}
