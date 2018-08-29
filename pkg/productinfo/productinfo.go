package productinfo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
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
	// CurrentGen signals whether the instance type generation is the current one. Only applies for amazon
	CurrentGen bool `json:"currentGen"`
}

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
		[]string{"provider", "region"},
	)
	// ScrapeFailuresTotalCounter collects metrics for the prometheus
	ScrapeFailuresTotalCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scrape",
		Name:      "failures_total",
		Help:      "Total number of scrape failures, partitioned by provider and region",
	},
		[]string{"provider", "region"},
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
func (cpi *CachingProductInfo) GetProviders() []string {
	var providers []string
	for p := range cpi.productInfoers {
		providers = append(providers, p)
	}
	return providers
}

// renewProviderInfo renews provider information for the provider argument. It optionally signals the end of renewal to the
// provided WaitGroup (if provided)
func (cpi *CachingProductInfo) renewProviderInfo(provider string, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	start := time.Now()
	// get the provider specific infoer
	pi := cpi.productInfoers[provider]

	log.Infof("renewing product info for provider [%s]", provider)
	if _, err := cpi.Initialize(provider); err != nil {
		ScrapeFailuresTotalCounter.WithLabelValues(provider, "N/A").Inc()
		log.Errorf("couldn't renew attribute values in cache: %s", err.Error())
		return
	}
	attributes := []string{Cpu, Memory}
	for _, attr := range attributes {
		if _, err := cpi.renewAttrValues(provider, attr); err != nil {
			ScrapeFailuresTotalCounter.WithLabelValues(provider, "N/A").Inc()
			log.Errorf("couldn't renew attribute values in cache: %s", err.Error())
			return
		}
	}
	if regions, err := pi.GetRegions(); err == nil {
		for regionId := range regions {
			start := time.Now()
			if _, err := cpi.renewVms(provider, regionId); err != nil {
				ScrapeFailuresTotalCounter.WithLabelValues(provider, regionId).Inc()
				log.Errorf("couldn't renew attribute values in cache: %s", err.Error())
			} else {
				ScrapeRegionDurationGauge.WithLabelValues(provider, regionId).Set(time.Since(start).Seconds())
			}
		}
	} else {
		ScrapeFailuresTotalCounter.WithLabelValues(provider, "N/A").Inc()
		log.Errorf("couldn't renew attribute values in cache: %s", err.Error())
		return
	}
	cpi.renewStatus(provider)

	ScrapeCompleteDurationGauge.WithLabelValues(provider).Set(time.Since(start).Seconds())
}

func (cpi *CachingProductInfo) renewStatus(provider string) (string, error) {
	values := strconv.Itoa(int(time.Now().UnixNano() / 1e6))

	cpi.vmAttrStore.Set(cpi.getStatusKey(provider), values, cpi.renewalInterval)
	return values, nil
}

// renewAll sequentially renews information for all provider
func (cpi *CachingProductInfo) renewAll() {
	var providerWg sync.WaitGroup
	for provider := range cpi.productInfoers {
		providerWg.Add(1)
		go cpi.renewProviderInfo(provider, &providerWg)
	}
	providerWg.Wait()
	log.Info("finished renewing product info")
}

func (cpi *CachingProductInfo) renewShortLived() {
	var providerWg sync.WaitGroup
	for provider, infoer := range cpi.productInfoers {
		providerWg.Add(1)
		go func(p string, i ProductInfoer) {
			defer providerWg.Done()
			if i.HasShortLivedPriceInfo() {
				log.Infof("renewing short lived %s product info", p)
				start := time.Now()
				var wg sync.WaitGroup
				regions, err := i.GetRegions()
				if err != nil {
					ScrapeShortLivedFailuresTotalCounter.WithLabelValues(p, "N/A").Inc()
					log.Errorf("couldn't renew attribute values in cache: %s", err.Error())
					return
				}
				for regionId := range regions {
					wg.Add(1)
					go func(p string, r string) {
						defer wg.Done()
						_, err := cpi.renewShortLivedInfo(p, r)
						if err != nil {
							ScrapeShortLivedFailuresTotalCounter.WithLabelValues(p, r).Inc()
							log.Errorf("couldn't renew short lived info in cache: %s", err.Error())
							return
						}
						ScrapeShortLivedRegionDurationGauge.WithLabelValues(p, r).Set(time.Since(start).Seconds())
					}(p, regionId)
				}
				wg.Wait()
				ScrapeShortLivedCompleteDurationGauge.WithLabelValues(p).Set(time.Since(start).Seconds())
			}
		}(provider, infoer)
	}
	providerWg.Wait()
	log.Info("finished renewing short lived product info")
}

