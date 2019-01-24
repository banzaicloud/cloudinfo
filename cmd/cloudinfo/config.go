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

import (
	"fmt"
	"strings"
	"time"

	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/alibaba"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/amazon"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/azure"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/google"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/oracle"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Config holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type Config struct {
	// Meaningful values are recommended (eg. production, development, staging, release/123, etc)
	Environment string

	// Turns on some debug functionality
	Debug bool

	// Timeout for graceful shutdown
	ShutdownTimeout time.Duration

	RenewalInterval time.Duration

	// Log configuration
	Log log.Config

	// providers to be scraped for product information
	Providers []string

	// Amazon configuration
	Amazon amazon.Config

	// Google configuration
	Google google.Config

	// Alibaba configuration
	Alibaba alibaba.Config

	// Oracle configuration
	Oracle oracle.Config

	// Azure configuration
	Azure azure.Config
}

// defineFlags defines supported flags and makes them available for viper
func defineFlags(pf *pflag.FlagSet) {
	pf.String(logLevelFlag, "info", "log level")
	pf.String(logFormatFlag, "", "log format")
	pf.String(listenAddressFlag, ":9090", "the address the cloudinfo app listens to HTTP requests.")
	pf.Duration(prodInfRenewalIntervalFlag, 24*time.Hour, "duration (in go syntax) between renewing the product information. Example: 2h30m")
	pf.String(prometheusAddressFlag, "", "http address of a Prometheus instance that has AWS spot "+
		"price metrics via banzaicloud/spot-price-exporter. If empty, the cloudinfo app will use current spot prices queried directly from the AWS API.")
	pf.String(prometheusQueryFlag, "avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])",
		"advanced configuration: change the query used to query spot price info from Prometheus.")
	pf.String(awsAccessKeyId, "", "aws access key id")
	pf.String(awsSecretAccessKey, "", "aws secret access key")
	pf.String(gceApiKeyFlag, "", "GCE API key to use for getting SKUs")
	pf.String(gceApplicationCred, "", "google application credentials location")
	pf.StringSlice(providerFlag, []string{Amazon, Google, Azure, Oracle, Alibaba}, "Providers that will be used with the cloudinfo application.")
	pf.Bool(helpFlag, false, "print usage")
	pf.Bool(metricsEnabledFlag, false, "internal metrics are exposed if enabled")
	pf.String(metricsAddressFlag, ":9900", "the address where internal metrics are exposed")
	pf.String(azureAuthLocation, "", "azure authentication file location")
	pf.String(alibabaRegionId, "", "alibaba region id")
	pf.String(alibabaAccessKeyId, "", "alibaba access key id")
	pf.String(alibabaAccessKeySecret, "", "alibaba access key secret")
	pf.String(oracleConfigLocation, "", "oracle config file location")
	pf.String(alibabaPriceInfoUrl, "https://g.alicdn.com/aliyun/ecs-price-info-intl/2.0.9/price/download/instancePrice.json", "Alibaba get price info from this file")
}

// Configure configures some defaults in the Viper instance.
func Configure(v *viper.Viper, pf *pflag.FlagSet) {
	// configure viper
	// Viper check for an environment variable

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	v.AutomaticEnv()

	// Log configuration
	v.RegisterAlias("log.format", logFormatFlag)
	v.RegisterAlias("log.level", logLevelFlag)
	v.RegisterAlias("log.noColor", "no_color")

	v.RegisterAlias("renewalinterval", prodInfRenewalIntervalFlag)

	v.RegisterAlias("providers", providerFlag)

	// Amazon config
	v.RegisterAlias("amazon.accesskeyid", awsAccessKeyId)
	v.RegisterAlias("amazon.secretaccesskey", awsSecretAccessKey)
	v.RegisterAlias("amazon.prometheusaddress", prometheusAddressFlag)
	v.RegisterAlias("amazon.prometheusquery", prometheusQueryFlag)

	//Google config
	v.RegisterAlias("google.apikey", gceApiKeyFlag)
	v.RegisterAlias("google.appcredentials", gceApplicationCred)

	// Alibaba config
	v.RegisterAlias("alibaba.accesskeyid", alibabaAccessKeyId)
	v.RegisterAlias("alibaba.accesskeysecret", alibabaAccessKeySecret)
	v.RegisterAlias("alibaba.regionid", alibabaRegionId)

	// Oracle config
	v.RegisterAlias("oracle.configlocation", oracleConfigLocation)

	// Azure config
	v.RegisterAlias("azure.authlocation", azureAuthLocation)

	pf.Init(FriendlyServiceName, pflag.ExitOnError)

	// define flags
	defineFlags(pf)

	// bind flags to viper
	if err := viper.BindPFlags(pf); err != nil {
		panic(fmt.Errorf("could not parse flags. error: %s", err))
	}

}
