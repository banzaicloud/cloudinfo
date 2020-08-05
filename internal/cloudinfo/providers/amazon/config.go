// Copyright © 2019 Banzai Cloud
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

package amazon

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
)

// Config represents configuration for obtaining cloud information from Amazon.
type Config struct {
	Credentials `mapstructure:",squash"`

	Pricing PricingConfig

	// Prometheus settings
	PrometheusAddress string
	PrometheusQuery   string
}

// PricingConfig represents configuration for obtaining pricing information from Amazon.
type PricingConfig struct {
	Region string

	Credentials `mapstructure:",squash"`
}

func (c Config) GetPricingCredentials() Credentials {
	creds := c.Pricing.Credentials

	if creds.AccessKey == "" {
		creds.AccessKey = c.AccessKey
	}

	if creds.SecretKey == "" {
		creds.SecretKey = c.SecretKey
	}

	if creds.SessionToken == "" {
		creds.SessionToken = c.SessionToken
	}

	if creds.SharedCredentialsFile == "" {
		creds.SharedCredentialsFile = c.SharedCredentialsFile
	}

	if creds.Profile == "" {
		creds.Profile = c.Profile
	}

	return creds
}

// Credentials used for creating an AWS Session.
type Credentials struct {
	// Static credentials
	AccessKey    string
	SecretKey    string
	SessionToken string

	// Shared credentials
	SharedCredentialsFile string
	Profile               string
}

func configFromCredentials(creds Credentials) *aws.Config {
	providers := []credentials.Provider{
		&credentials.StaticProvider{Value: credentials.Value{
			AccessKeyID:     creds.AccessKey,
			SecretAccessKey: creds.SecretKey,
			SessionToken:    creds.SessionToken,
		}},
		&credentials.SharedCredentialsProvider{
			Filename: creds.SharedCredentialsFile,
			Profile:  creds.Profile,
		},
	}

	return &aws.Config{
		Credentials: credentials.NewChainCredentials(providers),
	}
}
