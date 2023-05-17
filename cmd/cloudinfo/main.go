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
//	Schemes: http, https
//	BasePath: /api/v1
//	Version: 0.0.1
//	License: Apache 2.0 http://www.apache.org/licenses/LICENSE-2.0.html
//	Contact: Banzai Cloud<info@banzaicloud.com>
//
// swagger:meta
package main

import (
	"fmt"
	"net/url"
	"os"

	"emperror.dev/emperror"
	"emperror.dev/errors"
	"github.com/gin-gonic/gin"
	vaultremote "github.com/sagikazarmark/viperx/remote"
	_ "github.com/sagikazarmark/viperx/remote/bankvaults"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/api"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/cistore"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/loader"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/management"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/messaging"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/tracing"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/cloudinfoadapter"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/cloudinfodriver"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/metrics"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/providers/alibaba"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/providers/amazon"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/providers/azure"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/providers/digitalocean"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/providers/google"
	"github.com/banzaicloud/cloudinfo/internal/cloudinfo/providers/oracle"
	"github.com/banzaicloud/cloudinfo/internal/platform/buildinfo"
	"github.com/banzaicloud/cloudinfo/internal/platform/errorhandler"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/internal/platform/profiler"
)

// Provisioned by ldflags
// nolint: gochecknoglobals
var (
	version    string
	commitHash string
	buildDate  string
	branch     string
)

func main() {
	v, p := viper.New(), pflag.NewFlagSet(friendlyAppName, pflag.ExitOnError)
	configure(v, p)

	p.String("config", "", "Configuration file")
	p.Bool("version", false, "Show version information")
	p.Bool("dump-config", false, "Dump configuration to the console (and exit)")

	_ = p.Parse(os.Args[1:])

	if v, _ := p.GetBool("version"); v {
		fmt.Printf("%s version %s (%s) built on %s at %s\n", friendlyAppName, version, commitHash, branch, buildDate)

		os.Exit(0)
	}

	if c, _ := p.GetString("config"); c != "" {
		v.SetConfigFile(c)
	}

	err := v.ReadInConfig()
	_, configFileNotFound := err.(viper.ConfigFileNotFoundError)
	if !configFileNotFound {
		emperror.Panic(errors.Wrap(err, "failed to read configuration"))
	}

	var metaConfig metaConfiguration
	err = v.UnmarshalKey("config", &metaConfig)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal meta configuration"))

	err = metaConfig.Validate()
	emperror.Panic(err)

	if metaConfig.Vault.Enabled {
		vaultremote.SetErrorHandler(errorhandler.NewPanicHandler())

		u, _ := url.Parse(metaConfig.Vault.Address)
		q := u.Query()
		q.Set("token", metaConfig.Vault.Token)
		u.RawQuery = q.Encode()

		err = v.AddRemoteProvider("bankvaults", u.String(), metaConfig.Vault.SecretPath)
		emperror.Panic(errors.Wrap(err, "failed to add vault config provider"))

		v.SetConfigType("json")
		err = v.ReadRemoteConfig()
		emperror.Panic(errors.Wrap(err, "failed to read remote configuration"))
	}

	var config configuration
	err = v.Unmarshal(&config)
	emperror.Panic(errors.Wrap(err, "failed to unmarshal configuration"))

	// Create logger (first thing after configuration loading)
	logger := log.NewLogger(config.Log)

	// Provide some basic context to all log lines
	logger = log.WithFields(logger, map[string]interface{}{"environment": config.Environment, "application": appName})

	log.SetStandardLogger(logger)

	if configFileNotFound {
		logger.Warn("configuration file not found")
	}

	err = config.Validate()
	if err != nil {
		logger.Error(err.Error())

		os.Exit(3)
	}

	if d, _ := p.GetBool("dump-config"); d {
		fmt.Printf("%+v\n", config)

		os.Exit(0)
	}

	// Configure error handler
	errorHandler := errorhandler.New(logger)
	defer emperror.HandleRecover(errorHandler)

	buildInfo := buildinfo.New(version, commitHash, buildDate, branch)

	logger.Info("starting application", buildInfo.Fields())

	// default tracer
	tracer := tracing.NewNoOpTracer()

	// Configure Jaeger
	if config.Jaeger.Enabled {
		logger.Info("jaeger exporter enabled")

		tracing.SetupTracing(config.Jaeger.Config, emperror.NoopHandler{})
		tracer = tracing.NewTracer()
	}

	if config.Profiling.Enabled {
		logger.Info("profiling is enabled.", map[string]interface{}{
			"service_name": config.Profiling.ServiceName,
		})
		err := profiler.StartProfiling(config.Profiling.ServiceName)
		if err != nil {
			logger.Error(err.Error())
		}
	}

	cloudInfoLogger := cloudinfoadapter.NewLogger(logger)

	// use the configured store implementation
	cloudInfoStore := cistore.NewCloudInfoStore(config.Store, cloudInfoLogger)
	defer cloudInfoStore.Close()
	if !cloudInfoStore.Ready() {
		emperror.Panic(errors.New("configured product store not available"))
	}

	infoers, providers, err := loadInfoers(config, cloudInfoLogger)
	emperror.Panic(err)

	reporter := metrics.NewDefaultMetricsReporter()

	eventBus := messaging.NewDefaultEventBus(errorHandler)

	serviceManager := loader.NewDefaultServiceManager(config.ServiceLoader, cloudInfoStore, cloudInfoLogger, eventBus)
	serviceManager.ConfigureServices(providers)

	serviceManager.LoadServiceInformation(providers)

	prodInfo, err := cloudinfo.NewCloudInfo(providers, cloudInfoStore, cloudInfoLogger)
	emperror.Panic(err)

	if config.Scrape.Enabled {
		scrapingDriver := cloudinfo.NewScrapingDriver(config.Scrape.Interval, infoers, cloudInfoStore, eventBus, reporter, tracer, errorHandler, cloudInfoLogger)

		err = scrapingDriver.StartScraping()
		emperror.Panic(err)

		// start the management service
		// TODO: management requires scraping at the moment. Let's remove that dependency.
		if config.Management.Enabled {
			go management.StartManagementEngine(config.Management, cloudInfoStore, *scrapingDriver, cloudInfoLogger)
		}
	}

	err = api.ConfigureValidator(providers, prodInfo, cloudInfoLogger)
	emperror.Panic(err)

	cloudinfoLogger := cloudinfoadapter.NewLogger(logger)
	providerService := cloudinfo.NewProviderService(prodInfo)
	serviceService := cloudinfo.NewServiceService(prodInfo)
	regionService := cloudinfo.NewRegionService(prodInfo)
	instanceTypeService := cloudinfo.NewInstanceTypeService(prodInfo)
	endpoints := cloudinfodriver.MakeEndpoints(instanceTypeService)
	providerEndpoints := cloudinfodriver.MakeProviderEndpoints(providerService, cloudinfoLogger)
	serviceEndpoints := cloudinfodriver.MakeServiceEndpoints(serviceService, cloudinfoLogger)
	regionEndpoints := cloudinfodriver.MakeRegionEndpoints(regionService, cloudinfoLogger)
	graphqlHandler := cloudinfodriver.MakeGraphQLHandler(
		endpoints,
		providerEndpoints,
		serviceEndpoints,
		regionEndpoints,
		errorHandler,
	)

	routeHandler := api.NewRouteHandler(prodInfo, buildInfo, graphqlHandler, cloudInfoLogger)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// add prometheus metric endpoint
	if config.Metrics.Enabled {
		logger.Info("metrics enabled")

		routeHandler.EnableMetrics(router, config.Metrics.Address)
	}

	routeHandler.ConfigureRoutes(router, config.App.BasePath)

	err = router.Run(config.App.Address)
	emperror.Panic(errors.Wrap(err, "failed to run router"))
}

