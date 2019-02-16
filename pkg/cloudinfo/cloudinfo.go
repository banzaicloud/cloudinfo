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

package cloudinfo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/tracing"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/goph/emperror"
)

// cachingCloudInfo is the module struct, holds configuration and cache
// It's the entry point for the product info retrieval and management subsystem
// It's also responsible for delegating to the cloud provider specific implementations
type cachingCloudInfo struct {
	cloudInfoers   map[string]CloudInfoer
	cloudInfoStore CloudInfoStore
	metrics        metrics.Reporter
	tracer         tracing.Tracer
}

func (cpi *cachingCloudInfo) RefreshProvider(ctx context.Context, provider string) {
	if _, ok := cpi.cloudInfoers[provider]; !ok {
		logger.Extract(ctx).Error("refresh - unsupported provider", map[string]interface{}{"provider": provider, "op": "refreshProvider"})
	}
	cpi.renewProviderInfo(ctx, provider)
}

func (v AttrValues) floatValues() []float64 {
	floatValues := make([]float64, len(v))
	for i, av := range v {
		floatValues[i] = av.Value
	}
	return floatValues
}

// SpotPriceInfo represents different prices per availability zones
type SpotPriceInfo map[string]float64

// Price describes the on demand price and spot prices per availability zones
type Price struct {
	OnDemandPrice float64       `json:"onDemandPrice"`
	SpotPrice     SpotPriceInfo `json:"spotPrice"`
}

// VmInfo representation of a virtual machine
type VmInfo struct {
	Type          string            `json:"type"`
	OnDemandPrice float64           `json:"onDemandPrice"`
	SpotPrice     SpotPriceInfo     `json:"spotPrice"`
	Cpus          float64           `json:"cpusPerVm"`
	Mem           float64           `json:"memPerVm"`
	Gpus          float64           `json:"gpusPerVm"`
	NtwPerf       string            `json:"ntwPerf"`
	NtwPerfCat    string            `json:"ntwPerfCategory"`
	Zones         []string          `json:"zones"`
	Attributes    map[string]string `json:"attributes"`
	// CurrentGen signals whether the instance type generation is the current one. Only applies for amazon
	CurrentGen bool `json:"currentGen"`
}

// IsBurst returns true if the EC2 instance vCPU is burst type
// the decision is made based on the instance type
func (vm VmInfo) IsBurst() bool {
	return strings.HasPrefix(strings.ToUpper(vm.Type), "T")
}

// NewCachingCloudInfo creates a new cachingCloudInfo instance
func NewCachingCloudInfo(ciStore CloudInfoStore, infoers map[string]CloudInfoer, reporter metrics.Reporter, tracer tracing.Tracer) (*cachingCloudInfo, error) {
	if infoers == nil || ciStore == nil {
		return nil, errors.New("could not create product infoer")
	}

	pi := cachingCloudInfo{
		cloudInfoers:   infoers,
		cloudInfoStore: ciStore,
		metrics:        reporter,
		tracer:         tracer,
	}
	return &pi, nil
}

// GetProviders returns the supported providers
func (cpi *cachingCloudInfo) GetProviders(ctx context.Context) []Provider {
	var (
		providers []Provider
		provider  Provider
		err       error
	)
	log := logger.Extract(ctx)

	// iterate over supported provider names only
	for pn := range cpi.cloudInfoers {
		if provider, err = cpi.GetProvider(ctx, pn); err != nil {
			log.Warn("could not retrieve provider", map[string]interface{}{"provider": provider})
		}

		providers = append(providers, provider)
	}

	return providers
}

// GetProvider returns the supported provider
func (cpi *cachingCloudInfo) GetProvider(ctx context.Context, provider string) (Provider, error) {
	var (
		srvcs []Service
		err   error
	)

	if _, ok := cpi.cloudInfoers[provider]; !ok {
		return Provider{}, emperror.With(errors.New("unsupported provider"), "provider", provider)
	}

	if srvcs, err = cpi.GetServices(ctx, provider); err != nil {
		return Provider{}, emperror.With(errors.New("no supported services for provider"), "provider", provider)
	}

	// decorate the provider with service information
	p := NewProvider(provider)
	p.Services = srvcs

	return p, nil
}

