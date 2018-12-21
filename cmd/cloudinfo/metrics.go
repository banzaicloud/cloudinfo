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

package main

import "github.com/prometheus/client_golang/prometheus"

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
	// OnDemandPriceGauge collects metrics for the prometheus
	OnDemandPriceGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "cloudinfo",
		Name:      "on_demand_price",
		Help:      "On demand price for each instance type",
	},
		[]string{"provider", "region", "instanceType"},
	)
)