// Start starts the information retrieval in a new goroutine
func (cpi *CachingProductInfo) Start(ctx context.Context) {

	go cpi.renewAll()
	ticker := time.NewTicker(cpi.renewalInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				cpi.renewAll()
			case <-ctx.Done():
				log.Debugf("closing ticker")
				ticker.Stop()
				return
			}
		}
	}()
	go cpi.renewShortLived()
	shortTicker := time.NewTicker(1 * time.Minute)
	for {
		select {
		case <-shortTicker.C:
			cpi.renewShortLived()
		case <-ctx.Done():
			log.Debugf("closing ticker")
			shortTicker.Stop()
			return
		}
	}
}

// Initialize stores the result of the Infoer's Initialize output in cache
func (cpi *CachingProductInfo) Initialize(provider string) (map[string]map[string]Price, error) {
	allPrices, err := cpi.productInfoers[provider].Initialize()
	if err != nil {
		return nil, err
	}
	for region, ap := range allPrices {
		for instType, p := range ap {
			cpi.vmAttrStore.Set(cpi.getPriceKey(provider, region, instType), p, cpi.renewalInterval)
		}
	}
	return allPrices, nil
}

// GetAttributes returns the supported attribute names
func (cpi *CachingProductInfo) GetAttributes() []string {
	return []string{Cpu, Memory}
}

// GetAttrValues returns a slice with the values for the given attribute name
func (cpi *CachingProductInfo) GetAttrValues(provider string, attribute string) ([]float64, error) {
	v, err := cpi.getAttrValues(provider, attribute)
	if err != nil {
		return nil, err
	}
	floatValues := v.floatValues()
	log.Debugf("%s attribute values: %v", attribute, floatValues)
	return floatValues, nil
}

