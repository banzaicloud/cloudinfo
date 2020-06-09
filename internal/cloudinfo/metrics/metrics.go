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

package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ScrapeCompleteDurationGauge collects metrics for the prometheus
	scrapeCompleteDurationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scrape",
		Name:      "complete_duration_seconds",
		Help:      "Cloud provider scrape complete duration in seconds",
	},
		[]string{"provider"},
	)
	// ScrapeRegionDurationGauge collects metrics for the prometheus
	scrapeRegionDurationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scrape",
		Name:      "region_duration_seconds",
		Help:      "Cloud provider scrape region duration in seconds",
	},
		[]string{"provider", "service", "region"},
	)
	// ScrapeFailuresTotalCounter collects metrics for the prometheus
	scrapeFailuresTotalCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scrape",
		Name:      "failures_total",
		Help:      "Total number of scrape failures, partitioned by provider and region",
	},
		[]string{"provider", "service", "region"},
	)
	// ScrapeShortLivedCompleteDurationGauge collects metrics for the prometheus
	scrapeShortLivedCompleteDurationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scrape",
		Name:      "short_lived_complete_duration_seconds",
		Help:      "Cloud provider short lived scrape complete duration in seconds",
	},
		[]string{"provider"},
	)
	// ScrapeShortLivedRegionDurationGauge collects metrics for the prometheus
	scrapeShortLivedRegionDurationGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "scrape",
		Name:      "short_lived_region_duration_seconds",
		Help:      "Cloud provider short lived scrape region duration in seconds",
	},
		[]string{"provider", "region"},
	)
	// ScrapeShortLivedFailuresTotalCounter collects metrics for the prometheus
	scrapeShortLivedFailuresTotalCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "scrape",
		Name:      "short_lived_failures_total",
		Help:      "Total number of short lived scrape failures, partitioned by provider and region",
	},
		[]string{"provider", "region"},
	)
	// OnDemandPriceGauge collects metrics for the prometheus
	OnDemandPriceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "cloudinfo",
		Name:      "on_demand_price",
		Help:      "On demand price for each instance type",
	},
		[]string{"provider", "region", "instanceType"},
	)
)

// Defines application specific operations for collecting metrics
type Reporter interface {
	// ReportScrapeProviderCompleted registers the event of a successful scrape completion
	ReportScrapeProviderCompleted(provider string, startTime time.Time)

	// ReportScrapeRegionCompleted registers the completion of a successful scrape for the service, region and provider
	ReportScrapeRegionCompleted(provider, service, region string, startTime time.Time)

	// ReportScrapeFailure reports a scraping failure
	ReportScrapeFailure(provider, service, region string)

	// ReportScrapeProviderShortLivedCompleted reports a successful short lived information scrape for the provider
	ReportScrapeProviderShortLivedCompleted(provider string, startTime time.Time)

	// ReportScrapeRegionShortLivedCompleted reports a successful scrape for short lived information for the given region
	ReportScrapeRegionShortLivedCompleted(provider, region string, startTime time.Time)

	// ReportScrapeShortLivedFailure reports the failure of scraping short lived information
	ReportScrapeShortLivedFailure(provider, region string)
}

// DefaultMetricsReporter default metrics source for the application
type DefaultMetricsReporter struct {
	// Collectors holds application metric collector references for "bulk" operations
	Collectors []*prometheus.Collector
}

// registerCollectors registers collectors held by this metrics source
func (ms *DefaultMetricsReporter) registerCollectors() {
	for _, collector := range ms.Collectors {
		prometheus.MustRegister(*collector)
	}
}

func (ms *DefaultMetricsReporter) addCollector(c prometheus.Collector) {
	if ms.Collectors == nil {
		ms.Collectors = make([]*prometheus.Collector, 0)
	}
	ms.Collectors = append(ms.Collectors, &c)
}

func (ms *DefaultMetricsReporter) ReportScrapeProviderCompleted(provider string, startTime time.Time) {
	scrapeCompleteDurationGauge.WithLabelValues(provider).Set(time.Since(startTime).Seconds())
}

func (ms *DefaultMetricsReporter) ReportScrapeRegionCompleted(provider, service, region string, startTime time.Time) {
	scrapeRegionDurationGauge.WithLabelValues(provider, service, region).Set(time.Since(startTime).Seconds())
}

func (ms *DefaultMetricsReporter) ReportScrapeFailure(provider, service, region string) {
	scrapeFailuresTotalCounter.WithLabelValues(provider, service, region).Inc()
}

