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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/goph/emperror"
)

// CachingCloudInfo is the module struct, holds configuration and cache
// It's the entry point for the product info retrieval and management subsystem
// It's also responsible for delegating to the cloud provider specific implementations
type CachingCloudInfo struct {
	cloudInfoers    map[string]CloudInfoer
	renewalInterval time.Duration
	cloudInfoStore  CloudInfoStore
	metrics         metrics.Reporter
}

func (cpi *CachingCloudInfo) HasShortLivedPriceInfo(ctx context.Context, provider string) bool {
	if cier, err := cpi.GetInfoer(ctx, provider); err != nil {
		return cier.HasShortLivedPriceInfo()
	}
	return false
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

var (
	scrapeCounterShortLived uint64
	scrapeCounterComplete   uint64
)

// IsBurst returns true if the EC2 instance vCPU is burst type
// the decision is made based on the instance type
func (vm VmInfo) IsBurst() bool {
	return strings.HasPrefix(strings.ToUpper(vm.Type), "T")
}

// NewCachingCloudInfo creates a new CachingCloudInfo instance
func NewCachingCloudInfo(ri time.Duration, ciStore CloudInfoStore, infoers map[string]CloudInfoer, reporter metrics.Reporter) (*CachingCloudInfo, error) {
	if infoers == nil || ciStore == nil {
		return nil, errors.New("could not create product infoer")
	}

	pi := CachingCloudInfo{
		cloudInfoers:    infoers,
		cloudInfoStore:  ciStore,
		renewalInterval: ri,
		metrics:         reporter,
	}
	return &pi, nil
}

// GetProviders returns the supported providers
func (cpi *CachingCloudInfo) GetProviders(ctx context.Context) []Provider {
	var providers []Provider

	for name, infoer := range cpi.cloudInfoers {
		services, err := infoer.GetServices()
		if err != nil {
			logger.Extract(ctx).WithField("provider", name).WithError(err).Error("could not retrieve services")
		}

		// decorate the provider with service information
		svcs := make([]Service, 0)
		for _, s := range services {
			svcs = append(svcs, NewService(s.ServiceName()))
		}
		provider := NewProvider(name)
		provider.Services = svcs

		providers = append(providers, provider)
	}
	return providers
}

// GetProvider returns the supported provider
func (cpi *CachingCloudInfo) GetProvider(ctx context.Context, provider string) (Provider, error) {
	for name, infoer := range cpi.cloudInfoers {
		if provider == name {
			services, err := infoer.GetServices()
			if err != nil {
				logger.Extract(ctx).WithField("provider", name).WithError(err).Error("could not retrieve services")
			}

			// decorate the provider with service information
			svcs := make([]Service, 0)
			for _, s := range services {
				svcs = append(svcs, NewService(s.ServiceName()))
			}

			p := NewProvider(name)
			p.Services = svcs

			return p, nil
		}
	}
	return Provider{}, emperror.With(errors.New("unsupported provider"), "provider", provider)
}

// renewProviderInfo renews provider information for the provider argument. It optionally signals the end of renewal to the
// provided WaitGroup (if provided)
func (cpi *CachingCloudInfo) renewProviderInfo(ctx context.Context, provider string, wg *sync.WaitGroup) {
	log := logger.Extract(ctx)
	if wg != nil {
		defer wg.Done()
	}
	start := time.Now()
	// get the provider specific infoer
	pi := cpi.cloudInfoers[provider]

	log.Info("renewing product info")
	if _, err := cpi.Initialize(ctx, provider); err != nil {
		cpi.metrics.ReportScrapeFailure(provider, "N/A", "N/A")
		log.WithError(err).Error("failed to renew product info")
		return
	}

	services, err := pi.GetServices()
	if err != nil {
		cpi.metrics.ReportScrapeFailure(provider, "N/A", "N/A")
		log.WithError(err).Error("failed to renew products")
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
				logger.Extract(ctxLog).WithError(err).Errorf("failed to renew %s attribute values", attr)
				return
			}
		}
	}
	log.Info("finished to renew attribute values")

	log.Info("start to renew products (vm-s)")
	for _, service := range services {
		ctxLog := logger.ToContext(ctx,
			logger.NewLogCtxBuilder().
				WithService(service.ServiceName()).
				Build())
		regions, err := pi.GetRegions(ctx, service.ServiceName())
		if err != nil {
			cpi.metrics.ReportScrapeFailure(provider, service.ServiceName(), "N/A")
			logger.Extract(ctxLog).Error("failed to renew products")
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
				logger.Extract(c).WithError(err).Error("failed to renew products")
			}
			if pi.HasImages() {
				_, imgErr := cpi.renewImages(c, provider, service.ServiceName(), regionId)
				if imgErr != nil {
					cpi.metrics.ReportScrapeFailure(provider, service.ServiceName(), regionId)
					logger.Extract(c).WithError(imgErr).Error("failed to renew images")
				}
			}
			_, versionErr := cpi.renewVersions(c, provider, service.ServiceName(), regionId)
			if versionErr != nil {
				cpi.metrics.ReportScrapeFailure(provider, service.ServiceName(), regionId)
				logger.Extract(c).WithError(versionErr).Error("failed to renew versions")
			}
			if err == nil && versionErr == nil {
				cpi.metrics.ReportScrapeRegionCompleted(provider, service.ServiceName(), regionId, start)
			}
		}
	}
	log.Info("finished to renew products (vm-s)")

	if _, err := cpi.renewStatus(provider); err != nil {
		log.Errorf("failed to renew status: %s", err)
		return
	}
	cpi.metrics.ReportScrapeProviderCompleted(provider, start)
}

