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

// Package main Product Info.
//
// The product info application uses the cloud provider APIs to asynchronously fetch and parse instance type attributes
// and prices, while storing the results in an in memory cache and making it available as structured data through a REST API.
//
//     Schemes: http, https
//     BasePath: /api/v1
//     Version: 0.0.1
//     License: Apache 2.0 http://www.apache.org/licenses/LICENSE-2.0.html
//     Contact: Banzai Cloud<info@banzaicloud.com>
//
// swagger:meta
package main

import (
	"context"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/api"
	"github.com/banzaicloud/cloudinfo/internal/platform/buildinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/alibaba"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/amazon"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/azure"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/google"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/oracle"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/banzaicloud/go-gin-prometheus"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// the list of flags supported by the application
	// these constants can be used to retrieve the passed in values or defaults via viper
	logLevelFlag                = "log-level"
	logFormatFlag               = "log-format"
	listenAddressFlag           = "listen-address"
	cloudInfRenewalIntervalFlag = "cloud-info-renewal-interval"
	prometheusAddressFlag       = "prometheus-address"
	prometheusQueryFlag         = "prometheus-query"
	providerFlag                = "provider"
	helpFlag                    = "help"
	metricsEnabledFlag          = "metrics-enabled"
	metricsAddressFlag          = "metrics-address"

	//temporary flags
	gceApiKeyFlag          = "gce-api-key"
	gceApplicationCred     = "google-application-credentials"
	azureAuthLocation      = "azure-auth-location"
	alibabaRegionId        = "alibaba-region-id"
	alibabaAccessKeyId     = "alibaba-access-key-id"
	alibabaAccessKeySecret = "alibaba-access-key-secret"
	alibabaPriceInfoUrl    = "alibaba-price-info-url"
	oracleConfigLocation   = "oracle-cli-config-location"

	// Google is the identifier of the Google Cloud Engine provider
	Google = "google"
	// Amazon is the identifier of the Amazon provider
	Amazon = "amazon"
	// Azure is the identifier of the MS Azure provider
	Azure = "azure"
	// Oracle is the identifier of the Oracle Cloud Infrastructure provider
	Oracle = "oracle"
	// Alibaba is the identifier of the Alibaba Cloud provider
	Alibaba = "alibaba"
)

// defineFlags defines supported flags and makes them available for viper
func defineFlags() {
	flag.String(logLevelFlag, "info", "log level")
	flag.String(logFormatFlag, "", "log format")
	flag.String(listenAddressFlag, ":9090", "the address the cloudinfo app listens to HTTP requests.")
	flag.Duration(cloudInfRenewalIntervalFlag, 24*time.Hour, "duration (in go syntax) between renewing the product information. Example: 2h30m")
	flag.String(prometheusAddressFlag, "", "http address of a Prometheus instance that has AWS spot "+
		"price metrics via banzaicloud/spot-price-exporter. If empty, the cloudinfo app will use current spot prices queried directly from the AWS API.")
	flag.String(prometheusQueryFlag, "avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])",
		"advanced configuration: change the query used to query spot price info from Prometheus.")
	flag.String(gceApiKeyFlag, "", "GCE API key to use for getting SKUs")
	flag.String(gceApplicationCred, "", "google application credentials location")
	flag.StringSlice(providerFlag, []string{Amazon, Google, Azure, Oracle, Alibaba}, "Providers that will be used with the cloudinfo application.")
	flag.Bool(helpFlag, false, "print usage")
	flag.Bool(metricsEnabledFlag, false, "internal metrics are exposed if enabled")
	flag.String(metricsAddressFlag, ":9900", "the address where internal metrics are exposed")
	flag.String(azureAuthLocation, "", "azure authentication file location")
	flag.String(alibabaRegionId, "", "alibaba region id")
	flag.String(alibabaAccessKeyId, "", "alibaba access key id")
	flag.String(alibabaAccessKeySecret, "", "alibaba access key secret")
	flag.String(oracleConfigLocation, "", "oracle config file location")
	flag.String(alibabaPriceInfoUrl, "https://g.alicdn.com/aliyun/ecs-price-info-intl/2.0.8/price/download/instancePrice.json", "Alibaba get price info from this file")
}

// bindFlags binds parsed flags into viper
func bindFlags() {
	flag.Parse()
	if err := viper.BindPFlags(flag.CommandLine); err != nil {
		panic(errors.Wrap(err, "failed to parse flags"))
	}

}

