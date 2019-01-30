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

const (
	// It identifies the service itself, the actual instance needs to be identified via environment
	// and other details.
	ServiceName = "cloudinfo"

	// FriendlyServiceName is the visible name of the service.
	FriendlyServiceName = "Banzai Cloud Cloudinfo Service"

	// the list of flags supported by the application
	// these constants can be used to retrieve the passed in values or defaults via viper
	logLevelFlag               = "log-level"
	logFormatFlag              = "log-format"
	listenAddressFlag          = "listen-address"
	prodInfRenewalIntervalFlag = "product-info-renewal-interval"
	prometheusAddressFlag      = "prometheus-address"
	prometheusQueryFlag        = "prometheus-query"
	providerFlag               = "provider"
	helpFlag                   = "help"
	metricsEnabledFlag         = "metrics-enabled"
	metricsAddressFlag         = "metrics-address"

	//temporary flags
	awsAccessKeyId         = "aws-access-key-id"
	awsSecretAccessKey     = "aws-secret-access-key"
	gceApiKeyFlag          = "gce-api-key"
	gceApplicationCred     = "google-application-credentials"
	azureAuthLocation      = "azure-auth-location"
	alibabaRegionId        = "alibaba-region-id"
	alibabaAccessKeyId     = "alibaba-access-key-id"
	alibabaAccessKeySecret = "alibaba-access-key-secret"
	oracleConfigLocation   = "oracle-cli-config-location"

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