func (cpi *CachingCloudInfo) renewStatus(provider string) (string, error) {
	values := strconv.Itoa(int(time.Now().UnixNano() / 1e6))

	cpi.cloudInfoStore.StoreStatus(provider, values)
	return values, nil
}

// renewAll sequentially renews information for all provider
func (cpi *CachingCloudInfo) renewAll(ctx context.Context) {
	atomic.AddUint64(&scrapeCounterComplete, 1)
	var providerWg sync.WaitGroup
	for provider := range cpi.cloudInfoers {
		ctxWithFields := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(provider).
			WithScrapeIdFull(atomic.LoadUint64(&scrapeCounterComplete)).
			Build())
		providerWg.Add(1)
		go cpi.renewProviderInfo(ctxWithFields, provider, &providerWg)
	}
	providerWg.Wait()
	logger.Extract(ctx).WithField("scrape-id-full", atomic.LoadUint64(&scrapeCounterComplete)).Info("finished renewing product info")
}

func (cpi *CachingCloudInfo) renewShortLived(ctx context.Context) {
	atomic.AddUint64(&scrapeCounterShortLived, 1)
	var providerWg sync.WaitGroup
	for provider, infoer := range cpi.cloudInfoers {
		ctxWithFields := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(provider).
			WithScrapeIdShort(atomic.LoadUint64(&scrapeCounterShortLived)).
			Build())

		providerWg.Add(1)
		go func(c context.Context, p string, i CloudInfoer) {
			defer providerWg.Done()
			if !i.HasShortLivedPriceInfo() {
				logger.Extract(c).Info("no short lived price info")
				return
			}

			logger.Extract(c).Info("renewing short lived product info")
			start := time.Now()

			regions, err := i.GetRegions(c, "compute")
			if err != nil {
				cpi.metrics.ReportScrapeShortLivedFailure(p, "N/A")
				logger.Extract(c).WithError(err).Error("couldn't renew attribute values in cache")
				return
			}
			var wg sync.WaitGroup
			for regionId := range regions {
				ctx := logger.ToContext(c, logger.NewLogCtxBuilder().
					WithRegion(regionId).
					Build())

				wg.Add(1)
				go func(c context.Context, p string, r string) {
					defer wg.Done()
					_, err := cpi.renewShortLivedInfo(c, p, r)
					if err != nil {
						cpi.metrics.ReportScrapeShortLivedFailure(p, r)
						logger.Extract(c).WithError(err).Error("couldn't renew short lived info in cache")
						return
					}
					cpi.metrics.ReportScrapeRegionShortLivedCompleted(p, r, start)
				}(ctx, p, regionId)
			}
			wg.Wait()
			cpi.metrics.ReportScrapeProviderShortLivedCompleted(p, start)

		}(ctxWithFields, provider, infoer)
	}
	providerWg.Wait()
	logger.Extract(ctx).WithField("scrape-id-short", atomic.LoadUint64(&scrapeCounterShortLived)).Info("finished renewing short lived product info")
}

