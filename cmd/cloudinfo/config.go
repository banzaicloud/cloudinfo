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
	"errors"
	"fmt"
	"github.com/banzaicloud/cloudinfo/internal/platform/prometheus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"strings"
	"time"
)

// Config holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type Config struct {
	// Meaningful values are recommended (eg. production, development, staging, release/123, etc)
	Environment string

	// Turns on some debug functionality (eg. more verbose logs)
	Debug bool

	// Timeout for graceful shutdown
	ShutdownTimeout time.Duration

	// Instrumentation configuration
	Instrumentation InstrumentationConfig

	// App configuration
	App struct {
		// App server address
		Addr string
	}
}

// Validate validates the configuration.
func (c Config) Validate() error {
	if c.Environment == "" {
		return errors.New("environment is required")
	}

	if err := c.Instrumentation.Validate(); err != nil {
		return err
	}

	if c.App.Addr == "" {
		return errors.New("app server address is required")
	}

	return nil
}

// InstrumentationConfig represents the instrumentation related configuration.
type InstrumentationConfig struct {
	// Instrumentation HTTP server address
	Addr string

	// Prometheus configuration
	Prometheus struct {
		Enabled           bool
		prometheus.Config `mapstructure:",squash"`
	}
}

// Validate validates the configuration.
func (c InstrumentationConfig) Validate() error {
	if c.Addr == "" {
		return errors.New("instrumentation http server address is required")
	}

	if c.Prometheus.Enabled {
		if err := c.Prometheus.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// defineFlags defines supported flags and makes them available for viper
func defineFlags(pf *pflag.FlagSet) {
	pflag.String(logLevelFlag, "info", "log level")
	pflag.String(logFormatFlag, "", "log format")
	pflag.String(listenAddressFlag, ":9090", "the address the cloudinfo app listens to HTTP requests.")
	pflag.Duration(prodInfRenewalIntervalFlag, 24*time.Hour, "duration (in go syntax) between renewing the product information. Example: 2h30m")
	pflag.String(prometheusAddressFlag, "", "http address of a Prometheus instance that has AWS spot "+
		"price metrics via banzaicloud/spot-price-exporter. If empty, the cloudinfo app will use current spot prices queried directly from the AWS API.")
	pflag.String(prometheusQueryFlag, "avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])",
		"advanced configuration: change the query used to query spot price info from Prometheus.")
	pflag.String(gceApiKeyFlag, "", "GCE API key to use for getting SKUs")
	pflag.String(gceApplicationCred, "", "google application credentials location")
	pflag.StringSlice(providerFlag, []string{Amazon, Google, Azure, Oracle, Alibaba}, "Providers that will be used with the cloudinfo application.")
	pflag.Bool(helpFlag, false, "print usage")
	pflag.Bool(metricsEnabledFlag, false, "internal metrics are exposed if enabled")
	pflag.String(metricsAddressFlag, ":9900", "the address where internal metrics are exposed")
	pflag.String(azureAuthLocation, "", "azure authentication file location")
	pflag.String(alibabaRegionId, "", "alibaba region id")
	pflag.String(alibabaAccessKeyId, "", "alibaba access key id")
	pflag.String(alibabaAccessKeySecret, "", "alibaba access key secret")
	pflag.String(oracleConfigLocation, "", "oracle config file location")
	pflag.String(alibabaPriceInfoUrl, "https://g.alicdn.com/aliyun/ecs-price-info-intl/2.0.8/price/download/instancePrice.json", "Alibaba get price info from this file")
}

// Configure configures some defaults in the Viper instance.
func Configure(v *viper.Viper, pf *pflag.FlagSet) {
	// configure viper
	// Viper check for an environment variable
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	pf.Init(FriendlyServiceName, pflag.ExitOnError)

	// define flags
	defineFlags(pf)

	// parse the command line
	pflag.Parse()

	// bind flags to viper
	if err := viper.BindPFlags(pf); err != nil {
		panic(fmt.Errorf("could not parse flags. error: %s", err))
	}

}
