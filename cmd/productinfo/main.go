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
	"os"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/banzaicloud/go-gin-prometheus"
	"github.com/banzaicloud/productinfo/internal/app/productinfo/api"
	"github.com/banzaicloud/productinfo/pkg/productinfo"
	"github.com/banzaicloud/productinfo/pkg/productinfo/alibaba"
	"github.com/banzaicloud/productinfo/pkg/productinfo/azure"
	"github.com/banzaicloud/productinfo/pkg/productinfo/ec2"
	"github.com/banzaicloud/productinfo/pkg/productinfo/gce"
	"github.com/banzaicloud/productinfo/pkg/productinfo/oci"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	flag "github.com/spf13/pflag"
)

const (
	// the list of flags supported by the application
	// these constants can be used to retrieve the passed in values or defaults via viper
	logLevelFlag               = "log-level"
	listenAddressFlag          = "listen-address"
	prodInfRenewalIntervalFlag = "product-info-renewal-interval"
	prometheusAddressFlag      = "prometheus-address"
	prometheusQueryFlag        = "prometheus-query"
	providerFlag               = "provider"
	helpFlag                   = "help"
	metricsEnabledFlag         = "metrics-enabled"
	metricsAddressFlag         = "metrics-address"

	//temporary flags
	gceApiKeyFlag          = "gce-api-key"
	azureSubscriptionId    = "azure-subscription-id"
	alibabaRegionId        = "alibaba-region-id"
	alibabaAccessKeyId     = "alibaba-access-key-id"
	alibabaAccessKeySecret = "alibaba-access-key-secret"

	// Gce is the identifier of the Google Cloud Engine provider
	Gce = "gce"
	// Ec2 is the identifier of the Amazon Ec2 provider
	Ec2 = "ec2"
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
	flag.String(listenAddressFlag, ":9090", "the address the productinfo app listens to HTTP requests.")
	flag.Duration(prodInfRenewalIntervalFlag, 24*time.Hour, "duration (in go syntax) between renewing the product information. Example: 2h30m")
	flag.String(prometheusAddressFlag, "", "http address of a Prometheus instance that has AWS spot "+
		"price metrics via banzaicloud/spot-price-exporter. If empty, the productinfo app will use current spot prices queried directly from the AWS API.")
	flag.String(prometheusQueryFlag, "avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])",
		"advanced configuration: change the query used to query spot price info from Prometheus.")
	flag.String(gceApiKeyFlag, "", "GCE API key to use for getting SKUs")
	flag.StringSlice(providerFlag, []string{Ec2, Gce, Azure, Oracle, Alibaba}, "Providers that will be used with the productinfo application.")
	flag.String(azureSubscriptionId, "", "Azure subscription ID to use with the APIs")
	flag.Bool(helpFlag, false, "print usage")
	flag.Bool(metricsEnabledFlag, false, "internal metrics are exposed if enabled")
	flag.String(metricsAddressFlag, ":9900", "the address where internal metrics are exposed")
	flag.String(alibabaRegionId, "", "alibaba region id")
	flag.String(alibabaAccessKeyId, "", "alibaba access key id")
	flag.String(alibabaAccessKeySecret, "", "alibaba access key secret")
}

// bindFlags binds parsed flags into viper
func bindFlags() {
	flag.Parse()
	viper.BindPFlags(flag.CommandLine)
}

// setLogLevel sets the log level
func setLogLevel() {
	parsedLevel, err := log.ParseLevel(viper.GetString("log-level"))
	if err != nil {
		log.WithError(err).Warnf("Couldn't parse log level, using default: %s", log.GetLevel())
	} else {
		log.SetLevel(parsedLevel)
		log.Debugf("Set log level to %s", parsedLevel)
	}
}
func init() {

	// describe the flags for the application
	defineFlags()

	// all the flags should be referenced through viper after this call
	// flags are available through the entire application via viper
	bindFlags()

	// handle log level
	setLogLevel()

	// Viper check for an environment variable
	viper.AutomaticEnv()
	replacer := strings.NewReplacer("-", "_")
	viper.SetEnvKeyReplacer(replacer)

	// register prometheus custom metrics
	prometheus.MustRegister(productinfo.ScrapeCompleteDurationGauge)
	prometheus.MustRegister(productinfo.ScrapeRegionDurationGauge)
	prometheus.MustRegister(productinfo.ScrapeFailuresTotalCounter)
	prometheus.MustRegister(productinfo.ScrapeShortLivedCompleteDurationGauge)
	prometheus.MustRegister(productinfo.ScrapeShortLivedRegionDurationGauge)
	prometheus.MustRegister(productinfo.ScrapeShortLivedFailuresTotalCounter)
}

func main() {

	if viper.GetBool(helpFlag) {
		flag.Usage()
		return
	}

	prodInfo, err := productinfo.NewCachingProductInfo(viper.GetDuration(prodInfRenewalIntervalFlag),
		cache.New(24*time.Hour, 24.*time.Hour), infoers())
	quitOnError("error encountered", err)

	go prodInfo.Start(context.Background())

	quitOnError("error encountered", err)

	// configure the gin validator
	api.ConfigureValidator(viper.GetStringSlice(providerFlag), prodInfo)

	routeHandler := api.NewRouteHandler(prodInfo)

	// new default gin engine (recovery, logger middleware)
	router := gin.Default()

	// add prometheus metric endpoint
	if viper.GetBool(metricsEnabledFlag) {
		p := ginprometheus.NewPrometheus("http", []string{"provider", "region"})
		p.SetListenAddress(viper.GetString(metricsAddressFlag))
		p.Use(router)
	}

	log.Info("Initialized gin router")
	routeHandler.ConfigureRoutes(router)
	log.Info("Configured routes")

	router.Run(viper.GetString(listenAddressFlag))
}

func infoers() map[string]productinfo.ProductInfoer {
	providers := viper.GetStringSlice(providerFlag)
	infoers := make(map[string]productinfo.ProductInfoer, len(providers))
	for _, p := range providers {
		var infoer productinfo.ProductInfoer
		var err error

		switch p {
		case Ec2:
			infoer, err = ec2.NewEc2Infoer(viper.GetString(prometheusAddressFlag), viper.GetString(prometheusQueryFlag))
		case Gce:
			infoer, err = gce.NewGceInfoer(viper.GetString(gceApiKeyFlag))
		case Azure:
			infoer, err = azure.NewAzureInfoer(viper.GetString(azureSubscriptionId))
		case Oracle:
			infoer, err = oci.NewInfoer()
		case Alibaba:
			infoer, err = alibaba.NewAlibabaInfoer(viper.GetString(alibabaRegionId), viper.GetString(alibabaAccessKeyId), viper.GetString(alibabaAccessKeySecret))
		default:
			log.Fatalf("provider %s is not supported", p)
		}

		quitOnError("could not initialize product info provider", err)

		infoers[p] = infoer
		log.Infof("Configured '%s' product info provider", p)
	}
	return infoers
}

func quitOnError(msg string, err error) {
	if err != nil {
		log.Errorf("%s : %s", msg, err.Error())
		flag.Usage()
		os.Exit(-1)
	}
}