func (ms *DefaultMetricsReporter) ReportScrapeProviderShortLivedCompleted(provider string, startTime time.Time) {
	scrapeShortLivedCompleteDurationGauge.WithLabelValues(provider).Set(time.Since(startTime).Seconds())
}

func (ms *DefaultMetricsReporter) ReportScrapeRegionShortLivedCompleted(provider, region string, startTime time.Time) {
	scrapeShortLivedRegionDurationGauge.WithLabelValues(provider, region).Set(time.Since(startTime).Seconds())
}

func (ms *DefaultMetricsReporter) ReportScrapeShortLivedFailure(provider, region string) {
	scrapeShortLivedFailuresTotalCounter.WithLabelValues(provider, region).Inc()
}

// NewMetricsSource assembles a Reporter with custom collectors
func NewDefaultMetricsReporter() Reporter {
	dms := &DefaultMetricsReporter{}
	dms.addCollector(scrapeCompleteDurationGauge)
	dms.addCollector(scrapeRegionDurationGauge)
	dms.addCollector(scrapeFailuresTotalCounter)
	dms.addCollector(scrapeShortLivedCompleteDurationGauge)
	dms.addCollector(scrapeShortLivedRegionDurationGauge)
	dms.addCollector(scrapeShortLivedFailuresTotalCounter)

	dms.registerCollectors()

	return dms
}

func GetPriceGatherers() prometheus.Gatherers {
	reg := prometheus.NewRegistry()
	reg.MustRegister(OnDemandPriceGauge, googleSpotPriceGauge, azureSpotPriceGauge)
	return prometheus.Gatherers{reg}
}

func GetSpotPriceGatherers() prometheus.Gatherers {
	spotReg := prometheus.NewRegistry()
	spotReg.MustRegister(amazonSpotPriceGauge, alibabaSpotPriceGauge)
	return prometheus.Gatherers{spotReg}
}

// SpotPriceGauge collects metrics for the prometheus
var alibabaSpotPriceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "cloudinfo",
	Name:      "alibaba_spot_price",
	Help:      "spot price for each instance type",
},
	[]string{"region", "zone", "instanceType"},
)

func ReportAlibabaSpotPrice(region, zone, instanceType string, price float64) {
	alibabaSpotPriceGauge.WithLabelValues(region, zone, instanceType).Set(price)
}

// SpotPriceGauge collects metrics for the prometheus
var amazonSpotPriceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "cloudinfo",
	Name:      "amazon_spot_price",
	Help:      "spot price for each instance type",
},
	[]string{"region", "zone", "instanceType"},
)

func ReportAmazonSpotPrice(region, zone, instanceType string, price float64) {
	amazonSpotPriceGauge.WithLabelValues(region, zone, instanceType).Set(price)
}

// SpotPriceGauge collects metrics for the prometheus
var azureSpotPriceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "cloudinfo",
	Name:      "azure_spot_price",
	Help:      "spot price for each instance type",
},
	[]string{"region", "instanceType"},
)

func ReportAzureSpotPrice(region, instanceType string, price float64) {
	azureSpotPriceGauge.WithLabelValues(region, instanceType).Set(price)
}

// SpotPriceGauge collects metrics for the prometheus
var googleSpotPriceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "cloudinfo",
	Name:      "google_spot_price",
	Help:      "spot price for each instance type",
},
	[]string{"region", "zone", "instanceType"},
)

func ReportGoogleSpotPrice(region, zone, instanceType string, price float64) {
	googleSpotPriceGauge.WithLabelValues(region, zone, instanceType).Set(price)
}

type noOpReporter struct {
}

func (nor *noOpReporter) ReportScrapeProviderCompleted(provider string, startTime time.Time) {}

func (nor *noOpReporter) ReportScrapeRegionCompleted(provider, service, region string, startTime time.Time) {
}

func (nor *noOpReporter) ReportScrapeFailure(provider, service, region string) {}

func (nor *noOpReporter) ReportScrapeProviderShortLivedCompleted(provider string, startTime time.Time) {
}

func (nor *noOpReporter) ReportScrapeRegionShortLivedCompleted(provider, region string, startTime time.Time) {
}

func (nor *noOpReporter) ReportScrapeShortLivedFailure(provider, region string) {}

func NewNoOpMetricsReporter() Reporter {
	return &noOpReporter{}
}
