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
	"fmt"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/goph/emperror"
	"github.com/goph/logur"
	"github.com/pkg/errors"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/api"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/cistore"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/loader"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/management"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/messaging"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/tracing"
	cloudinfo2 "github.com/banzaicloud/cloudinfo/internal/cloudinfo"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/cloudinfodriver"
	"github.com/banzaicloud/cloudinfo/internal/platform/buildinfo"
	"github.com/banzaicloud/cloudinfo/internal/platform/errorhandler"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/alibaba"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/amazon"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/azure"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/google"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/oracle"
)

// nolint: gochecknoinits
func init() {
	pflag.Bool("version", false, "Show version information")
	pflag.Bool("dump-config", false, "Dump configuration to the console (and exit)")
}

func main() {
	Configure(viper.GetViper(), pflag.CommandLine)
	pflag.Usage = pflag.CommandLine.Usage

	pflag.Parse()

	if v, _ := pflag.CommandLine.GetBool("version"); v {
		fmt.Printf("%s version %s (%s) built on %s\n", friendlyServiceName, Version, CommitHash, BuildDate)

		os.Exit(0)
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
	logger := log.NewLogger(config.Log)

	// Provide some basic context to all log lines
	logger = log.WithFields(logger, map[string]interface{}{"environment": config.Environment, "application": serviceName})

	log.SetStandardLogger(logger)

	if configFileNotFound {
		logger.Warn("configuration file not found")
	}

	err = config.Validate()
	if err != nil {
		logger.Error(err.Error())

		os.Exit(3)
	}

	if d, _ := pflag.CommandLine.GetBool("dump-config"); d {
		fmt.Printf("%+v\n", config)

		os.Exit(0)
	}

	// Configure error handler
	errorHandler := errorhandler.New(logger)
	defer emperror.HandleRecover(errorHandler)

	buildInfo := buildinfo.New(Version, CommitHash, BuildDate)

	logger.Info("starting application", buildInfo.Fields())

	// default tracer
	tracer := tracing.NewNoOpTracer()

	// Configure Jaeger
	if config.Instrumentation.Jaeger.Enabled {
		logger.Info("jaeger exporter enabled")

		tracing.SetupTracing(config.Instrumentation.Jaeger.Config, emperror.NewNoopHandler())
		tracer = tracing.NewTracer()
	}

	// use the configured store implementation
	cloudInfoStore := cistore.NewCloudInfoStore(config.Store, logger)
	defer cloudInfoStore.Close()

	infoers := loadInfoers(config, logger)

	reporter := metrics.NewDefaultMetricsReporter()

	eventBus := messaging.NewDefaultEventBus()

	serviceManager := loader.NewDefaultServiceManager(config.ServiceLoader, cloudInfoStore, logger, eventBus)
	serviceManager.ConfigureServices(config.App.Providers)

	serviceManager.LoadServiceInformation(config.App.Providers)

	prodInfo, err := cloudinfo.NewCachingCloudInfo(infoers, cloudInfoStore, logger)
	emperror.Panic(err)

	scrapingDriver := cloudinfo.NewScrapingDriver(config.App.RenewalInterval, infoers, cloudInfoStore, logger, reporter, tracer, eventBus)

	err = scrapingDriver.StartScraping()
	emperror.Panic(err)

	// start the management service
	if config.Management.Enabled {
		go management.StartManagementEngine(config.Management, cloudInfoStore, *scrapingDriver, logger)
	}

	err = api.ConfigureValidator(config.App.Providers, prodInfo, logger)
	emperror.Panic(err)

	instanceTypeService := cloudinfo2.NewInstanceTypeService(prodInfo)
	endpoints := cloudinfodriver.MakeEndpoints(instanceTypeService)
	graphqlHandler := cloudinfodriver.MakeGraphQLHandler(endpoints, errorHandler)

	routeHandler := api.NewRouteHandler(prodInfo, buildInfo, graphqlHandler, logger)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// add prometheus metric endpoint
	if config.Instrumentation.Metrics.Enabled {
		routeHandler.EnableMetrics(router, config.Instrumentation.Metrics.Address)
	}

	routeHandler.ConfigureRoutes(router)

	err = router.Run(config.App.Address)
	emperror.Panic(errors.Wrap(err, "failed to run router"))
}

func loadInfoers(config Config, log logur.Logger) map[string]cloudinfo.CloudInfoer {
	infoers := make(map[string]cloudinfo.CloudInfoer, len(config.App.Providers))

	var (
		infoer cloudinfo.CloudInfoer
		err    error
	)

	for _, p := range config.App.Providers {
		log = logur.WithFields(log, map[string]interface{}{"provider": p})

		switch p {
		case Amazon:
			infoer, err = amazon.NewAmazonInfoer(config.Amazon, log)
		case Google:
			infoer, err = google.NewGoogleInfoer(config.Google, log)
		case Azure:
			infoer, err = azure.NewAzureInfoer(config.Azure, log)
		case Oracle:
			infoer, err = oracle.NewOracleInfoer(config.Oracle, log)
		case Alibaba:
			infoer, err = alibaba.NewAliInfoer(config.Alibaba, log)
		default:
			log.Error("provider is not supported")
		}

		emperror.Panic(err)

		infoers[p] = infoer
		log.Info("configured product info provider", map[string]interface{}{"provider": p})
	}

	return infoers
}
