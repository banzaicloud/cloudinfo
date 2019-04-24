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
	serviceName = "cloudinfo"

	// friendlyServiceName is the visible name of the service.
	friendlyServiceName = "Banzai Cloud Cloudinfo Service"

	// envPrefix is prepended to environment variables when processing configuration.
	envPrefix = "cloudinfo"

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