func loadInfoers(config configuration, logger cloudinfo.Logger) (map[string]cloudinfo.CloudInfoer, []string, error) {
	infoers := map[string]cloudinfo.CloudInfoer{}

	var providers []string

	if config.Provider.Amazon.Enabled {
		providers = append(providers, Amazon)
		logger := logger.WithFields(map[string]interface{}{"provider": Amazon})

		infoer, err := amazon.NewAmazonInfoer(config.Provider.Amazon.Config, logger)
		if err != nil {
			return nil, nil, errors.WithDetails(err, "provider", Amazon)
		}

		infoers[Amazon] = infoer

		logger.Info("configured cloud info provider")
	}

	if config.Provider.Google.Enabled {
		providers = append(providers, Google)
		logger := logger.WithFields(map[string]interface{}{"provider": Google})

		infoer, err := google.NewGoogleInfoer(config.Provider.Google.Config, logger)
		if err != nil {
			return nil, nil, emperror.With(err, "provider", Google)
		}

		infoers[Google] = infoer

		logger.Info("configured cloud info provider")
	}

	if config.Provider.Alibaba.Enabled {
		providers = append(providers, Alibaba)
		logger := logger.WithFields(map[string]interface{}{"provider": Alibaba})

		infoer, err := alibaba.NewAlibabaInfoer(config.Provider.Alibaba.Config, logger)
		if err != nil {
			return nil, nil, emperror.With(err, "provider", Alibaba)
		}

		infoers[Alibaba] = infoer

		logger.Info("configured cloud info provider")
	}

	if config.Provider.Oracle.Enabled {
		providers = append(providers, Oracle)
		logger := logger.WithFields(map[string]interface{}{"provider": Oracle})

		infoer, err := oracle.NewOracleInfoer(config.Provider.Oracle.Config, logger)
		if err != nil {
			return nil, nil, emperror.With(err, "provider", Oracle)
		}

		infoers[Oracle] = infoer

		logger.Info("configured cloud info provider")
	}

	if config.Provider.Azure.Enabled {
		providers = append(providers, Azure)
		logger := logger.WithFields(map[string]interface{}{"provider": Azure})

		infoer, err := azure.NewAzureInfoer(config.Provider.Azure.Config, logger)
		if err != nil {
			return nil, nil, emperror.With(err, "provider", Azure)
		}

		infoers[Azure] = infoer

		logger.Info("configured cloud info provider")
	}

	if config.Provider.Digitalocean.Enabled {
		providers = append(providers, Digitalocean)
		logger := logger.WithFields(map[string]interface{}{"provider": Digitalocean})

		infoer, err := digitalocean.NewDigitaloceanInfoer(config.Provider.Digitalocean.Config, logger)
		if err != nil {
			return nil, nil, emperror.With(err, "provider", Digitalocean)
		}

		infoers[Digitalocean] = infoer

		logger.Info("configured cloud info provider")
	}

	if config.Provider.VSphere.Enabled {
		providers = append(providers, Vsphere)
		logger := logger.WithFields(map[string]interface{}{"provider": Vsphere})
		logger.Info("configured cloud info provider")
	}

	return infoers, providers, nil
}