// Start starts the information retrieval in a new goroutine
func (cpi *CachingCloudInfo) Start(ctx context.Context) {

	// start scraping providers for vm information
	if err := NewPeriodicExecutor(cpi.renewalInterval).Execute(ctx, cpi.renewAll); err != nil {
		logger.Extract(ctx).Info("failed to scrape for vm information")
		return
	}

	// start scraping providers for pricing information
	if err := NewPeriodicExecutor(4 * time.Minute).Execute(ctx, cpi.renewShortLived); err != nil {
		logger.Extract(ctx).Info("failed to scrape for pricing information")
		return
	}

}

// Initialize stores the result of the Infoer's Initialize output in cache
func (cpi *CachingCloudInfo) Initialize(ctx context.Context, provider string) (map[string]map[string]Price, error) {
	log := logger.Extract(ctx)
	log.Info("initializing cloud product information")
	allPrices, err := cpi.cloudInfoers[provider].Initialize(ctx)
	if err != nil {
		log.WithError(err).Warn("failed to initialize cloud product information")
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
func (cpi *CachingCloudInfo) GetAttributes(ctx context.Context) []string {
	return []string{Cpu, Memory}
}

// GetAttrValues returns a slice with the values for the given attribute name
func (cpi *CachingCloudInfo) GetAttrValues(ctx context.Context, provider, service, attribute string) ([]float64, error) {
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
func (cpi *CachingCloudInfo) renewAttrValues(ctx context.Context, provider, service, attribute string) (AttrValues, error) {
	var (
		attr   string
		err    error
		values AttrValues
	)

	if attr, err = cpi.toProviderAttribute(provider, attribute); err != nil {
		return nil, emperror.With(err, "renewal")
	}

	if values, err = cpi.cloudInfoers[provider].GetAttributeValues(ctx, service, attr); err != nil {
		return nil, emperror.With(err, "renewal")
	}

	cpi.cloudInfoStore.StoreAttribute(provider, service, attribute, values)
	return values, nil
}

// GetPrice returns the on demand price and zone averaged computed spot price for a given instance type in a given region
func (cpi *CachingCloudInfo) GetPrice(ctx context.Context, provider string, region string, instanceType string, zones []string) (float64, float64, error) {
	var p Price
	ctx = logger.ToContext(ctx, logger.NewLogCtxBuilder().
		WithProvider(provider).
		WithRegion(region).
		Build())

	if cachedVal, ok := cpi.cloudInfoStore.GetPrice(provider, region, instanceType); ok {
		logger.Extract(ctx).Debugf("Getting price info from cache [instance type=%s].", instanceType)
		p = cachedVal.(Price)
	} else {
		allPriceInfo, err := cpi.renewShortLivedInfo(ctx, provider, region)
		if err != nil {
			return 0, 0, err
		}
		p = allPriceInfo[instanceType]
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
func (cpi *CachingCloudInfo) renewShortLivedInfo(ctx context.Context, provider string, region string) (map[string]Price, error) {

	var (
		err    error
		prices map[string]Price
	)

	if prices, err = cpi.cloudInfoers[provider].GetCurrentPrices(ctx, region); err != nil {
		return nil, emperror.With(err, "failed to retrieve prices",
			"provider", provider, "region", region)
	}

	for instType, price := range prices {
		cpi.cloudInfoStore.StorePrice(provider, region, instType, price)
	}
	return prices, nil

}

func (cpi *CachingCloudInfo) toProviderAttribute(provider string, attr string) (string, error) {
	switch attr {
	case Cpu:
		return cpi.cloudInfoers[provider].GetCpuAttrName(), nil
	case Memory:
		return cpi.cloudInfoers[provider].GetMemoryAttrName(), nil
	}
	return "", emperror.With(errors.New("unsupported attribute"),
		"provider", provider, "attribute", attr)
}

func (cpi *CachingCloudInfo) renewVms(ctx context.Context, provider, service, regionId string) ([]VmInfo, error) {
	var (
		values []VmInfo
		err    error
	)

	if values, err = cpi.cloudInfoers[provider].GetProducts(ctx, service, regionId); err != nil {
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
func (cpi *CachingCloudInfo) GetZones(ctx context.Context, provider string, region string) ([]string, error) {
	var (
		zones []string
		err   error
	)
	log := logger.Extract(ctx)

	// check the cache
	if cachedVal, ok := cpi.cloudInfoStore.GetZone(provider, region); ok {
		log.Debug("returning available zones from the cache")
		return cachedVal.([]string), nil
	}

	// retrieve zones from the provider
	if zones, err = cpi.cloudInfoers[provider].GetZones(ctx, region); err != nil {
		log.WithError(err).Error("error while retrieving zones.")
		return nil, emperror.With(err, "provider", provider, "region", region)
	}

	// cache the results / use the cache default expiry
	cpi.cloudInfoStore.StoreZone(provider, region, zones)
	return zones, nil
}

// GetRegions gets the regions for the provided provider
func (cpi *CachingCloudInfo) GetRegions(ctx context.Context, provider, service string) (map[string]string, error) {
	var (
		regions map[string]string
		err     error
	)
	log := logger.Extract(ctx)

	// check the cache
	if cachedVal, ok := cpi.cloudInfoStore.GetRegion(provider, service); ok {
		log.Debug("returning available regions from cache")
		return cachedVal.(map[string]string), nil
	}

	// retrieve regions from the provider
	if regions, err = cpi.cloudInfoers[provider].GetRegions(ctx, service); err != nil {
		log.WithError(err).Error("could not retrieve regions.")
		return nil, emperror.With(err, "provider", provider, "service", service)
	}

	// cache the results / use the cache default expiry
	cpi.cloudInfoStore.StoreRegion(provider, service, regions)
	return regions, nil
}

// GetProductDetails retrieves product details form the given provider and region
func (cpi *CachingCloudInfo) GetProductDetails(ctx context.Context, provider, service, region string) ([]ProductDetails, error) {
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
			log.WithField("instanceType", vm.Type).Debugf("price info not yet cached")
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
func (cpi *CachingCloudInfo) GetStatus(provider string) (string, error) {
	if cachedStatus, ok := cpi.cloudInfoStore.GetStatus(provider); ok {
		return cachedStatus.(string), nil
	}
	return "", emperror.With(errors.New("status not yet cached"), "provider", provider)
}

func (cpi *CachingCloudInfo) GetInfoer(ctx context.Context, provider string) (CloudInfoer, error) {
	if infoer, ok := cpi.cloudInfoers[provider]; ok {
		return infoer, nil
	}
	return nil, emperror.With(errors.New("could not find infoer for provider"), "provider", provider)
}

func (cpi *CachingCloudInfo) renewImages(ctx context.Context, provider, service, regionId string) ([]ImageDescriber, error) {
	var (
		values []ImageDescriber
		err    error
	)

	if values, err = cpi.cloudInfoers[provider].GetServiceImages(service, regionId); err != nil {
		return nil, emperror.With(err, "provider", provider, "service", service, "region", regionId)
	}

	cpi.cloudInfoStore.StoreImage(provider, service, regionId, values)
	return values, nil
}

// GetServiceImages retrieves available images for the given provider, service and region
func (cpi *CachingCloudInfo) GetServiceImages(ctx context.Context, provider, service, region string) ([]ImageDescriber, error) {
	logger.Extract(ctx).Debug("getting available images")

	if cachedImages, ok := cpi.cloudInfoStore.GetImage(provider, service, region); ok {
		return cachedImages.([]ImageDescriber), nil
	}

	return nil, emperror.With(errors.New("images not yet cached"), "provider", provider,
		"service", service, "region", region)

}

func (cpi *CachingCloudInfo) renewVersions(ctx context.Context, provider, service, region string) ([]string, error) {
	var (
		values []string
		err    error
	)

	if values, err = cpi.cloudInfoers[provider].GetVersions(ctx, service, region); err != nil {
		return nil, emperror.With(errors.New("failed to renew versions"), "provider", provider, "service", service, "region", region)
	}

	cpi.cloudInfoStore.StoreVersion(provider, service, region, values)
	return values, nil

}

// GetVersions retrieves available versions for the given provider, service and region
func (cpi *CachingCloudInfo) GetVersions(ctx context.Context, provider, service, region string) ([]string, error) {
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
