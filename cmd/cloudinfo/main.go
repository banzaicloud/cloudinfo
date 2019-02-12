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
	"fmt"
	"time"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/api"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/loader"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/management"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/tracing"
	"github.com/banzaicloud/cloudinfo/internal/platform/buildinfo"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/alibaba"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/amazon"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/azure"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/google"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/oracle"
	"github.com/banzaicloud/cloudinfo/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func main() {

	// read configuration (commandline, env etc)
	Configure(viper.GetViper(), pflag.CommandLine)

	// parse the command line
	pflag.Parse()

	if viper.GetBool(helpFlag) {
		pflag.Usage()
		return
	}

	err := viper.ReadInConfig()
	_, configFileNotFound := err.(viper.ConfigFileNotFoundError)
	if !configFileNotFound {
		emperror.Panic(errors.Wrap(err, "failed to read configuration"))
	}

	var config Config
	// configuration gets populated here - external configuration sources (flags, env vars) are processed into the instance
	err = viper.Unmarshal(&config)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	// Create logger (first thing after configuration loading)
	logur := log.NewLogger(config.Log)

	// Provide some basic context to all log lines
	logur = log.WithFields(logur, map[string]interface{}{"environment": config.Environment, "application": ServiceName})

	// inject the configured logger instance
	logger.Init(logur)

	ctx := logger.ToContext(context.Background(), logger.NewLogCtxBuilder().WithField("application", ServiceName).Build())

	logger.Extract(ctx).Info("initializing the application",
		map[string]interface{}{"version": Version, "commit_hash": CommitHash, "build_date": BuildDate})

	// default tracer
	tracer := tracing.NewNoOpTracer()

	// Configure Jaeger
	if config.Instrumentation.Jaeger.Enabled {
		logur.Info("jaeger exporter enabled", nil)
		tracing.SetupTracing(config.Instrumentation.Jaeger.Config, emperror.NewNoopHandler())
		// set the app tracer
		tracer = tracing.NewTracer()
	}

	cloudInfoStore := cloudinfo.NewCacheProductStore(24*time.Hour, config.RenewalInterval, logur)

	infoers := loadInfoers(ctx, config)

	reporter := metrics.NewDefaultMetricsReporter()

	serviceManager := loader.NewDefaultServiceManager(config.ServiceLoader, cloudInfoStore, logur)
	serviceManager.ConfigureServices(ctx, config.Providers)

	serviceManager.LoadServiceInformation(ctx, config.Providers)

	prodInfo, err := cloudinfo.NewCachingCloudInfo(cloudInfoStore, infoers, reporter, tracer, logur)
	emperror.Panic(err)

	scrapingDriver := cloudinfo.NewScrapingDriver(config.RenewalInterval, infoers, cloudInfoStore, logur, reporter, tracer)

	err = scrapingDriver.StartScraping(ctx)
	emperror.Panic(err)

	// start the management service
	if config.Management.Enabled {
		go management.StartManagementEngine(config.Management, cloudInfoStore, prodInfo, logur)
	}

	err = api.ConfigureValidator(ctx, config.Providers, prodInfo)
	emperror.Panic(err)

	buildInfo := buildinfo.New(Version, CommitHash, BuildDate)
	routeHandler := api.NewRouteHandler(prodInfo, buildInfo)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// add prometheus metric endpoint
	if config.Metrics.Enabled {
		routeHandler.EnableMetrics(ctx, router, config.Metrics.Address)
	}

	routeHandler.ConfigureRoutes(ctx, router)

	if err := router.Run(viper.GetString(listenAddressFlag)); err != nil {
		panic(fmt.Errorf("could not run router. error: %s", err))
	}
}

func loadInfoers(ctx context.Context, config Config) map[string]cloudinfo.CloudInfoer {

	infoers := make(map[string]cloudinfo.CloudInfoer, len(config.Providers))

	var (
		infoer cloudinfo.CloudInfoer
		err    error
	)

	for _, p := range config.Providers {
		pctx := logger.ToContext(ctx, logger.NewLogCtxBuilder().WithProvider(p).Build())

		switch p {
		case Amazon:
			infoer, err = amazon.NewAmazonInfoer(pctx, config.Amazon)
		case Google:
			infoer, err = google.NewGoogleInfoer(pctx, config.Google)
		case Azure:
			infoer, err = azure.NewAzureInfoer(pctx, config.Azure)
		case Oracle:
			infoer, err = oracle.NewOracleInfoer(pctx, config.Oracle)
		case Alibaba:
			infoer, err = alibaba.NewAliInfoer(pctx, config.Alibaba)
		default:
			logger.Extract(pctx).Error("provider is not supported")
		}

		emperror.Panic(err)

		infoers[p] = infoer
		logger.Extract(pctx).Info("configured product info provider", map[string]interface{}{"provider": p})
	}
	return infoers
}
