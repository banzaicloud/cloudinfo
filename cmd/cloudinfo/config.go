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
	"os"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/cistore"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/loader"
	"github.com/banzaicloud/cloudinfo/internal/app/cloudinfo/management"
	"github.com/banzaicloud/cloudinfo/internal/platform/jaeger"
	"github.com/banzaicloud/cloudinfo/internal/platform/log"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/alibaba"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/amazon"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/azure"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/google"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/oracle"
)

// Provider constants
const (
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

// configuration holds any kind of configuration that comes from the outside world and
// is necessary for running the application.
type configuration struct {
	// Meaningful values are recommended (eg. production, development, staging, release/123, etc)
	Environment string

	// Turns on some debug functionality
	Debug bool

	// Timeout for graceful shutdown
	ShutdownTimeout time.Duration

	// Log configuration
	Log log.Config

	// Instrumentation configuration
	Instrumentation instrumentationConfig

	// App configuration
	App struct {
		// HTTP server address
		Address string

		RenewalInterval time.Duration

		// Providers to be scraped for product information
		Providers []string
	}

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

	Management management.Config

	ServiceLoader loader.Config

	Store cistore.Config
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	// TODO: write config validation
	return nil
}

// instrumentationConfig represents the instrumentation related configuration.
type instrumentationConfig struct {
	// Metrics configuration
	Metrics struct {
		Enabled bool
		Address string
	}

	// Jaeger configuration
	Jaeger struct {
		Enabled       bool
		jaeger.Config `mapstructure:",squash"`
	}
}

// configure configures some defaults in the Viper instance.
func configure(v *viper.Viper, p *pflag.FlagSet) {
	// Viper settings
	v.AddConfigPath(".")
	v.AddConfigPath(fmt.Sprintf("$%s_CONFIG_DIR/", strings.ToUpper(envPrefix)))

	// Environment variable settings
	// TODO: enable env prefix
	// v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Application constants
	v.Set("appName", appName)

	// Global configuration
	v.SetDefault("environment", "production")
	v.SetDefault("debug", false)
	v.SetDefault("shutdownTimeout", 5*time.Second)
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		v.SetDefault("no_color", true)
	}

	// Log configuration
	p.String("log-level", "info", "log level")
	_ = v.BindPFlag("log.level", p.Lookup("log-level"))

	p.String("log-format", "json", "log format")
	_ = v.BindPFlag("log.format", p.Lookup("log-format"))

	v.RegisterAlias("log.noColor", "no_color")

	// Instrumentation
	p.Bool("metrics-enabled", false, "internal metrics are exposed if enabled")
	_ = v.BindPFlag("instrumentation.metrics.enabled", p.Lookup("metrics-enabled"))
	_ = v.BindEnv("instrumentation.metrics.enabled", "METRICS_ENABLED")

	p.String("metrics-address", ":9090", "the address where internal metrics are exposed")
	_ = v.BindPFlag("instrumentation.metrics.address", p.Lookup("metrics-address"))
	_ = v.BindEnv("instrumentation.metrics.address", "METRICS_ADDRESS")

	v.SetDefault("instrumentation.jaeger.enabled", false)
	v.SetDefault("instrumentation.jaeger.collectorEndpoint", "http://localhost:14268/api/traces?format=jaeger.thrift")
	v.SetDefault("instrumentation.jaeger.agentEndpoint", "localhost:6832")
	v.RegisterAlias("instrumentation.jaeger.serviceName", "appName")
	_ = v.BindEnv("instrumentation.jaeger.username")
	_ = v.BindEnv("instrumentation.jaeger.password")

	// App configuration
	p.String("listen-address", ":8000", "application listen address")
	_ = v.BindPFlag("app.address", p.Lookup("listen-address"))
	_ = v.BindEnv("app.address", "LISTEN_ADDRESS")

	p.Duration("product-info-renewal-interval", 24*time.Hour, "duration (in go syntax) between renewing the product information. Example: 2h30m")
	_ = v.BindPFlag("app.renewalInterval", p.Lookup("product-info-renewal-interval"))
	_ = v.BindEnv("app.renewalInterval", "PRODUCT_INFO_RENEWAL_INTERVAL")

	p.StringSlice("provider", []string{Amazon, Google, Azure, Oracle, Alibaba}, "Providers that will be used with the cloudinfo application.")
	_ = v.BindPFlag("app.providers", p.Lookup("provider"))
	_ = v.BindEnv("app.providers", "PROVIDER")

	// Amazon config
	p.String("aws-access-key-id", "", "aws access key id")
	_ = v.BindPFlag("amazon.accessKeyId", p.Lookup("aws-access-key-id"))
	_ = v.BindEnv("amazon.accessKeyId", "AWS_ACCESS_KEY_ID")

	p.String("aws-secret-access-key", "", "aws secret access key")
	_ = v.BindPFlag("amazon.secretAccessKey", p.Lookup("aws-secret-access-key"))
	_ = v.BindEnv("amazon.secretAccessKey", "AWS_SECRET_ACCESS_KEY")

	p.String(
		"prometheus-address",
		"",
		"http address of a Prometheus instance that has AWS spot "+
			"price metrics via banzaicloud/spot-price-exporter. "+
			"If empty, the cloudinfo app will use current spot prices queried directly from the AWS API.",
	)
	_ = v.BindPFlag("amazon.prometheusAddress", p.Lookup("prometheus-address"))
	_ = v.BindEnv("amazon.prometheusAddress", "PROMETHEUS_ADDRESS")

	p.String(
		"prometheus-query",
		"avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])",
		"advanced configuration: change the query used to query spot price info from Prometheus.",
	)
	_ = v.BindPFlag("amazon.prometheusQuery", p.Lookup("prometheus-query"))
	_ = v.BindEnv("amazon.prometheusQuery", "PROMETHEUS_QUERY")

	// Google config
	p.String("gce-api-key", "", "GCE API key to use for getting SKUs")
	_ = v.BindPFlag("google.apiKey", p.Lookup("gce-api-key"))
	_ = v.BindEnv("google.apiKey", "GCE_API_KEY")

	p.String("google-application-credentials", "", "google application credentials location")
	_ = v.BindPFlag("google.appCredentials", p.Lookup("google-application-credentials"))
	_ = v.BindEnv("google.appCredentials", "GOOGLE_APPLICATION_CREDENTIALS")

	// Alibaba config
	p.String("alibaba-region-id", "", "alibaba region id")
	_ = v.BindPFlag("alibaba.regionId", p.Lookup("alibaba-region-id"))
	_ = v.BindEnv("alibaba.regionId", "ALIBABA_REGION_ID")

	p.String("alibaba-access-key-id", "", "alibaba access key id")
	_ = v.BindPFlag("alibaba.accessKeyId", p.Lookup("alibaba-access-key-id"))
	_ = v.BindEnv("alibaba.accessKeyId", "ALIBABA_ACCESS_KEY_ID")

	p.String("alibaba-access-key-secret", "", "alibaba access key secret")
	_ = v.BindPFlag("alibaba.accessKeySecret", p.Lookup("alibaba-access-key-secret"))
	_ = v.BindEnv("alibaba.accessKeySecret", "ALIBABA_ACCESS_KEY_SECRET")

	// Oracle config
	p.String("oracle-cli-config-location", "", "oracle config file location")
	_ = v.BindPFlag("oracle.configLocation", p.Lookup("oracle-cli-config-location"))
	_ = v.BindEnv("oracle.configLocation", "ORACLE_CLI_CONFIG_LOCATION")

	// Azure config
	p.String("azure-auth-location", "", "azure authentication file location")
	_ = v.BindPFlag("azure.authLocation", p.Lookup("azure-auth-location"))
	_ = v.BindEnv("azure.authLocation", "AZURE_AUTH_LOCATION")

	// Management
	v.SetDefault("management.enabled", true)
	v.SetDefault("management.address", ":8001")

	// ServiceLoader
	v.SetDefault("serviceloader.serviceConfigLocation", "./configs")
	v.SetDefault("serviceloader.serviceConfigName", "services")
	v.SetDefault("serviceloader.format", "yaml")

	// CloudInfoStore
	// Redis product store
	v.SetDefault("store.redis.enabled", false)
	v.SetDefault("store.redis.host", "localhost")
	v.SetDefault("store.redis.port", 6379)

	// Cassandra product store
	v.SetDefault("store.cassandra.enabled", false)
	v.SetDefault("store.cassandra.hosts", "localhost")
	v.SetDefault("store.cassandra.port", 9042)
	v.SetDefault("store.cassandra.keyspace", "cloudinfo")
	v.SetDefault("store.cassandra.table", "products")

	// InMemory product store
	v.SetDefault("store.gocache.expiration", 0)
	v.SetDefault("store.gocache.cleanupInterval", 0)
}
