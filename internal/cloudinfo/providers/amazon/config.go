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
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
)

// Config represents configuration for obtaining cloud information from Amazon.
type Config struct {
	Credentials `mapstructure:",squash"`

	Region string

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
	if reflect.DeepEqual(c.Pricing.Credentials, Credentials{}) {
		return c.Credentials
	}

	return c.Pricing.Credentials
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

	// IAM role ARN to assume
	AssumeRoleARN string
}

func configFromCredentials(creds Credentials) (*aws.Config, error) {
	var providers []credentials.Provider

	if creds.AccessKey != "" && creds.SecretKey != "" {
		providers = append(providers, &credentials.StaticProvider{Value: credentials.Value{
			AccessKeyID:     creds.AccessKey,
			SecretAccessKey: creds.SecretKey,
			SessionToken:    creds.SessionToken,
		}})
	}

	if creds.SharedCredentialsFile != "" {
		providers = append(providers, &credentials.SharedCredentialsProvider{
			Filename: creds.SharedCredentialsFile,
			Profile:  creds.Profile,
		})
	}

	awsConfig := &aws.Config{}
	if len(providers) > 0 {
		awsConfig.Credentials = credentials.NewChainCredentials(providers)
	}

	if creds.AssumeRoleARN != "" {
		sess, err := session.NewSession(awsConfig)
		if err != nil {
			return nil, err
		}

		return &aws.Config{
			Credentials: stscreds.NewCredentials(sess, creds.AssumeRoleARN),
		}, nil
	}

	return awsConfig, nil
}
