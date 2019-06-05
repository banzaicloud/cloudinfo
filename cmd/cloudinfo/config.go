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
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
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
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/digitalocean"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/google"
	"github.com/banzaicloud/cloudinfo/pkg/cloudinfo/oracle"
)

// Provider constants
const (
	// Amazon is the identifier of the Amazon provider
	Amazon = "amazon"
	// Google is the identifier of the Google Cloud Engine provider
	Google = "google"
	// Alibaba is the identifier of the Alibaba Cloud provider
	Alibaba = "alibaba"
	// Oracle is the identifier of the Oracle Cloud Infrastructure provider
	Oracle = "oracle"
	// Azure is the identifier of the MS Azure provider
	Azure = "azure"
	// Digitalocean is the identifier of the Digitalocean provider
	Digitalocean = "digitalocean"
)

// metaConfiguration contains meta configuration for eg. remote config providers.
type metaConfiguration struct {
	// Vault configuration
	Vault struct {
		Enabled    bool
		Address    string
		Token      string
		SecretPath string
	}
}

// Validate validates the metaConfiguration.
func (c metaConfiguration) Validate() error {
	if c.Vault.Enabled {
		if c.Vault.Address != "" {
			if _, err := url.Parse(c.Vault.Address); err != nil {
				return errors.Wrap(err, "invalid vault address")
			}
		}

		if c.Vault.SecretPath == "" {
			return errors.New("vault secret path is required")
		}
	}

	return nil
}

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

	// App configuration
	App struct {
		// HTTP server address
		Address string

		BasePath string
	}

	// Scrape configuration
	Scrape struct {
		Enabled bool

		// Cloud info scrape interval
		Interval time.Duration
	}

	// Provider configuration
	Provider struct {
		// Amazon configuration
		Amazon struct {
			Enabled       bool
			amazon.Config `mapstructure:",squash"`
		}

		// Google configuration
		Google struct {
			Enabled       bool
			google.Config `mapstructure:",squash"`
		}

		// Alibaba configuration
		Alibaba struct {
			Enabled        bool
			alibaba.Config `mapstructure:",squash"`
		}

		// Oracle configuration
		Oracle struct {
			Enabled       bool
			oracle.Config `mapstructure:",squash"`
		}

		// Azure configuration
		Azure struct {
			Enabled      bool
			azure.Config `mapstructure:",squash"`
		}

		// Digitalocean configuration
		Digitalocean struct {
			Enabled             bool
			digitalocean.Config `mapstructure:",squash"`
		}
	}

	Management management.Config

	ServiceLoader loader.Config

	Store cistore.Config
}

// Validate validates the configuration.
func (c configuration) Validate() error {
	// TODO: write config validation

	if !c.Scrape.Enabled && !(c.Store.Redis.Enabled || c.Store.Cassandra.Enabled) {
		return errors.New("storage is required when scraping is disabled")
	}

	return nil
}

