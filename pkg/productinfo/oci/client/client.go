package client

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

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
func NewOCI() (oci *OCI, err error) {

	configFileLocation := os.Getenv("ORACLE_CLI_CONFIG_LOCATION")
	if configFileLocation == "" {
		return nil, fmt.Errorf("Environment variable ORACLE_CLI_CONFIG_LOCATION is not set")
	}

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