// renewProviderInfo renews provider information for the provider argument. It optionally signals the end of renewal to the
// provided WaitGroup (if provided)
func (cpi *cachingCloudInfo) renewProviderInfo(ctx context.Context, provider string) {
	ctx, _ = cpi.tracer.StartWithTags(ctx, fmt.Sprintf("renew-provider (%s)", provider), map[string]interface{}{"provider": provider})
	defer cpi.tracer.EndSpan(ctx)

	log := logger.Extract(ctx)

	start := time.Now()

	log.Info("renewing product info")
	if _, err := cpi.Initialize(ctx, provider); err != nil {
		cpi.metrics.ReportScrapeFailure(provider, "N/A", "N/A")
		log.Error("failed to renew product info")
		return
	}

	services, err := cpi.cloudInfoers[provider].GetServices()
	if err != nil {
		cpi.metrics.ReportScrapeFailure(provider, "N/A", "N/A")
		log.Error(emperror.Wrap(err, "failed to renew products").Error())
		return
	}

	log.Info("start to renew attribute values")
	for _, service := range services {
		ctxLog := logger.ToContext(ctx,
			logger.NewLogCtxBuilder().
				WithService(service.ServiceName()).
				Build())
		attributes := []string{Cpu, Memory}
		for _, attr := range attributes {
			_, err := cpi.renewAttrValues(ctxLog, provider, service.ServiceName(), attr)
			if err != nil {
				cpi.metrics.ReportScrapeFailure(provider, "N/A", "N/A")
				logger.Extract(ctxLog).Error(emperror.Wrap(err, "failed to renew attribute values").Error(), map[string]interface{}{"attr": attr})
				return
			}
		}
	}
	log.Info("finished to renew attribute values")

	log.Info("start to renew products (vm-s)")

	// todo spans to be created in individual method calls instead
	vmCtx, _ := cpi.tracer.StartWithTags(ctx, "renew-products", map[string]interface{}{"provider": provider})

	for _, service := range services {
		ctxLog := logger.ToContext(ctx,
			logger.NewLogCtxBuilder().
				WithService(service.ServiceName()).
				Build())
		regions, err := cpi.cloudInfoers[provider].GetRegions(service.ServiceName())
		if err != nil {
			cpi.metrics.ReportScrapeFailure(provider, service.ServiceName(), "N/A")
			logger.Extract(ctxLog).Error(emperror.Wrap(err, "failed to renew products").Error())
			return
		}

		for regionId := range regions {
			c := logger.ToContext(ctxLog,
				logger.NewLogCtxBuilder().
					WithRegion(regionId).
					Build())

			start := time.Now()
			_, err := cpi.renewVms(c, provider, service.ServiceName(), regionId)
			if err != nil {
				cpi.metrics.ReportScrapeFailure(provider, service.ServiceName(), regionId)
				logger.Extract(c).Error(emperror.Wrap(err, "failed to renew products").Error())
			}
			if cpi.cloudInfoers[provider].HasImages() {
				_, imgErr := cpi.renewImages(c, provider, service.ServiceName(), regionId)
				if imgErr != nil {
					cpi.metrics.ReportScrapeFailure(provider, service.ServiceName(), regionId)
					logger.Extract(c).Error("failed to renew images")
				}
			}
			_, versionErr := cpi.renewVersions(c, provider, service.ServiceName(), regionId)
			if versionErr != nil {
				cpi.metrics.ReportScrapeFailure(provider, service.ServiceName(), regionId)
				logger.Extract(c).Error("failed to renew versions")
			}
			if err == nil && versionErr == nil {
				cpi.metrics.ReportScrapeRegionCompleted(provider, service.ServiceName(), regionId, start)
			}
		}
	}
	log.Info("finished to renew products (vm-s)")

	// close the span
	cpi.tracer.EndSpan(vmCtx)

	if _, err := cpi.renewStatus(ctx, provider); err != nil {
		log.Error("failed to renew status")
		return
	}

	cpi.metrics.ReportScrapeProviderCompleted(provider, start)
}

func (cpi *cachingCloudInfo) renewStatus(ctx context.Context, provider string) (string, error) {
	ctx, _ = cpi.tracer.StartWithTags(ctx, "renew-status", map[string]interface{}{"provider": provider})
	defer cpi.tracer.EndSpan(ctx)

	values := strconv.Itoa(int(time.Now().UnixNano() / 1e6))

	cpi.cloudInfoStore.StoreStatus(provider, values)
	return values, nil
}

