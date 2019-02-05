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

package cloudinfo

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/tracing"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/goph/emperror"
	"github.com/goph/logur"
)

// scrapingManager manages data renewal for a given provider
// retrieves data from the cloud provider and stores it in the store
type scrapingManager struct {
	provider string
	infoer   CloudInfoer
	store    CloudInfoStore
	metrics  metrics.Reporter
	tracer   tracing.Tracer
	log      logur.Logger
}

func (sm *scrapingManager) initialize(ctx context.Context) error {
	var (
		err    error
		prices map[string]map[string]Price
	)
	ctx, _ = sm.tracer.StartWithTags(ctx, "initialize", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	sm.log.Info("initializing cloud product information")
	if prices, err = sm.infoer.Initialize(ctx); err != nil {
		sm.log.Warn("failed to initialize cloud product information")
		return err
	}
	for region, ap := range prices {
		for instType, p := range ap {
			sm.store.StorePrice(sm.provider, region, instType, p)
			metrics.OnDemandPriceGauge.WithLabelValues(sm.provider, region, instType).Set(p.OnDemandPrice)
		}
	}
	sm.log.Info("finished initializing cloud product information")
	return nil
}

func (sm *scrapingManager) scrapeServiceAttributes(ctx context.Context, services []Service) error {
	var (
		err      error
		attrVals AttrValues
	)
	ctx, _ = sm.tracer.StartWithTags(ctx, "renew-attribute-values", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	sm.log.Info("start to renew attribute values")
	for _, service := range services {
		for _, attr := range []string{sm.infoer.GetCpuAttrName(), sm.infoer.GetMemoryAttrName()} {

			if attrVals, err = sm.infoer.GetAttributeValues(ctx, service.ServiceName(), attr); err != nil {
				sm.metrics.ReportScrapeFailure(sm.provider, "N/A", "N/A")
				sm.log.Error("failed to retrieve attribute values")
				// should the process go forward here?
				return emperror.WrapWith(err, "failed to retrieve attribute values", "attribute", attr)
			}
			sm.store.StoreAttribute(sm.provider, service.ServiceName(), attr, attrVals)
		}
	}
	return nil
}

func (sm *scrapingManager) scrapeServiceRegionProducts(ctx context.Context, service Service, regionId string) error {
	var (
		values []VmInfo
		err    error
	)
	sm.log.Debug("retrieving regional product information", map[string]interface{}{"region": regionId})
	if values, err = sm.infoer.GetProducts(ctx, service.ServiceName(), regionId); err != nil {

		sm.log.Error("failed to retrieve products for region", map[string]interface{}{"service": service.ServiceName(), "region": regionId})
		return emperror.WrapWith(err, "failed to retrieve products for region",
			"provider", sm.provider, "service", service.ServiceName(), "region", regionId)
	}

	for _, vm := range values {
		if vm.OnDemandPrice > 0 {
			metrics.OnDemandPriceGauge.WithLabelValues(sm.provider, regionId, vm.Type).Set(vm.OnDemandPrice)
		}
	}
	sm.store.StoreVm(sm.provider, service.ServiceName(), regionId, values)

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionImages(ctx context.Context, service Service, regionId string) error {
	var (
		images []Image
		err    error
	)
	if sm.infoer.HasImages() {
		sm.log.Debug("retrieving regional image information", map[string]interface{}{"region": regionId})
		if images, err = sm.infoer.GetServiceImages(service.ServiceName(), regionId); err != nil {

			sm.log.Warn("failed to retrieve service images for region",
				map[string]interface{}{"service": service.ServiceName(), "region": regionId})
			return emperror.WrapWith(err, "failed to retrieve service images for region",
				"provider", sm.provider, "service", service.ServiceName(), "region", regionId)
		}
		sm.store.StoreImage(sm.provider, service.ServiceName(), regionId, images)
	}
	return nil
}

func (sm *scrapingManager) scrapeServiceRegionVersions(ctx context.Context, service Service, regionId string) error {
	var (
		versions []string
		err      error
	)

	sm.log.Debug("retrieving regional version information", map[string]interface{}{"region": regionId})
	if versions, err = sm.infoer.GetVersions(ctx, service.ServiceName(), regionId); err != nil {
		sm.log.Warn("failed to retrieve service versions for region")
	}
	sm.store.StoreVersion(sm.provider, service.ServiceName(), regionId, versions)

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionInfo(ctx context.Context, services []Service) error {
	var (
		regions map[string]string
		err     error
	)
	ctx, _ = sm.tracer.StartWithTags(ctx, "scrape-region-info", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	sm.log.Info("start to scrape service region information")
	for _, service := range services {
		if regions, err = sm.infoer.GetRegions(ctx, service.ServiceName()); err != nil {

			sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), "N/A")
			sm.log.Error("failed to retrieve regions")
			return emperror.WrapWith(err, "failed to retrieve regions",
				"provider", sm.provider, "service", service.ServiceName())
		}

		for regionId := range regions {

			if err = sm.scrapeServiceRegionProducts(ctx, service, regionId); err != nil {
				sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), regionId)
			}

			if err = sm.scrapeServiceRegionImages(ctx, service, regionId); err != nil {
				sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), regionId)
			}

			if err = sm.scrapeServiceRegionVersions(ctx, service, regionId); err != nil {
				sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), regionId)
			}
		}
	}
	return nil
}

