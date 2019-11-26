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

	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/messaging"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/tracing"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
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
	eventBus messaging.EventBus
}

func (sm *scrapingManager) initialize(ctx context.Context) {
	ctx, _ = sm.tracer.StartWithTags(ctx, "initialize", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	sm.log.Info("initializing cloud product information")
	prices, err := sm.infoer.Initialize()
	if err != nil {
		sm.log.Error(emperror.Wrap(err, "failed to initialize cloud product information").Error(),
			log.ToMap(emperror.Context(err)))
		return
	}

	for region, ap := range prices {
		for instType, p := range ap {
			sm.store.StorePrice(sm.provider, region, instType, p)
			metrics.OnDemandPriceGauge.WithLabelValues(sm.provider, region, instType).Set(p.OnDemandPrice)
		}
	}
	sm.log.Info("finished initializing cloud product information")
}

func (sm *scrapingManager) scrapeServiceRegionProducts(ctx context.Context, service string, regionId string) error {
	logger := log.WithFields(sm.log, map[string]interface{}{"service": service, "region": regionId})

	if service != "compute" {
		sm.log.Debug("retrieving regional product information", map[string]interface{}{"service": service, "region": regionId})
		vms, ok := sm.store.GetVm(sm.provider, service, regionId)
		if !ok {
			logger.Warn("vms not yet cached")
		}

		values, err := sm.infoer.GetProducts(vms, service, regionId)
		if err != nil {
			return emperror.Wrap(err, "failed to retrieve products for region")
		}

		for _, vm := range values {
			if vm.OnDemandPrice > 0 {
				metrics.OnDemandPriceGauge.WithLabelValues(sm.provider, regionId, vm.Type).Set(vm.OnDemandPrice)
			}
		}
		sm.store.StoreVm(sm.provider, service, regionId, values)
	}

	err := sm.updateVirtualMachines(service, regionId)
	if err != nil {
		return err
	}

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionImages(ctx context.Context, service string, regionId string) error {
	if sm.infoer.HasImages() {
		sm.log.Debug("retrieving regional image information",
			map[string]interface{}{"service": service, "region": regionId})
		images, err := sm.infoer.GetServiceImages(service, regionId)
		if err != nil {
			return emperror.Wrap(err, "failed to retrieve service images for region")
		}

		sm.store.DeleteImage(sm.provider, service, regionId)
		sm.store.StoreImage(sm.provider, service, regionId, images)
	}
	return nil
}

func (sm *scrapingManager) scrapeServiceRegionVersions(ctx context.Context, service string, regionId string) error {

	versions, err := sm.infoer.GetVersions(service, regionId)
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve service versions for region")
	}

	sm.store.DeleteVersion(sm.provider, service, regionId)
	sm.store.StoreVersion(sm.provider, service, regionId, versions)

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionZones(ctx context.Context, service, region string) error {

	zones, err := sm.infoer.GetZones(region)
	if err != nil {
		return emperror.Wrap(err, "failed to retrieve zones for region")
	}

	sm.store.DeleteZones(sm.provider, service, region)
	sm.store.StoreZones(sm.provider, service, region, zones)

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionVms(ctx context.Context, region string) error {
	vms, err := sm.infoer.GetVirtualMachines(region)
	if err != nil {
		sm.metrics.ReportScrapeFailure(sm.provider, "compute", "N/A")
		return emperror.WrapWith(err, "failed to retrieve regions",
			"provider", sm.provider, "service", "compute")
	}
	sm.store.StoreVm(sm.provider, "compute", region, vms)

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionInfo(ctx context.Context, services []Service) error {
	ctx, _ = sm.tracer.StartWithTags(ctx, "scrape-region-info", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	regions, err := sm.infoer.GetRegions("compute")
	if err != nil {
		sm.metrics.ReportScrapeFailure(sm.provider, "compute", "N/A")
		return emperror.WrapWith(err, "failed to retrieve regions",
			"provider", sm.provider, "service", "compute")
	}

	sm.store.DeleteRegions(sm.provider, "compute")

	sm.store.StoreRegions(sm.provider, "compute", regions)

	for regionId := range regions {
		if err = sm.scrapeServiceRegionVms(ctx, regionId); err != nil {
			sm.metrics.ReportScrapeFailure(sm.provider, "compute", regionId)
			return emperror.With(err, "provider", sm.provider, "service", "compute", "region", regionId)
		}

		if err = sm.scrapeServiceRegionZones(ctx, "compute", regionId); err != nil {
			sm.metrics.ReportScrapeFailure(sm.provider, "compute", regionId)
			return emperror.With(err, "provider", sm.provider, "service", "compute", "region", regionId)
		}
	}

	sm.log.Info("start to scrape service region information")
	for _, service := range services {

		if service.IsStatic {
			sm.log.Info("skipping scraping for region information - service is static", map[string]interface{}{"service": service.ServiceName()})
			continue
		}

		if service.ServiceName() == "compute" {
			_regions, ok := sm.store.GetRegions(sm.provider, service.ServiceName())
			if !ok {
				sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), "N/A")
				return emperror.WrapWith(err, "failed to retrieve regions",
					"provider", sm.provider, "service", service.ServiceName())
			}
			regions = _regions
		} else {
			if regions, err = sm.infoer.GetRegions(service.ServiceName()); err != nil {

				sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), "N/A")
				return emperror.WrapWith(err, "failed to retrieve regions",
					"provider", sm.provider, "service", service.ServiceName())
			}
			sm.store.DeleteRegions(sm.provider, service.ServiceName())

			sm.store.StoreRegions(sm.provider, service.ServiceName(), regions)
		}

		for regionId := range regions {

			start := time.Now()
			if err = sm.updateServiceRegionZones(ctx, service.ServiceName(), regionId); err != nil {
				return emperror.With(err, "provider", sm.provider, "service", service.ServiceName(), "region", regionId)
			}
			if err = sm.scrapeServiceRegionProducts(ctx, service.ServiceName(), regionId); err != nil {
				sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), regionId)
				return emperror.With(err, "provider", sm.provider, "service", service.ServiceName(), "region", regionId)
			}
			if err = sm.scrapeServiceRegionImages(ctx, service.ServiceName(), regionId); err != nil {
				sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), regionId)
				return emperror.With(err, "provider", sm.provider, "service", service.ServiceName(), "region", regionId)
			}
			if err = sm.scrapeServiceRegionVersions(ctx, service.ServiceName(), regionId); err != nil {
				sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), regionId)
				return emperror.With(err, "provider", sm.provider, "service", service.ServiceName(), "region", regionId)
			}
			sm.metrics.ReportScrapeRegionCompleted(sm.provider, service.ServiceName(), regionId, start)
		}
	}
	return nil
}