// Initialize stores the result of the Infoer's Initialize output in cache
func (cpi *cachingCloudInfo) Initialize(ctx context.Context, provider string) (map[string]map[string]Price, error) {
	ctx, _ = cpi.tracer.StartWithTags(ctx, "initialize", map[string]interface{}{"provider": provider})
	defer cpi.tracer.EndSpan(ctx)
	log := logger.Extract(ctx)
	log.Info("initializing cloud product information")
	allPrices, err := cpi.cloudInfoers[provider].Initialize()
	if err != nil {
		log.Warn("failed to initialize cloud product information")
		return nil, err
	}

	for region, ap := range allPrices {
		for instType, p := range ap {
			cpi.cloudInfoStore.StorePrice(provider, region, instType, p)
			metrics.OnDemandPriceGauge.WithLabelValues(provider, region, instType).Set(p.OnDemandPrice)
		}
	}
	log.Info("finished initializing cloud product information")
	return allPrices, nil
}

// GetAttributes returns the supported attribute names
func (cpi *cachingCloudInfo) GetAttributes(ctx context.Context) []string {
	return []string{Cpu, Memory}
}

// GetAttrValues returns a slice with the values for the given attribute name
func (cpi *cachingCloudInfo) GetAttrValues(ctx context.Context, provider, service, attribute string) ([]float64, error) {
	var (
		err    error
		values AttrValues
	)
	// check the cache
	if cachedVal, ok := cpi.cloudInfoStore.GetAttribute(provider, service, attribute); ok {
		logger.Extract(ctx).Debug("returning attribute values from cache")
		return cachedVal.(AttrValues).floatValues(), nil
	}

	// scrape provider for attribute values
	if values, err = cpi.renewAttrValues(ctx, provider, service, attribute); err == nil {
		logger.Extract(ctx).Debug("returning freshly scraped attribute values")
		return values.floatValues(), nil
	}

	return nil, emperror.Wrap(err, "failed to retrieve attribute values")

}

// renewAttrValues retrieves attribute values from the cloud provider and refreshes the attribute store with them
func (cpi *cachingCloudInfo) renewAttrValues(ctx context.Context, provider, service, attribute string) (AttrValues, error) {
	var (
		attr   string
		err    error
		values AttrValues
	)

	ctx, _ = cpi.tracer.StartWithTags(ctx, "renew-attribute-values", map[string]interface{}{"provider": provider, "service": service, "attribute": attribute})
	defer cpi.tracer.EndSpan(ctx)

	if attr, err = cpi.toProviderAttribute(provider, attribute); err != nil {
		return nil, emperror.With(err, "renewal")
	}

	if values, err = cpi.cloudInfoers[provider].GetAttributeValues(service, attr); err != nil {
		return nil, emperror.With(err, "renewal")
	}

	cpi.cloudInfoStore.StoreAttribute(provider, service, attribute, values)
	return values, nil
}

// GetPrice returns the on demand price and zone averaged computed spot price for a given instance type in a given region
func (cpi *cachingCloudInfo) GetPrice(ctx context.Context, provider string, region string, instanceType string, zones []string) (float64, float64, error) {
	var p Price
	ctx = logger.ToContext(ctx, logger.NewLogCtxBuilder().WithProvider(provider).WithRegion(region).Build())

	if cachedVal, ok := cpi.cloudInfoStore.GetPrice(provider, region, instanceType); !ok {
		var (
			allPriceInfo map[string]Price
			err          error
		)
		if allPriceInfo, err = cpi.renewShortLivedInfo(ctx, provider, region); err != nil {
			return 0, 0, emperror.Wrap(err, "failed to renew short lived info")
		}
		if allPriceInfo == nil {
			return 0, 0, errors.New("no prices found or short lived info disabled")
		}
		p = allPriceInfo[instanceType]
	} else {
		p = cachedVal.(Price)
	}

	var sumPrice float64
	for _, z := range zones {
		for zone, price := range p.SpotPrice {
			if zone == z {
				sumPrice += price
			}
		}
	}
	return p.OnDemandPrice, sumPrice / float64(len(zones)), nil
}