func (sm *scrapingManager) updateStatus(ctx context.Context) {
	values := strconv.Itoa(int(time.Now().UnixNano() / 1e6))
	sm.log.Info("updating status for provider")
	sm.store.StoreStatus(sm.provider, values)
}

// scrapeServiceInformation scrapes service and region dependant cloud information and stores its
func (sm *scrapingManager) scrapeServiceInformation(ctx context.Context) error {
	var (
		err      error
		services []Service
	)
	ctx, _ = sm.tracer.StartWithTags(ctx, "scrape-service-info", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	if services, err = sm.infoer.GetServices(); err != nil {
		sm.metrics.ReportScrapeFailure(sm.provider, "N/A", "N/A")
		sm.log.Error("failed to renew products")
		return emperror.Wrap(err, "failed to retrieve services")
	}

	if err := sm.scrapeServiceAttributes(ctx, services); err != nil {
		sm.log.Error("failed to load service attribute values")
		return emperror.Wrap(err, "failed to load service attribute values")
	}

	if err := sm.scrapeServiceRegionInfo(ctx, services); err != nil {
		sm.log.Error("failed to load service region information")
		return emperror.Wrap(err, "failed to load service region information")
	}

	sm.updateStatus(ctx)

	return nil
}

func (sm *scrapingManager) scrapePricesInRegion(ctx context.Context, region string, wg *sync.WaitGroup) error {
	var (
		err    error
		prices map[string]Price
	)
	if wg != nil {
		defer wg.Done()
	}
	start := time.Now()
	if prices, err = sm.infoer.GetCurrentPrices(ctx, region); err != nil {
		sm.metrics.ReportScrapeShortLivedFailure(sm.provider, region)
		sm.log.Error("failed to scrape prices in region")
		return emperror.With(err, "failed to crape prices in region", "provider", sm.provider, "region", region)
	}

	for instType, price := range prices {
		sm.store.StorePrice(sm.provider, region, instType, price)
	}

	sm.metrics.ReportScrapeRegionShortLivedCompleted(sm.provider, region, start)
	return nil
}

func (sm *scrapingManager) scrapePricesInAllRegions(ctx context.Context) error {
	var (
		regions map[string]string
		err     error
		wg      sync.WaitGroup
	)

	ctx, _ = sm.tracer.StartWithTags(ctx, "scrape-region-prices", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	// record current time for metrics
	start := time.Now()
	if regions, err = sm.infoer.GetRegions(ctx, "compute"); err != nil {
		sm.log.Error("failed to retrieve regions")
		return emperror.WrapWith(err, "failed to retrieve regions", "provider", sm.provider, "service", "compute")
	}

	for regionId := range regions {
		wg.Add(1)
		go sm.scrapePricesInRegion(ctx, regionId, &wg)
	}
	wg.Wait()
	sm.metrics.ReportScrapeProviderShortLivedCompleted(sm.provider, start)
	return nil
}

// scrape implements the scraping logic for a provider
func (sm *scrapingManager) scrape(ctx context.Context) error {
	ctx, _ = sm.tracer.StartWithTags(ctx, fmt.Sprintf("scraping-%s", sm.provider), map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	if err := sm.initialize(ctx); err != nil {
		return err
	}
	if err := sm.scrapeServiceInformation(ctx); err != nil {
		return err
	}
	return nil
}

func NewScrapingManager(provider string, infoer CloudInfoer, store CloudInfoStore, log logur.Logger, metrics metrics.Reporter, tracer tracing.Tracer) *scrapingManager {

	return &scrapingManager{
		provider: provider,
		infoer:   infoer,
		store:    store,
		log:      logur.WithFields(log, map[string]interface{}{"provider": provider}),
		metrics:  metrics,
		tracer:   tracer,
	}
}

type ScrapingDriver struct {
	scrapingManagers []*scrapingManager
	renewalInterval  time.Duration
	log              logur.Logger
}

func (sd *ScrapingDriver) StartScraping(ctx context.Context) error {

	if err := NewPeriodicExecutor(sd.renewalInterval).Execute(ctx, sd.renewAll); err != nil {
		sd.log.Error("failed to scrape for vm information")
		return emperror.Wrap(err, "failed to scrape cloud information")
	}

	// start scraping providers for pricing information
	if err := NewPeriodicExecutor(4*time.Minute).Execute(ctx, sd.renewShortLived); err != nil {
		logger.Extract(ctx).Info("failed to scrape for pricing information")
		return nil
	}

	return nil
}

func (sd *ScrapingDriver) renewAll(ctx context.Context) {
	for _, manager := range sd.scrapingManagers {
		go manager.scrape(ctx)
	}
}

func (sd *ScrapingDriver) renewShortLived(ctx context.Context) {

	for _, manager := range sd.scrapingManagers {
		if !manager.infoer.HasShortLivedPriceInfo() {
			// the manager's logger is used here - that has the provider in it's context
			manager.log.Debug("skip scraping for short lived prices (not applicable for provider)")
			return
		}
		go manager.scrapePricesInAllRegions(ctx)
	}
}

func NewScrapingDriver(renewalInterval time.Duration, infoers map[string]CloudInfoer,
	store CloudInfoStore, log logur.Logger, metrics metrics.Reporter, tracer tracing.Tracer) *ScrapingDriver {
	var managers []*scrapingManager

	for provider, infoer := range infoers {
		managers = append(managers, NewScrapingManager(provider, infoer, store, log, metrics, tracer))
	}

	return &ScrapingDriver{
		managers,
		renewalInterval,
		log,
	}
}
