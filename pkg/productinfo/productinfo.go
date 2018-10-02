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

package productinfo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/banzaicloud/productinfo/pkg/logger"

	"github.com/prometheus/client_golang/prometheus"
)

// CachingProductInfo is the module struct, holds configuration and cache
// It's the entry point for the product info retrieval and management subsystem
// It's also responsible for delegating to the cloud provider specific implementations
type CachingProductInfo struct {
	productInfoers  map[string]ProductInfoer
	renewalInterval time.Duration
	vmAttrStore     ProductStorer
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
	Type          string        `json:"type"`
	OnDemandPrice float64       `json:"onDemandPrice"`
	SpotPrice     SpotPriceInfo `json:"spotPrice"`
	Cpus          float64       `json:"cpusPerVm"`
	Mem           float64       `json:"memPerVm"`
	Gpus          float64       `json:"gpusPerVm"`
	NtwPerf       string        `json:"ntwPerf"`
	NtwPerfCat    string        `json:"ntwPerfCategory"`
	Zones         []string      `json:"zones"`
	// CurrentGen signals whether the instance type generation is the current one. Only applies for amazon
	CurrentGen bool `json:"currentGen"`
}

var (
	scrapeCounterShortLived uint64
	scrapeCounterComplete   uint64
)

var (
	// ScrapeCompleteDurationGauge collects metrics for the prometheus
	ScrapeCompleteDurationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scrape",
		Name:      "complete_duration_seconds",
		Help:      "Cloud provider scrape complete duration in seconds",
	},
		[]string{"provider"},
	)
	// ScrapeRegionDurationGauge collects metrics for the prometheus
	ScrapeRegionDurationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scrape",
		Name:      "region_duration_seconds",
		Help:      "Cloud provider scrape region duration in seconds",
	},
		[]string{"provider", "service", "region"},
	)
	// ScrapeFailuresTotalCounter collects metrics for the prometheus
	ScrapeFailuresTotalCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scrape",
		Name:      "failures_total",
		Help:      "Total number of scrape failures, partitioned by provider and region",
	},
		[]string{"provider", "service", "region"},
	)
	// ScrapeShortLivedCompleteDurationGauge collects metrics for the prometheus
	ScrapeShortLivedCompleteDurationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scrape",
		Name:      "short_lived_complete_duration_seconds",
		Help:      "Cloud provider short lived scrape complete duration in seconds",
	},
		[]string{"provider"},
	)
	// ScrapeShortLivedRegionDurationGauge collects metrics for the prometheus
	ScrapeShortLivedRegionDurationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scrape",
		Name:      "short_lived_region_duration_seconds",
		Help:      "Cloud provider short lived scrape region duration in seconds",
	},
		[]string{"provider", "region"},
	)
	// ScrapeShortLivedFailuresTotalCounter collects metrics for the prometheus
	ScrapeShortLivedFailuresTotalCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scrape",
		Name:      "short_lived_failures_total",
		Help:      "Total number of short lived scrape failures, partitioned by provider and region",
	},
		[]string{"provider", "region"},
	)
)

// IsBurst returns true if the EC2 instance vCPU is burst type
// the decision is made based on the instance type
func (vm VmInfo) IsBurst() bool {
	return strings.HasPrefix(strings.ToUpper(vm.Type), "T")
}

// NewCachingProductInfo creates a new CachingProductInfo instance
func NewCachingProductInfo(ri time.Duration, cache ProductStorer, infoers map[string]ProductInfoer) (*CachingProductInfo, error) {
	if infoers == nil || cache == nil {
		return nil, errors.New("could not create product infoer")
	}

	pi := CachingProductInfo{
		productInfoers:  infoers,
		vmAttrStore:     cache,
		renewalInterval: ri,
	}
	return &pi, nil
}