// configure configures some defaults in the Viper instance.
func configure(v *viper.Viper, p *pflag.FlagSet) {
	// Viper settings
	v.AddConfigPath(".")
	v.AddConfigPath(fmt.Sprintf("$%s_CONFIG_DIR/", strings.ToUpper(envPrefix)))

	// Environment variable settings
	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
	v.AllowEmptyEnv(true)
	v.AutomaticEnv()

	// Pflag settings
	p.SortFlags = false

	// Application constants
	v.Set("appName", appName)

	// Global configuration
	v.SetDefault("environment", "production")
	v.SetDefault("debug", false)
	v.SetDefault("shutdownTimeout", 5*time.Second)
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		v.SetDefault("no_color", true)
	}

	// Vault configuration
	p.String("config-vault", "", "enable config Vault")
	_ = v.BindPFlag("config.vault.enabled", p.Lookup("config-vault"))

	p.String("config-vault-address", "", "config Vault address")
	_ = v.BindPFlag("config.vault.address", p.Lookup("config-vault-address"))

	p.String("config-vault-token", "", "config Vault token")
	_ = v.BindPFlag("config.vault.token", p.Lookup("config-vault-token"))

	p.String("config-vault-secret-path", "", "config Vault secret path")
	_ = v.BindPFlag("config.vault.secretPath", p.Lookup("config-vault-secret-path"))

	// Log configuration
	p.String("log-level", "info", "log level")
	_ = v.BindPFlag("log.level", p.Lookup("log-level"))

	p.String("log-format", "json", "log format")
	_ = v.BindPFlag("log.format", p.Lookup("log-format"))

	v.RegisterAlias("log.noColor", "no_color")

	// Instrumentation
	p.Bool("metrics-enabled", false, "internal metrics are exposed if enabled")
	_ = v.BindPFlag("metrics.enabled", p.Lookup("metrics-enabled"))

	p.String("metrics-address", ":9090", "the address where internal metrics are exposed")
	_ = v.BindPFlag("metrics.address", p.Lookup("metrics-address"))

	v.SetDefault("jaeger.enabled", false)
	v.SetDefault("jaeger.collectorEndpoint", "http://localhost:14268/api/traces?format=jaeger.thrift")
	v.SetDefault("jaeger.agentEndpoint", "localhost:6832")
	v.RegisterAlias("jaeger.serviceName", "appName")
	_ = v.BindEnv("jaeger.username")
	_ = v.BindEnv("jaeger.password")

	// App configuration
	p.String("listen-address", ":8000", "application listen address")
	_ = v.BindPFlag("app.address", p.Lookup("listen-address"))

	v.SetDefault("app.basePath", "/")

	// Scrape configuration
	p.Bool("scrape", true, "enable cloud info scraping")
	_ = v.BindPFlag("scrape.enabled", p.Lookup("scrape"))

	p.Duration("scrape-interval", 24*time.Hour, "duration (in go syntax) between renewing information")
	_ = v.BindPFlag("scrape.interval", p.Lookup("scrape-interval"))

	// Amazon config
	p.Bool("provider-amazon", false, "enable amazon provider")
	_ = v.BindPFlag("provider.amazon.enabled", p.Lookup("provider-amazon"))

	_ = v.BindEnv("provider.amazon.accessKey")
	_ = v.BindEnv("provider.amazon.secretKey")
	_ = v.BindEnv("provider.amazon.sharedCredentialsFile")
	_ = v.BindEnv("provider.amazon.profile")
	v.SetDefault("provider.amazon.prometheusAddress", "")
	v.SetDefault("provider.amazon.prometheusQuery", "avg_over_time(aws_spot_current_price{region=\"%s\", product_description=\"Linux/UNIX\"}[1w])")

	// Google config
	p.Bool("provider-google", false, "enable google provider")
	_ = v.BindPFlag("provider.google.enabled", p.Lookup("provider-google"))

	_ = v.BindEnv("provider.google.credentials", "GOOGLE_CREDENTIALS")
	_ = v.BindEnv("provider.google.credentialsFile", "GOOGLE_CREDENTIALS_FILE")
	_ = v.BindEnv("provider.google.project", "GOOGLE_PROJECT")

	// Alibaba config
	p.Bool("provider-alibaba", false, "enable alibaba provider")
	_ = v.BindPFlag("provider.alibaba.enabled", p.Lookup("provider-alibaba"))

	_ = v.BindEnv("provider.alibaba.region", "ALIBABA_REGION_ID")
	_ = v.BindEnv("provider.alibaba.region", "ALICLOUD_REGION")
	_ = v.BindEnv("provider.alibaba.accessKey", "ALIBABA_ACCESS_KEY_ID")
	_ = v.BindEnv("provider.alibaba.accessKey", "ALICLOUD_ACCESS_KEY")
	_ = v.BindEnv("provider.alibaba.secretKey", "ALIBABA_ACCESS_KEY_SECRET")
	_ = v.BindEnv("provider.alibaba.secretKey", "ALICLOUD_SECRET_KEY")

	// Oracle config
	p.Bool("provider-oracle", false, "enable oracle provider")
	_ = v.BindPFlag("provider.oracle.enabled", p.Lookup("provider-oracle"))

	_ = v.BindEnv("provider.oracle.tenancy", "ORACLE_TENANCY_OCID")
	_ = v.BindEnv("provider.oracle.user", "ORACLE_USER_OCID")
	_ = v.BindEnv("provider.oracle.region", "ORACLE_REGION")
	_ = v.BindEnv("provider.oracle.fingerprint", "ORACLE_FINGERPRINT")
	_ = v.BindEnv("provider.oracle.privateKey", "ORACLE_PRIVATE_KEY")
	_ = v.BindEnv("provider.oracle.privateKeyPassphrase", "ORACLE_PRIVATE_KEY_PASSPHRASE")
	_ = v.BindEnv("provider.oracle.configFilePath", "ORACLE_CONFIG_FILE_PATH")
	_ = v.BindEnv("provider.oracle.profile", "ORACLE_PROFILE")

	// Azure config
	p.Bool("provider-azure", false, "enable azure provider")
	_ = v.BindPFlag("provider.azure.enabled", p.Lookup("provider-azure"))

	_ = v.BindEnv("provider.azure.subscriptionId", "AZURE_SUBSCRIPTION_ID")
	_ = v.BindEnv("provider.azure.clientId")
	_ = v.BindEnv("provider.azure.clientSecret")
	_ = v.BindEnv("provider.azure.tenantId")

	// DigitalOcean config
	p.Bool("provider-digitalocean", false, "enable digitalocean provider")
	_ = v.BindPFlag("provider.digitalocean.enabled", p.Lookup("provider-digitalocean"))

	_ = v.BindEnv("provider.digitalocean.accessToken", "DIGITALOCEAN_ACCESS_TOKEN")

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