// renewAttrValues retrieves attribute values from the cloud provider and refreshes the attribute store with them
func (cpi *cachingCloudInfo) renewShortLivedInfo(ctx context.Context, provider string, region string) (map[string]Price, error) {

	var (
		err    error
		prices map[string]Price
	)

	if !cpi.cloudInfoers[provider].HasShortLivedPriceInfo() {
		return nil, nil
	}

	if prices, err = cpi.cloudInfoers[provider].GetCurrentPrices(region); err != nil {
		return nil, emperror.WrapWith(err, "failed to retrieve prices",
			"provider", provider, "region", region)
	}

	for instType, price := range prices {
		cpi.cloudInfoStore.StorePrice(provider, region, instType, price)
	}
	return prices, nil

}

func (cpi *cachingCloudInfo) toProviderAttribute(provider string, attr string) (string, error) {
	switch attr {
	case Cpu:
		return cpi.cloudInfoers[provider].GetCpuAttrName(), nil
	case Memory:
		return cpi.cloudInfoers[provider].GetMemoryAttrName(), nil
	}
	return "", emperror.With(errors.New("unsupported attribute"),
		"provider", provider, "attribute", attr)
}

func (cpi *cachingCloudInfo) renewVms(ctx context.Context, provider, service, regionId string) ([]VmInfo, error) {
	var (
		vms    []VmInfo
		values []VmInfo
		err    error
	)

	if vms, err = cpi.cloudInfoers[provider].GetVirtualMachines(regionId); err != nil {
		return nil, emperror.With(err, "provider", provider, "service", service, "region", regionId)
	}

	if values, err = cpi.cloudInfoers[provider].GetProducts(vms, service, regionId); err != nil {
		return nil, emperror.With(err, "provider", provider, "service", service, "region", regionId)
	}

	for _, vm := range values {
		if vm.OnDemandPrice > 0 {
			metrics.OnDemandPriceGauge.WithLabelValues(provider, regionId, vm.Type).Set(vm.OnDemandPrice)
		}
	}

	cpi.cloudInfoStore.StoreVm(provider, service, regionId, values)
	return values, nil
}

// GetZones returns the availability zones in a region
func (cpi *cachingCloudInfo) GetZones(ctx context.Context, provider string, region string) ([]string, error) {
	var (
		zones []string
		err   error
	)
	log := logger.Extract(ctx)

	// check the cache
	if cachedVal, ok := cpi.cloudInfoStore.GetZones(provider, region); ok {
		log.Debug("returning available zones from the cache")
		return cachedVal.([]string), nil
	}

	// retrieve zones from the provider
	if zones, err = cpi.cloudInfoers[provider].GetZones(region); err != nil {
		log.Error("error while retrieving zones.")
		return nil, emperror.With(err, "provider", provider, "region", region)
	}

	// cache the results / use the cache default expiry
	cpi.cloudInfoStore.StoreZones(provider, region, zones)
	return zones, nil
}

// GetRegions gets the regions for the provided provider
func (cpi *cachingCloudInfo) GetRegions(ctx context.Context, provider, service string) (map[string]string, error) {
	var (
		regions map[string]string
		err     error
	)
	log := logger.Extract(ctx)

	// check the cache
	if cachedVal, ok := cpi.cloudInfoStore.GetRegions(provider, service); ok {
		log.Debug("returning available regions from cache")
		return cachedVal.(map[string]string), nil
	}

	// retrieve regions from the provider
	if regions, err = cpi.cloudInfoers[provider].GetRegions(service); err != nil {
		log.Error("could not retrieve regions")
		return nil, emperror.With(err, "provider", provider, "service", service)
	}

	// cache the results / use the cache default expiry
	cpi.cloudInfoStore.StoreRegions(provider, service, regions)
	return regions, nil
}

func (cpi *cachingCloudInfo) GetServices(ctx context.Context, provider string) ([]Service, error) {
	var (
		services []Service
		err      error
	)
	log := logger.Extract(ctx)

	// check the cache
	if cachedVal, ok := cpi.cloudInfoStore.GetServices(provider); ok {
		log.Debug("returning available services from cache")
		return cachedVal.([]Service), nil
	}

	// retrieve services from the provider
	if services, err = cpi.cloudInfoers[provider].GetServices(); err != nil {
		log.Error("could not retrieve services")
		return nil, emperror.With(err, "provider", provider)
	}

	// cache the results / use the cache default expiry
	cpi.cloudInfoStore.StoreServices(provider, services)
	return services, nil
}