func (cpi *CachingProductInfo) getAttrValues(provider string, attribute string) (AttrValues, error) {
	attrCacheKey := cpi.getAttrKey(provider, attribute)
	if cachedVal, ok := cpi.vmAttrStore.Get(attrCacheKey); ok {
		log.Debugf("Getting available %s values from cache.", attribute)
		return cachedVal.(AttrValues), nil
	}
	values, err := cpi.renewAttrValues(provider, attribute)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (cpi *CachingProductInfo) getAttrKey(provider string, attribute string) string {
	return fmt.Sprintf(AttrKeyTemplate, provider, attribute)
}

// renewAttrValues retrieves attribute values from the cloud provider and refreshes the attribute store with them
func (cpi *CachingProductInfo) renewAttrValues(provider string, attribute string) (AttrValues, error) {
	attr, err := cpi.toProviderAttribute(provider, attribute)
	if err != nil {
		return nil, err
	}
	values, err := cpi.productInfoers[provider].GetAttributeValues(attr)
	if err != nil {
		return nil, err
	}
	cpi.vmAttrStore.Set(cpi.getAttrKey(provider, attribute), values, cpi.renewalInterval)
	return values, nil
}

// HasShortLivedPriceInfo signals if a product info provider has frequently changing price info
func (cpi *CachingProductInfo) HasShortLivedPriceInfo(provider string) bool {
	return cpi.productInfoers[provider].HasShortLivedPriceInfo()
}

// GetPrice returns the on demand price and zone averaged computed spot price for a given instance type in a given region
func (cpi *CachingProductInfo) GetPrice(provider string, region string, instanceType string, zones []string) (float64, float64, error) {
	var p Price
	if cachedVal, ok := cpi.vmAttrStore.Get(cpi.getPriceKey(provider, region, instanceType)); ok {
		log.Debugf("Getting price info from cache [provider=%s, region=%s, type=%s].", provider, region, instanceType)
		p = cachedVal.(Price)
	} else {
		allPriceInfo, err := cpi.renewShortLivedInfo(provider, region)
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
func (cpi *CachingProductInfo) renewShortLivedInfo(provider string, region string) (map[string]Price, error) {
	prices, err := cpi.productInfoers[provider].GetCurrentPrices(region)
	if err != nil {
		return nil, err
	}
	for instType, p := range prices {
		cpi.vmAttrStore.Set(cpi.getPriceKey(provider, region, instType), p, 2*time.Minute)
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

func (cpi *CachingProductInfo) getVmKey(provider string, region string) string {
	return fmt.Sprintf(VmKeyTemplate, provider, region)
}

func (cpi *CachingProductInfo) renewVms(provider string, regionId string) ([]VmInfo, error) {
	values, err := cpi.productInfoers[provider].GetProducts(regionId)
	if err != nil {
		return nil, err
	}
	cpi.vmAttrStore.Set(cpi.getVmKey(provider, regionId), values, cpi.renewalInterval)
	return values, nil
}

// GetZones returns the availability zones in a region
func (cpi *CachingProductInfo) GetZones(provider string, region string) ([]string, error) {
	zoneCacheKey := cpi.getZonesKey(provider, region)

	// check the cache
	if cachedVal, ok := cpi.vmAttrStore.Get(zoneCacheKey); ok {
		log.Debugf("Getting available zones from cache. [provider=%s, region=%s]", provider, region)
		return cachedVal.([]string), nil
	}

	// retrieve zones from the provider
	zones, err := cpi.productInfoers[provider].GetZones(region)
	if err != nil {
		log.Errorf("error while retrieving zones. provider: %s, region: %s", provider, region)
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
func (cpi *CachingProductInfo) GetRegions(provider string) (map[string]string, error) {
	regionCacheKey := cpi.getRegionsKey(provider)

	// check the cache
	if cachedVal, ok := cpi.vmAttrStore.Get(regionCacheKey); ok {
		log.Debugf("Getting available regions from cache. [provider=%s]", provider)
		return cachedVal.(map[string]string), nil
	}

	// retrieve regions from the provider
	regions, err := cpi.productInfoers[provider].GetRegions()
	if err != nil {
		log.Errorf("could not retrieve regions. provider: %s", provider)
		return nil, err
	}

	// cache the results / use the cache default expiry
	cpi.vmAttrStore.Set(regionCacheKey, regions, 0)
	return regions, nil
}

func (cpi *CachingProductInfo) getRegionsKey(provider string) string {
	return fmt.Sprintf(RegionKeyTemplate, provider)
}

// GetProductDetails retrieves product details form the given provider and region
func (cpi *CachingProductInfo) GetProductDetails(cloud string, region string) ([]ProductDetails, error) {
	log.Debugf("getting product details for provider: %s, region: %s", cloud, region)

	cachedVms, ok := cpi.vmAttrStore.Get(cpi.getVmKey(cloud, region))
	if !ok {
		return nil, fmt.Errorf("vms not yet cached for the key: %s", cpi.getVmKey(cloud, region))
	}

	vms := cachedVms.([]VmInfo)
	var details []ProductDetails

	var pr Price
	for _, vm := range vms {
		pd := newProductDetails(vm)
		pdWithNtwPerfCat := cpi.decorateNtwPerfCat(cloud, pd)
		if cachedVal, ok := cpi.vmAttrStore.Get(cpi.getPriceKey(cloud, region, vm.Type)); ok {
			pr = cachedVal.(Price)
			// fill the on demand price if appropriate
			if pr.OnDemandPrice > 0 {
				pdWithNtwPerfCat.OnDemandPrice = pr.OnDemandPrice
			}
			for zone, price := range pr.SpotPrice {
				pdWithNtwPerfCat.SpotInfo = append(pdWithNtwPerfCat.SpotInfo, *newZonePrice(zone, price))
			}
		} else {
			log.Debugf("price info not yet cached for key: %s", cpi.getPriceKey(cloud, region, vm.Type))
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