func (sm *scrapingManager) updateServiceRegionZones(ctx context.Context, service string, region string) error {
	if service != "compute" {
		zones, ok := sm.store.GetZones(sm.provider, "compute", region)
		if !ok {
			return emperror.With(errors.New("zones not yet cached"),
				"provider", sm.provider, "service", service, "region", region)
		}

		sm.store.DeleteZones(sm.provider, service, region)
		sm.store.StoreZones(sm.provider, service, region, zones)
	}

	return nil
}

func (sm *scrapingManager) updateStatus(ctx context.Context) {
	values := strconv.Itoa(int(time.Now().UnixNano() / 1e6))
	sm.log.Info("updating status for provider")
	sm.store.StoreStatus(sm.provider, values)
}

// scrapeServiceInformation scrapes service and region dependant cloud information and stores its
func (sm *scrapingManager) scrapeServiceInformation(ctx context.Context) {
	ctx, _ = sm.tracer.StartWithTags(ctx, "scrape-service-info", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	storedServices, ok := sm.store.GetServices(sm.provider)
	if !ok {
		sm.metrics.ReportScrapeFailure(sm.provider, "N/A", "N/A")
		sm.log.Error("failed to retrieve services")
		return
	}

	err := sm.scrapeServiceRegionInfo(ctx, storedServices)
	if err != nil {
		sm.log.Error(emperror.Wrap(err, "failed to load service region information").Error(), log.ToMap(emperror.Context(err)))
		return
	}

	sm.updateStatus(ctx)
}

func (sm *scrapingManager) scrapePricesInRegion(ctx context.Context, region string, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	start := time.Now()
	prices, err := sm.infoer.GetCurrentPrices(region)
	if err != nil {
		sm.metrics.ReportScrapeShortLivedFailure(sm.provider, region)
		sm.log.Error(emperror.Wrap(err, "failed to scrape spot prices in region").Error(), map[string]interface{}{"region": region})
	}

	for instType, price := range prices {
		sm.store.StorePrice(sm.provider, region, instType, price)
	}

	sm.metrics.ReportScrapeRegionShortLivedCompleted(sm.provider, region, start)
}

func (sm *scrapingManager) scrapePricesInAllRegions(ctx context.Context) {
	var wg sync.WaitGroup

	ctx, _ = sm.tracer.StartWithTags(ctx, "scrape-region-prices", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)
	sm.log.Info("start scraping prices")

	// record current time for metrics
	start := time.Now()
	regions, err := sm.infoer.GetRegions("compute")
	if err != nil {
		sm.log.Error(emperror.Wrap(err, "failed to retrieve regions").Error(), map[string]interface{}{"service": "compute"})
	}

	for regionId := range regions {
		wg.Add(1)
		go sm.scrapePricesInRegion(ctx, regionId, &wg)
	}
	wg.Wait()
	sm.metrics.ReportScrapeProviderShortLivedCompleted(sm.provider, start)
}

func (sm *scrapingManager) updateVirtualMachines(service, region string) error {
	vms, ok := sm.store.GetVm(sm.provider, service, region)
	if !ok {
		return emperror.With(errors.New("vms not yet cached"),
			"provider", sm.provider, "service", service, "region", region)
	}

	virtualMachines := make([]VmInfo, 0, len(vms))
	for _, vm := range vms {
		prices, found := sm.store.GetPrice(sm.provider, region, vm.Type)

		if found {
			if prices.OnDemandPrice > 0 {
				vm.OnDemandPrice = prices.OnDemandPrice
			}
		}

		if vm.OnDemandPrice != 0 {
			virtualMachines = append(virtualMachines, vm)
		}
	}

	sm.store.DeleteVm(sm.provider, service, region)
	sm.store.StoreVm(sm.provider, service, region, virtualMachines)

	return nil
}

// scrape implements the scraping logic for a provider
func (sm *scrapingManager) scrape(ctx context.Context) {
	ctx, _ = sm.tracer.StartWithTags(ctx, fmt.Sprintf("scraping-%s", sm.provider), map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	sm.log.Info("start scraping for provider information")
	start := time.Now()

	sm.initialize(ctx)

	sm.scrapeServiceInformation(ctx)

	// emit a scraping complete event to notify potential subscribers
	sm.eventBus.PublishScrapingComplete(sm.provider)

	sm.metrics.ReportScrapeProviderCompleted(sm.provider, start)
}

func NewScrapingManager(provider string, infoer CloudInfoer, store CloudInfoStore, log logur.Logger,
	metrics metrics.Reporter, tracer tracing.Tracer, eventBus messaging.EventBus) *scrapingManager {

	return &scrapingManager{
		provider: provider,
		infoer:   infoer,
		store:    store,
		log:      logur.WithFields(log, map[string]interface{}{"component": "scraping-manager", "provider": provider}),
		metrics:  metrics,
		tracer:   tracer,
		eventBus: eventBus,
	}
}

type ScrapingDriver struct {
	scrapingManagers []*scrapingManager
	renewalInterval  time.Duration
	log              logur.Logger
}

func (sd *ScrapingDriver) StartScraping() error {

	ctx := context.Background()

	if err := NewPeriodicExecutor(sd.renewalInterval, sd.log).Execute(ctx, sd.renewAll); err != nil {
		return emperror.Wrap(err, "failed to scrape cloud information")
	}

	// start scraping providers for pricing information
	if err := NewPeriodicExecutor(4*time.Minute, sd.log).Execute(ctx, sd.renewShortLived); err != nil {
		return emperror.Wrap(err, "failed to scrape spot price info")
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
			continue
		}
		go manager.scrapePricesInAllRegions(ctx)
	}
}

func (sd *ScrapingDriver) RefreshProvider(ctx context.Context, provider string) {
	for _, manager := range sd.scrapingManagers {
		if manager.provider == provider {
			manager.scrape(ctx)
		}
	}
}

func NewScrapingDriver(renewalInterval time.Duration, infoers map[string]CloudInfoer,
	store CloudInfoStore, log logur.Logger, metrics metrics.Reporter, tracer tracing.Tracer, eventBus messaging.EventBus) *ScrapingDriver {
	var managers []*scrapingManager

	for provider, infoer := range infoers {
		managers = append(managers, NewScrapingManager(provider, infoer, store, log, metrics, tracer, eventBus))
	}

	return &ScrapingDriver{
		scrapingManagers: managers,
		renewalInterval:  renewalInterval,
		log:              logur.WithFields(log, map[string]interface{}{"component": "scraping-driver"}),
	}
}