// GetProductDetails retrieves product details form the given provider and region
func (cpi *cachingCloudInfo) GetProductDetails(ctx context.Context, provider, service, region string) ([]ProductDetails, error) {
	var (
		vms interface{}
		ok  bool
	)
	log := logger.Extract(ctx)
	log.Info("retrieving product details")
	if vms, ok = cpi.cloudInfoStore.GetVm(provider, service, region); !ok {
		return nil, emperror.With(errors.New("vms not yet cached"),
			"provider", provider, "service", service, "region", region)
	}

	var details []ProductDetails

	var pr Price
	for _, vm := range vms.([]VmInfo) {
		pd := newProductDetails(vm)
		if cachedVal, ok := cpi.cloudInfoStore.GetPrice(provider, region, vm.Type); ok {
			pr = cachedVal.(Price)
			// fill the on demand price if appropriate
			if pr.OnDemandPrice > 0 {
				pd.OnDemandPrice = pr.OnDemandPrice
			}
			for zone, price := range pr.SpotPrice {
				pd.SpotInfo = append(pd.SpotInfo, *newZonePrice(zone, price))
			}
		} else {
			log.Debug("price info not yet cached", map[string]interface{}{"instanceType": vm.Type})
		}

		if pd.OnDemandPrice != 0 {
			details = append(details, *pd)
		}
	}

	return details, nil
}

// Contains is a helper function to check if a slice contains a string
func Contains(slice []string, s string) bool {
	for _, e := range slice {
		if e == s {
			return true
		}
	}
	return false
}

// GetStatus retrieves status form the given provider
func (cpi *cachingCloudInfo) GetStatus(provider string) (string, error) {
	if cachedStatus, ok := cpi.cloudInfoStore.GetStatus(provider); ok {
		return cachedStatus.(string), nil
	}
	return "", emperror.With(errors.New("status not yet cached"), "provider", provider)
}

func (cpi *cachingCloudInfo) renewImages(ctx context.Context, provider, service, regionId string) ([]Image, error) {
	var (
		values []Image
		err    error
	)

	if values, err = cpi.cloudInfoers[provider].GetServiceImages(service, regionId); err != nil {
		return nil, emperror.With(err, "provider", provider, "service", service, "region", regionId)
	}

	cpi.cloudInfoStore.StoreImage(provider, service, regionId, values)
	return values, nil
}

// GetServiceImages retrieves available images for the given provider, service and region
func (cpi *cachingCloudInfo) GetServiceImages(ctx context.Context, provider, service, region string) ([]Image, error) {
	logger.Extract(ctx).Debug("getting available images")

	if cachedImages, ok := cpi.cloudInfoStore.GetImage(provider, service, region); ok {
		return cachedImages.([]Image), nil
	}

	return nil, emperror.With(errors.New("images not yet cached"), "provider", provider,
		"service", service, "region", region)

}

func (cpi *cachingCloudInfo) renewVersions(ctx context.Context, provider, service, region string) ([]string, error) {
	var (
		values []string
		err    error
	)
	if values, err = cpi.cloudInfoers[provider].GetVersions(service, region); err != nil {
		return nil, emperror.With(errors.New("failed to renew versions"), "provider", provider, "service", service, "region", region)
	}
	cpi.cloudInfoStore.StoreVersion(provider, service, region, values)
	return values, nil

}

// GetVersions retrieves available versions for the given provider, service and region
func (cpi *cachingCloudInfo) GetVersions(ctx context.Context, provider, service, region string) ([]string, error) {
	logger.Extract(ctx).Debug("getting available versions")

	if cachedVersions, ok := cpi.cloudInfoStore.GetVersion(provider, service, region); ok {
		return cachedVersions.([]string), nil
	}
	return nil, emperror.With(errors.New("versions not yet cached"),
		"provider", provider, "service", service, "region", region)

}

// Attributes create a map with the specified parameters
func Attributes(cpu, memory, ntwPerfCat string) map[string]string {
	var attributes = make(map[string]string)

	attributes[Cpu] = cpu
	attributes[Memory] = memory
	attributes["NetworkPerfCategory"] = ntwPerfCat

	return attributes
}

// CreateString builds strings effectively
func CreateString(keys ...string) string {
	var b strings.Builder
	for _, key := range keys {
		b.WriteString(key)
	}
	return b.String()
}