// GetProviders returns the supported providers
func (cpi *CachingProductInfo) GetProviders(ctx context.Context) []Provider {
	var providers []Provider

	for name, infoer := range cpi.productInfoers {
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

// GetProvider returns the supported providers
func (cpi *CachingProductInfo) GetProvider(ctx context.Context, provider string) (Provider, error) {
	for p := range cpi.productInfoers {
		if provider == p {
			return NewProvider(provider), nil
		}
	}
	return Provider{}, fmt.Errorf("unsupported provider: [%s]", provider)
}

// renewProviderInfo renews provider information for the provider argument. It optionally signals the end of renewal to the
// provided WaitGroup (if provided)
func (cpi *CachingProductInfo) renewProviderInfo(ctx context.Context, provider string, wg *sync.WaitGroup) {
	log := logger.Extract(ctx)
	if wg != nil {
		defer wg.Done()
	}
	start := time.Now()
	// get the provider specific infoer
	pi := cpi.productInfoers[provider]

	log.Info("renewing product info")
	if _, err := cpi.Initialize(ctx, provider); err != nil {
		ScrapeFailuresTotalCounter.WithLabelValues(provider, "N/A", "N/A").Inc()
		log.WithError(err).Error("failed to renew product info")
		return
	}

	services, err := pi.GetServices()
	if err != nil {
		ScrapeFailuresTotalCounter.WithLabelValues(provider, "N/A", "N/A").Inc()
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
				ScrapeFailuresTotalCounter.WithLabelValues(provider, "N/A", "N/A").Inc()
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
			ScrapeFailuresTotalCounter.WithLabelValues(provider, service.ServiceName(), "N/A").Inc()
			logger.Extract(ctxLog).Error("failed to renew products")
			return
		}

		for regionId := range regions {
			c := logger.ToContext(ctxLog,
				logger.NewLogCtxBuilder().
					WithRegion(regionId).
					Build())

			start := time.Now()
			if _, err := cpi.renewVms(c, provider, service.ServiceName(), regionId); err != nil {
				ScrapeFailuresTotalCounter.WithLabelValues(provider, service.ServiceName(), regionId).Inc()
				logger.Extract(c).WithError(err).Error("failed to renew products")
			} else {
				ScrapeRegionDurationGauge.WithLabelValues(provider, service.ServiceName(), regionId).Set(time.Since(start).Seconds())
			}
		}
	}
	log.Info("finished to renew products (vm-s)")

	cpi.renewStatus(provider)
	ScrapeCompleteDurationGauge.WithLabelValues(provider).Set(time.Since(start).Seconds())
}

func (cpi *CachingProductInfo) renewStatus(provider string) (string, error) {
	values := strconv.Itoa(int(time.Now().UnixNano() / 1e6))

	cpi.vmAttrStore.Set(cpi.getStatusKey(provider), values, cpi.renewalInterval)
	return values, nil
}

// renewAll sequentially renews information for all provider
func (cpi *CachingProductInfo) renewAll(ctx context.Context) {
	atomic.AddUint64(&scrapeCounterComplete, 1)
	var providerWg sync.WaitGroup
	for provider := range cpi.productInfoers {
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

func (cpi *CachingProductInfo) renewShortLived(ctx context.Context) {
	atomic.AddUint64(&scrapeCounterShortLived, 1)
	var providerWg sync.WaitGroup
	for provider, infoer := range cpi.productInfoers {
		ctxWithFields := logger.ToContext(ctx, logger.NewLogCtxBuilder().
			WithProvider(provider).
			WithScrapeIdShort(atomic.LoadUint64(&scrapeCounterShortLived)).
			Build())

		providerWg.Add(1)
		go func(c context.Context, p string, i ProductInfoer) {
			defer providerWg.Done()
			if !i.HasShortLivedPriceInfo() {
				logger.Extract(c).Info("no short lived price info")
				return
			}

			logger.Extract(c).Info("renewing short lived product info")
			start := time.Now()

			regions, err := i.GetRegions(c, "compute")
			if err != nil {
				ScrapeShortLivedFailuresTotalCounter.WithLabelValues(p, "N/A").Inc()
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
						ScrapeShortLivedFailuresTotalCounter.WithLabelValues(p, r).Inc()
						logger.Extract(c).WithError(err).Error("couldn't renew short lived info in cache")
						return
					}
					ScrapeShortLivedRegionDurationGauge.WithLabelValues(p, r).Set(time.Since(start).Seconds())
				}(ctx, p, regionId)
			}
			wg.Wait()
			ScrapeShortLivedCompleteDurationGauge.WithLabelValues(p).Set(time.Since(start).Seconds())

		}(ctxWithFields, provider, infoer)
	}
	providerWg.Wait()
	logger.Extract(ctx).WithField("scrape-id-short", atomic.LoadUint64(&scrapeCounterShortLived)).Info("finished renewing short lived product info")
}

// Start starts the information retrieval in a new goroutine
func (cpi *CachingProductInfo) Start(ctx context.Context) {

	go cpi.renewAll(ctx)
	ticker := time.NewTicker(cpi.renewalInterval)
	go func(c context.Context) {
		for {
			select {
			case <-ticker.C:
				cpi.renewAll(c)
			case <-c.Done():
				logger.Extract(c).Debug("closing ticker")
				ticker.Stop()
				return
			}
		}
	}(ctx)
	go cpi.renewShortLived(ctx)
	shortTicker := time.NewTicker(4 * time.Minute)
	for {
		select {
		case <-shortTicker.C:
			cpi.renewShortLived(ctx)
		case <-ctx.Done():
			logger.Extract(ctx).Debug("closing ticker")
			shortTicker.Stop()
			return
		}
	}
}

// Initialize stores the result of the Infoer's Initialize output in cache
func (cpi *CachingProductInfo) Initialize(ctx context.Context, provider string) (map[string]map[string]Price, error) {
	log := logger.Extract(ctx)
	log.Info("start initializing product information")
	allPrices, err := cpi.productInfoers[provider].Initialize(ctx)

	if err != nil {
		log.WithError(err).Warn("failed to initialize product information")
		return nil, err
	}

	for region, ap := range allPrices {
		for instType, p := range ap {
			cpi.vmAttrStore.Set(cpi.getPriceKey(provider, region, instType), p, cpi.renewalInterval)
		}
	}
	log.Info("finished to initialize product information")
	return allPrices, nil
}

// GetAttributes returns the supported attribute names
func (cpi *CachingProductInfo) GetAttributes() []string {
	return []string{Cpu, Memory}
}

// GetAttrValues returns a slice with the values for the given attribute name
func (cpi *CachingProductInfo) GetAttrValues(ctx context.Context, provider, service, attribute string) ([]float64, error) {
	v, err := cpi.getAttrValues(ctx, provider, service, attribute)
	if err != nil {
		return nil, err
	}
	floatValues := v.floatValues()
	logger.Extract(ctx).Debugf("%s attribute values: %v", attribute, floatValues)
	return floatValues, nil
}

func (cpi *CachingProductInfo) getAttrValues(ctx context.Context, provider, service, attribute string) (AttrValues, error) {
	attrCacheKey := cpi.getAttrKey(provider, service, attribute)
	if cachedVal, ok := cpi.vmAttrStore.Get(attrCacheKey); ok {
		logger.Extract(ctx).Debugf("Getting available %s values from cache.", attribute)
		return cachedVal.(AttrValues), nil
	}
	values, err := cpi.renewAttrValues(ctx, provider, service, attribute)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (cpi *CachingProductInfo) getAttrKey(provider, service, attribute string) string {
	return fmt.Sprintf(AttrKeyTemplate, provider, service, attribute)
}

// renewAttrValues retrieves attribute values from the cloud provider and refreshes the attribute store with them
func (cpi *CachingProductInfo) renewAttrValues(ctx context.Context, provider, service, attribute string) (AttrValues, error) {
	attr, err := cpi.toProviderAttribute(provider, attribute)
	if err != nil {
		return nil, err
	}
	values, err := cpi.productInfoers[provider].GetAttributeValues(ctx, service, attr)
	if err != nil {
		return nil, err
	}
	cpi.vmAttrStore.Set(cpi.getAttrKey(provider, service, attribute), values, cpi.renewalInterval)
	return values, nil
}

// HasShortLivedPriceInfo signals if a product info provider has frequently changing price info
func (cpi *CachingProductInfo) HasShortLivedPriceInfo(provider string) bool {
	return cpi.productInfoers[provider].HasShortLivedPriceInfo()
}

// GetPrice returns the on demand price and zone averaged computed spot price for a given instance type in a given region
func (cpi *CachingProductInfo) GetPrice(ctx context.Context, provider string, region string, instanceType string, zones []string) (float64, float64, error) {
	var p Price
	ctx = logger.ToContext(ctx, logger.NewLogCtxBuilder().
		WithProvider(provider).
		WithRegion(region).
		Build())

	if cachedVal, ok := cpi.vmAttrStore.Get(cpi.getPriceKey(provider, region, instanceType)); ok {
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

func (cpi *CachingProductInfo) getPriceKey(provider string, region string, instanceType string) string {
	return fmt.Sprintf(PriceKeyTemplate, provider, region, instanceType)
}

// renewAttrValues retrieves attribute values from the cloud provider and refreshes the attribute store with them
func (cpi *CachingProductInfo) renewShortLivedInfo(ctx context.Context, provider string, region string) (map[string]Price, error) {
	prices, err := cpi.productInfoers[provider].GetCurrentPrices(ctx, region)
	if err != nil {
		return nil, err
	}
	for instType, p := range prices {
		cpi.vmAttrStore.Set(cpi.getPriceKey(provider, region, instType), p, 8*time.Minute)
	}
	return prices, nil
}

func (cpi *CachingProductInfo) toProviderAttribute(provider string, attr string) (string, error) {
	switch attr {
	case Cpu:
		return cpi.productInfoers[provider].GetCpuAttrName(), nil
	case Memory:
		return cpi.productInfoers[provider].GetMemoryAttrName(), nil
	}
	return "", fmt.Errorf("unsupported attribute: %s", attr)
}

func (cpi *CachingProductInfo) getVmKey(provider, service, region string) string {
	return fmt.Sprintf(VmKeyTemplate, provider, service, region)
}

func (cpi *CachingProductInfo) renewVms(ctx context.Context, provider, service, regionId string) ([]VmInfo, error) {
	values, err := cpi.productInfoers[provider].GetProducts(ctx, service, regionId)
	if err != nil {
		return nil, err
	}
	cpi.vmAttrStore.Set(cpi.getVmKey(provider, service, regionId), values, cpi.renewalInterval)
	return values, nil
}

// GetZones returns the availability zones in a region
func (cpi *CachingProductInfo) GetZones(ctx context.Context, provider string, region string) ([]string, error) {
	log := logger.Extract(ctx)
	zoneCacheKey := cpi.getZonesKey(provider, region)

	// check the cache
	if cachedVal, ok := cpi.vmAttrStore.Get(zoneCacheKey); ok {
		log.Debug("Getting available zones from cache.")
		return cachedVal.([]string), nil
	}

	// retrieve zones from the provider
	zones, err := cpi.productInfoers[provider].GetZones(ctx, region)
	if err != nil {
		log.WithError(err).Error("error while retrieving zones.")
		return nil, err
	}

	// cache the results / use the cache default expiry
	cpi.vmAttrStore.Set(zoneCacheKey, zones, 0)
	return zones, nil
}

func (cpi *CachingProductInfo) getZonesKey(provider string, region string) string {
	return fmt.Sprintf(ZoneKeyTemplate, provider, region)
}

// GetNetworkPerfMapper returns the provider specific network performance mapper
func (cpi *CachingProductInfo) GetNetworkPerfMapper(provider string) (NetworkPerfMapper, error) {
	if infoer, ok := cpi.productInfoers[provider]; ok {
		return infoer.GetNetworkPerformanceMapper() // this also can return with err!
	}
	return nil, fmt.Errorf("could not retrieve network perf mapper for provider: [%s]", provider)
}

// GetRegions gets the regions for the provided provider
func (cpi *CachingProductInfo) GetRegions(ctx context.Context, provider, service string) (map[string]string, error) {
	log := logger.Extract(ctx)
	regionCacheKey := cpi.getRegionsKey(provider, service)

	// check the cache
	if cachedVal, ok := cpi.vmAttrStore.Get(regionCacheKey); ok {

		log.Debug("Getting available regions from cache.")
		return cachedVal.(map[string]string), nil
	}

	// retrieve regions from the provider
	regions, err := cpi.productInfoers[provider].GetRegions(ctx, service)
	if err != nil {
		log.WithError(err).Error("could not retrieve regions.")
		return nil, err
	}

	// cache the results / use the cache default expiry
	cpi.vmAttrStore.Set(regionCacheKey, regions, 0)
	return regions, nil
}

func (cpi *CachingProductInfo) getRegionsKey(provider, service string) string {
	return fmt.Sprintf(RegionKeyTemplate, provider, service)
}

// GetProductDetails retrieves product details form the given provider and region
func (cpi *CachingProductInfo) GetProductDetails(ctx context.Context, provider, service, region string) ([]ProductDetails, error) {
	log := logger.Extract(ctx)
	log.Debug("getting product details")
	cachedVms, ok := cpi.vmAttrStore.Get(cpi.getVmKey(provider, service, region))
	if !ok {
		return nil, fmt.Errorf("vms not yet cached for the key: %s", cpi.getVmKey(provider, service, region))
	}

	vms := cachedVms.([]VmInfo)
	var details []ProductDetails

	var pr Price
	for _, vm := range vms {
		pd := newProductDetails(vm)
		pdWithNtwPerfCat := cpi.decorateNtwPerfCat(provider, pd)
		if cachedVal, ok := cpi.vmAttrStore.Get(cpi.getPriceKey(provider, region, vm.Type)); ok {
			pr = cachedVal.(Price)
			// fill the on demand price if appropriate
			if pr.OnDemandPrice > 0 {
				pdWithNtwPerfCat.OnDemandPrice = pr.OnDemandPrice
			}
			for zone, price := range pr.SpotPrice {
				pdWithNtwPerfCat.SpotInfo = append(pdWithNtwPerfCat.SpotInfo, *newZonePrice(zone, price))
			}
		} else {
			log.Debugf("price info not yet cached for key: %s", cpi.getPriceKey(provider, region, vm.Type))
		}

		if pdWithNtwPerfCat.OnDemandPrice != 0 {
			details = append(details, *pdWithNtwPerfCat)
		}
	}

	return details, nil
}

// decorateNtwPerfCat returns ProductDetails with network performance category
func (cpi *CachingProductInfo) decorateNtwPerfCat(provider string, pd *ProductDetails) *ProductDetails {
	ntwMapper, _ := cpi.GetNetworkPerfMapper(provider)
	pd.NtwPerfCat, _ = ntwMapper.MapNetworkPerf(pd.VmInfo)
	return pd
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
func (cpi *CachingProductInfo) GetStatus(provider string) (string, error) {

	cachedStatus, ok := cpi.vmAttrStore.Get(cpi.getStatusKey(provider))
	if !ok {
		return "", fmt.Errorf("status not yet cached for the key: %s", cpi.getStatusKey(provider))
	}
	status := cachedStatus.(string)

	return status, nil
}

func (cpi *CachingProductInfo) getStatusKey(provider string) string {
	return fmt.Sprintf(StatusKeyTemplate, provider)
}

// GetInfoer returns the provider specific infoer implementation. This method is the discriminator for cloud providers
func (cpi *CachingProductInfo) GetInfoer(provider string) (ProductInfoer, error) {

	if infoer, ok := cpi.productInfoers[provider]; ok {
		return infoer, nil
	}

	return nil, fmt.Errorf("could not find infoer for: [ %s ]", provider)
}
