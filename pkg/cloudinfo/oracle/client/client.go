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

package client

import (
	"crypto/x509"
	"encoding/pem"

	"github.com/oracle/oci-go-sdk/common"
	"github.com/oracle/oci-go-sdk/identity"
	"github.com/sirupsen/logrus"
)

// OCI is for managing OCI API calls
type OCI struct {
	config  common.ConfigurationProvider
	logger  *logrus.Logger
	Tenancy identity.Tenancy
}

// NewOCI creates a new OCI and gets and caches tenancy info
func NewOCI(configFileLocation string) (oci *OCI, err error) {
	config, err := common.ConfigurationProviderFromFile(configFileLocation, "")
	if err != nil {
		return
	}

	oci = &OCI{
		config: config,
		logger: logrus.New(),
	}

	_, err = oci.GetTenancy()

	return oci, err
}

// ChangeRegion changes region in the config to the specified one
func (oci *OCI) ChangeRegion(regionName string) (err error) {

	i, err := oci.NewIdentityClient()
	if err != nil {
		return err
	}

	err = i.IsRegionAvailable(regionName)
	if err != nil {
		return err
	}

	tenancyOCID, _ := oci.config.TenancyOCID()
	userOCID, _ := oci.config.UserOCID()
	keyFingerprint, _ := oci.config.KeyFingerprint()
	privateRSAKey, _ := oci.config.PrivateRSAKey()

	var privateKey = &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateRSAKey),
	}

	privateKeyPEM := pem.EncodeToMemory(privateKey)

	config := common.NewRawConfigurationProvider(tenancyOCID, userOCID, regionName, keyFingerprint, string(privateKeyPEM), nil)
	oci.config = config

	return
}

// SetLogger sets a logrus logger
func (oci *OCI) SetLogger(logger *logrus.Logger) {

	oci.logger = logger
}

// GetLogger gets the previously set logrus logger
func (oci *OCI) GetLogger() *logrus.Logger {

	return oci.logger
}

// GetTenancy gets and caches tenancy info
func (oci *OCI) GetTenancy() (t identity.Tenancy, err error) {

	if oci.Tenancy.Id != nil {
		return oci.Tenancy, nil
	}

	tenancyID, err := oci.config.TenancyOCID()
	if err != nil {
		return t, err
	}

	i, err := oci.NewIdentityClient()
	if err != nil {
		return t, err
	}
	oci.Tenancy, err = i.GetTenancy(tenancyID)

	return oci.Tenancy, err
}
