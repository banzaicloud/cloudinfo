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

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/messaging"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/tracing"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
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
	var (
		err    error
		prices map[string]map[string]Price
	)
	ctx, _ = sm.tracer.StartWithTags(ctx, "initialize", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	sm.log.Info("initializing cloud product information")
	if prices, err = sm.infoer.Initialize(); err != nil {
		sm.log.Error(emperror.Wrap(err, "failed to initialize cloud product information").Error(),
			log.ToMap(emperror.Context(err)))
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
	var err error

	if service != "compute" {
		var (
			values []VmInfo
			vms    interface{}
			ok     bool
		)
		sm.log.Debug("retrieving regional product information", map[string]interface{}{"service": service, "region": regionId})
		if vms, ok = sm.store.GetVm(sm.provider, "compute", regionId); !ok {
			return emperror.With(errors.New("vms not yet cached"),
				"provider", sm.provider, "service", service, "region", regionId)
		}
		if values, err = sm.infoer.GetProducts(vms.([]VmInfo), service, regionId); err != nil {
			return emperror.Wrap(err, "failed to retrieve products for region")
		}

		for _, vm := range values {
			if vm.OnDemandPrice > 0 {
				metrics.OnDemandPriceGauge.WithLabelValues(sm.provider, regionId, vm.Type).Set(vm.OnDemandPrice)
			}
		}
		sm.store.StoreVm(sm.provider, service, regionId, values)
	}

	err = sm.updateVirtualMachines(service, regionId)
	if err != nil {
		return err
	}

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionImages(ctx context.Context, service string, regionId string) error {
	var (
		images []Image
		err    error
	)
	if sm.infoer.HasImages() {
		sm.log.Debug("retrieving regional image information",
			map[string]interface{}{"service": service, "region": regionId})
		if images, err = sm.infoer.GetServiceImages(service, regionId); err != nil {
			return emperror.Wrap(err, "failed to retrieve service images for region")
		}
		sm.store.StoreImage(sm.provider, service, regionId, images)
	}
	return nil
}

func (sm *scrapingManager) scrapeServiceRegionVersions(ctx context.Context, service string, regionId string) error {
	var (
		versions []LocationVersion
		err      error
	)

	sm.log.Debug("retrieving regional version information",
		map[string]interface{}{"service": service, "region": regionId})
	if versions, err = sm.infoer.GetVersions(service, regionId); err != nil {
		return emperror.Wrap(err, "failed to retrieve service versions for region")
	}
	sm.store.StoreVersion(sm.provider, service, regionId, versions)

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionZones(ctx context.Context, service, region string) error {
	var (
		zones []string
		err   error
	)

	sm.log.Debug("retrieving regional zone information", map[string]interface{}{"service": service, "region": region})

	if zones, err = sm.infoer.GetZones(region); err != nil {
		return emperror.Wrap(err, "failed to retrieve zones for region")
	}

	sm.store.StoreZones(sm.provider, service, region, zones)

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionVms(ctx context.Context, region string) error {
	var (
		vms []VmInfo
		err error
	)
	if vms, err = sm.infoer.GetVirtualMachines(region); err != nil {
		sm.metrics.ReportScrapeFailure(sm.provider, "compute", "N/A")
		return emperror.WrapWith(err, "failed to retrieve regions",
			"provider", sm.provider, "service", "compute")
	}
	sm.store.StoreVm(sm.provider, "compute", region, vms)

	return nil
}

func (sm *scrapingManager) scrapeServiceRegionInfo(ctx context.Context, services []Service) error {
	var (
		regions map[string]string
		err     error
	)
	ctx, _ = sm.tracer.StartWithTags(ctx, "scrape-region-info", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	if regions, err = sm.infoer.GetRegions("compute"); err != nil {
		sm.metrics.ReportScrapeFailure(sm.provider, "compute", "N/A")
		return emperror.WrapWith(err, "failed to retrieve regions",
			"provider", sm.provider, "service", "compute")
	}

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
			regions = _regions.(map[string]string)
		} else {
			if regions, err = sm.infoer.GetRegions(service.ServiceName()); err != nil {

				sm.metrics.ReportScrapeFailure(sm.provider, service.ServiceName(), "N/A")
				return emperror.WrapWith(err, "failed to retrieve regions",
					"provider", sm.provider, "service", service.ServiceName())
			}

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
	var (
		ok    bool
		zones interface{}
	)
	if service != "compute" {
		if zones, ok = sm.store.GetZones(sm.provider, "compute", region); !ok {
			return emperror.With(errors.New("zones not yet cached"),
				"provider", sm.provider, "service", service, "region", region)
		}

		sm.store.StoreZones(sm.provider, service, region, zones.([]string))
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
	var (
		err      error
		cached   interface{}
		services []Service
		ok       bool
	)
	ctx, _ = sm.tracer.StartWithTags(ctx, "scrape-service-info", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)

	if cached, ok = sm.store.GetServices(sm.provider); !ok {
		sm.metrics.ReportScrapeFailure(sm.provider, "N/A", "N/A")
		sm.log.Error(emperror.Wrap(err, "failed to retrieve services").Error(), log.ToMap(emperror.Context(err)))
	}

	if services, ok = cached.([]Service); !ok {
		sm.metrics.ReportScrapeFailure(sm.provider, "N/A", "N/A")
		sm.log.Error("invalid services stored in the store")
	}

	if err := sm.scrapeServiceRegionInfo(ctx, services); err != nil {
		sm.log.Error(emperror.Wrap(err, "failed to load service region information").Error(), log.ToMap(emperror.Context(err)))
	}

	sm.updateStatus(ctx)
}

func (sm *scrapingManager) scrapePricesInRegion(ctx context.Context, region string, wg *sync.WaitGroup) {
	var (
		err    error
		prices map[string]Price
	)
	if wg != nil {
		defer wg.Done()
	}
	start := time.Now()
	if prices, err = sm.infoer.GetCurrentPrices(region); err != nil {
		sm.metrics.ReportScrapeShortLivedFailure(sm.provider, region)
		sm.log.Error(emperror.Wrap(err, "failed to scrape spot prices in region").Error(), map[string]interface{}{"region": region})
	}

	for instType, price := range prices {
		sm.store.StorePrice(sm.provider, region, instType, price)
	}

	sm.metrics.ReportScrapeRegionShortLivedCompleted(sm.provider, region, start)
}

func (sm *scrapingManager) scrapePricesInAllRegions(ctx context.Context) {
	var (
		regions map[string]string
		err     error
		wg      sync.WaitGroup
	)

	ctx, _ = sm.tracer.StartWithTags(ctx, "scrape-region-prices", map[string]interface{}{"provider": sm.provider})
	defer sm.tracer.EndSpan(ctx)
	sm.log.Info("start scraping prices")

	// record current time for metrics
	start := time.Now()
	if regions, err = sm.infoer.GetRegions("compute"); err != nil {
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
	var (
		vms             interface{}
		prices          interface{}
		ok              bool
		pr              Price
		virtualMachines []VmInfo
	)

	if vms, ok = sm.store.GetVm(sm.provider, service, region); !ok {
		return emperror.With(errors.New("vms not yet cached"),
			"provider", sm.provider, "service", service, "region", region)
	}

	for _, vm := range vms.([]VmInfo) {
		if prices, ok = sm.store.GetPrice(sm.provider, region, vm.Type); ok {
			pr = prices.(Price)
			// fill the on demand price if appropriate
			if pr.OnDemandPrice > 0 {
				vm.OnDemandPrice = pr.OnDemandPrice
			}
		}

		if vm.OnDemandPrice != 0 {
			virtualMachines = append(virtualMachines, vm)
		}
	}

	sm.store.DeleteVm(sm.provider, service, region)

	sm.log.Debug("updated products information", map[string]interface{}{"numberOfVms": len(virtualMachines)})

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
		log:      logur.WithFields(log, map[string]interface{}{"provider": provider}),
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
		log:              log,
	}
}