func init() {

	// describe the flags for the application
	defineFlags()

	// all the flags should be referenced through viper after this call
	// flags are available through the entire application via viper
	bindFlags()

	// Viper check for an environment variable
	viper.AutomaticEnv()
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	// initialize the logging framework
	logger.InitLogger(viper.GetString(logLevelFlag), viper.GetString(logFormatFlag))

	// register prometheus custom metrics
	prometheus.MustRegister(cloudinfo.ScrapeCompleteDurationGauge)
	prometheus.MustRegister(cloudinfo.ScrapeRegionDurationGauge)
	prometheus.MustRegister(cloudinfo.ScrapeFailuresTotalCounter)
	prometheus.MustRegister(cloudinfo.ScrapeShortLivedCompleteDurationGauge)
	prometheus.MustRegister(cloudinfo.ScrapeShortLivedRegionDurationGauge)
	prometheus.MustRegister(cloudinfo.ScrapeShortLivedFailuresTotalCounter)
}

func main() {

	if viper.GetBool(helpFlag) {
		flag.Usage()
		return
	}

	ctx := logger.ToContext(context.Background(), logger.NewLogCtxBuilder().WithField("application", "cloudinfo").Build())

	logger.Extract(ctx).WithField("version", Version).WithField("commit_hash", CommitHash).WithField("build_date", BuildDate).Info("cloudinfo initialization")

	cloudInfo, err := cloudinfo.NewCachingCloudInfo(viper.GetDuration(cloudInfRenewalIntervalFlag),
		cache.New(cache.NoExpiration, 24.*time.Hour), infoers(ctx))
	quitOnError(ctx, "error encountered", err)

	go cloudInfo.Start(ctx)

	quitOnError(ctx, "error encountered", err)

	// configure the gin validator
	err = api.ConfigureValidator(ctx, viper.GetStringSlice(providerFlag), cloudInfo)
	quitOnError(ctx, "error encountered", err)

	buildInfo := buildinfo.New(Version, CommitHash, BuildDate)
	routeHandler := api.NewRouteHandler(cloudInfo, buildInfo)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// add prometheus metric endpoint
	if viper.GetBool(metricsEnabledFlag) {
		reg := prometheus.NewRegistry()
		reg.MustRegister(cloudinfo.OnDemandPriceGauge, google.SpotPriceGauge, azure.SpotPriceGauge)
		spotReg := prometheus.NewRegistry()
		spotReg.MustRegister(amazon.SpotPriceGauge, alibaba.SpotPriceGauge)
		p := ginprometheus.NewPrometheus("http", []string{"provider", "service", "region"})
		p.SetListenAddress(viper.GetString(metricsAddressFlag))
		p.Use(router, "/metrics")
		p.UseWithCustomMetrics(router, prometheus.Gatherers{reg}, "/metrics/price")
		p.UseWithCustomMetrics(router, prometheus.Gatherers{spotReg}, "/metrics/spotprice")
	}

	logger.Extract(ctx).Info("Initialized gin router")
	routeHandler.ConfigureRoutes(ctx, router)
	logger.Extract(ctx).Info("Configured routes")
	if err := router.Run(viper.GetString(listenAddressFlag)); err != nil {
		panic(errors.Wrap(err, "failed to run router"))
	}
}

func infoers(ctx context.Context) map[string]cloudinfo.CloudInfoer {
	providers := viper.GetStringSlice(providerFlag)
	infoers := make(map[string]cloudinfo.CloudInfoer, len(providers))
	for _, p := range providers {
		var infoer cloudinfo.CloudInfoer
		var err error
		pctx := logger.ToContext(ctx, logger.NewLogCtxBuilder().WithProvider(p).Build())

		switch p {
		case Amazon:
			infoer, err = amazon.NewEc2Infoer(pctx, viper.GetString(prometheusAddressFlag), viper.GetString(prometheusQueryFlag))
		case Google:
			infoer, err = google.NewGceInfoer(viper.GetString(gceApplicationCred), viper.GetString(gceApiKeyFlag))
		case Azure:
			infoer, err = azure.NewAzureInfoer(viper.GetString(azureAuthLocation))
		case Oracle:
			infoer, err = oracle.NewInfoer(viper.GetString(oracleConfigLocation))
		case Alibaba:
			infoer, err = alibaba.NewAlibabaInfoer(viper.GetString(alibabaRegionId), viper.GetString(alibabaAccessKeyId), viper.GetString(alibabaAccessKeySecret))
		default:
			logger.Extract(pctx).Fatal("provider is not supported")
		}

		quitOnError(pctx, "could not initialize cloud info provider", err)

		infoers[p] = infoer
		logger.Extract(pctx).Infof("Configured '%s' cloud info provider", p)
	}
	return infoers
}

func quitOnError(ctx context.Context, msg string, err error) {
	if err != nil {
		logger.Extract(ctx).WithError(err).Error(msg)
		flag.Usage()
		os.Exit(-1)
	}
}
