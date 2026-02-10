// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

// Package config provides configuration management for the DKAM service.
package config

import (
	"context"
	"flag"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"gopkg.in/yaml.v3"

	as "github.com/open-edge-platform/infra-core/inventory/v2/pkg/artifactservice"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
)

const (
	// DefaultTimeout is the default timeout for HTTP requests.
	DefaultTimeout = 3 * time.Second

	// DownloadPath is the directory path for downloading artifacts.
	DownloadPath = "/tmp"
)

// InfraConfig holds the infrastructure configuration settings.
//
//nolint:tagliatelle // yaml tags use kebab-case for Kubernetes/Helm compatibility
type InfraConfig struct {
	ENDebianPackagesRepo string `mapstructure:"enDebianPackagesRepo" yaml:"enDebianPackagesRepo"`
	ENFilesRsRoot        string `mapstructure:"enFilesRsRoot" yaml:"enFilesRsRoot"`

	ENManifestRepo     string `mapstructure:"enManifestRepo" yaml:"enManifestRepo"`
	ENAgentManifestTag string `mapstructure:"enAgentManifestTag" yaml:"enAgentManifestTag"`

	InfraURL                string `mapstructure:"orchInfra" yaml:"orchInfra"`
	ClusterURL              string `mapstructure:"orchCluster" yaml:"orchCluster"`
	UpdateURL               string `mapstructure:"orchUpdate" yaml:"orchUpdate"`
	ReleaseServiceURL       string `mapstructure:"orchRelease" yaml:"orchRelease"`
	LogsObservabilityURL    string `mapstructure:"orchPlatformObsLogs" yaml:"orchPlatformObsLogs"`
	MetricsObservabilityURL string `mapstructure:"orchPlatformObsMetrics" yaml:"orchPlatformObsMetrics"`
	ManageabilityURL        string `mapstructure:"orchDeviceManager" yaml:"orchDeviceManager"`
	RPSAddress              string `mapstructure:"orchRpsHost" yaml:"orchRpsHost"`
	KeycloakURL             string `mapstructure:"orchKeycloak" yaml:"orchKeycloak"`
	TelemetryURL            string `mapstructure:"orchTelemetry" yaml:"orchTelemetry"`
	RegistryURL             string `mapstructure:"orchRegistry" yaml:"orchRegistry"`
	FileServerURL           string `mapstructure:"orchFileServer" yaml:"orchFileServer"`
	RSType                  string `mapstructure:"rsType" yaml:"rsType"`

	ENServiceClients  []string `mapstructure:"enServiceClients" yaml:"enServiceClients"`
	ENOutboundClients []string `mapstructure:"enOutboundClients" yaml:"enOutboundClients"`
	ENMetricsEnabled  string   `mapstructure:"enMetricsEnabled" yaml:"enMetricsEnabled"`
	ENTokenClients    []string `mapstructure:"enTokenClients" yaml:"enTokenClients"`

	ProvisioningService string `mapstructure:"provisioningSvc" yaml:"provisioningSvc"`
	// ProvisioningServerURL full URL to the provisioning server, including prefixes and subpaths
	ProvisioningServerURL string `mapstructure:"provisioningServerURL" yaml:"provisioningServerURL"`
	TinkServerURL         string `mapstructure:"tinkerSvc" yaml:"tinkerSvc"`
	OnboardingURL         string `mapstructure:"omSvc" yaml:"omSvc"`
	OnboardingStreamURL   string `mapstructure:"omStreamSvc" yaml:"omStreamSvc"`
	CDN                   string `mapstructure:"cdnSvc" yaml:"cdnSvc"`

	SystemConfigFsInotifyMaxUserInstances uint32 `mapstructure:"systemConfigFsInotifyMaxUserInstances" yaml:"systemConfigFsInotifyMaxUserInstances"` //nolint:lll // long struct tags required
	//nolint:revive,stylecheck // keep the name in sync with charts values
	SystemConfigVmOverCommitMemory uint32 `mapstructure:"systemConfigVmOverCommitMemory" yaml:"systemConfigVmOverCommitMemory"` //nolint:lll // long struct tags required
	SystemConfigKernelPanicOnOops  uint32 `mapstructure:"systemConfigKernelPanicOnOops" yaml:"systemConfigKernelPanicOnOops"`   //nolint:lll // long struct tags required
	SystemConfigKernelPanic        uint32 `mapstructure:"systemConfigKernelPanic" yaml:"systemConfigKernelPanic"`

	ENProxyHTTP    string   `mapstructure:"enProxyHTTP" yaml:"enProxyHTTP"`
	ENProxyHTTPS   string   `mapstructure:"enProxyHTTPS" yaml:"enProxyHTTPS"`
	ENProxyFTP     string   `mapstructure:"enProxyFTP" yaml:"enProxyFTP"`
	ENProxyNoProxy string   `mapstructure:"enProxyNoProxy" yaml:"enProxyNoProxy"`
	ENProxySocks   string   `mapstructure:"enProxySocks" yaml:"enProxySocks"`
	NetIP          string   `mapstructure:"netIP" yaml:"netIP"`
	NTPServers     []string `mapstructure:"ntpServer" yaml:"ntpServer"`
	DNSServers     []string `mapstructure:"nameServers" yaml:"nameServers"`

	FirewallReqAllow string `mapstructure:"firewallReqAllow" yaml:"firewallReqAllow"`
	FirewallCfgAllow string `mapstructure:"firewallCfgAllow" yaml:"firewallCfgAllow"`

	ENManifest ENManifest

	EMBImageURL string `mapstructure:"embImageURL" yaml:"embImageURL"`
	// Disable AOCO config
	DisableCOProfile   bool `mapstructure:"disableCoProfile" yaml:"disableCoProfile"`
	DisableO11YProfile bool `mapstructure:"disableO11YProfile" yaml:"disableO11YProfile"`
	SkipOSProvisioning bool `mapstructure:"skipOSProvisioning" yaml:"skipOSProvisioning"`
}

// ENManifest represents the Edge Node Agents release manifest.
type ENManifest struct {
	Repository Repository      `yaml:"repository"`
	Packages   []AgentsVersion `yaml:"packages"`
}

// Repository represents a container image repository.
type Repository struct {
	Codename  string `yaml:"codename"`
	Component string `yaml:"component"`
}

// AgentsVersion represents version information for an agent.
type AgentsVersion struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

// As a variable to allow changes in tests.
var (
	zlog = logging.GetLogger("InfraConfig")

	FlagConfigFilePath = flag.String("configFile", "", "Path to shared infra configuration file")

	currentInfraConfig InfraConfig
	configLock         sync.RWMutex

	PVC                    = "/data"
	BootsCaCertificateFile = "/etc/ssl/boots-ca-cert/ca.crt"
	OrchCACertificateFile  = "/etc/ssl/orch-ca-cert/ca.crt"
	ScriptPath             = "/home/appuser/pkg/script"
)

// Read reads and validates the configuration from the config file.
func Read() error {
	zlog.Info().Msgf("Config file path: %s", *FlagConfigFilePath)
	viper.SetConfigFile(*FlagConfigFilePath)
	viper.SetTypeByDefaultValue(true)
	if err := viper.ReadInConfig(); err != nil {
		zlog.Error().Err(err).Msgf("Failed to read infra config from path %s", *FlagConfigFilePath)
		return err
	}

	updateConfig := func() error {
		var config InfraConfig

		err := viper.Unmarshal(&config)
		if err != nil {
			return err
		}

		if config.ENManifestRepo == "" || config.ENAgentManifestTag == "" {
			argErr := inv_errors.Errorfc(codes.InvalidArgument, "Missing EN manifest repo or tag")
			zlog.Error().Err(argErr).Msg("")
			return argErr
		}

		enManifestData, err := DownloadENManifest(config.ENManifestRepo, config.ENAgentManifestTag)
		if err != nil {
			return err
		}

		err = yaml.Unmarshal(enManifestData, &config.ENManifest)
		if err != nil {
			return err
		}

		SetInfraConfig(config)

		zlog.Info().Msg("New infra config has been set")

		return nil
	}

	if err := updateConfig(); err != nil {
		return err
	}

	viper.WatchConfig()
	viper.OnConfigChange(func(_ fsnotify.Event) {
		zlog.InfraSec().Info().Msg("Config file change detected, updating config")
		if err := updateConfig(); err != nil {
			zlog.InfraSec().Fatal().Err(err).Msgf("Failed to read new config")
		}
	})

	return nil
}

// DownloadENManifest downloads the Edge Node manifest from the artifact service.
func DownloadENManifest(manifestRepo, manifestTag string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()

	artifacts, err := as.DownloadArtifacts(ctx, manifestRepo, manifestTag)
	if err != nil {
		invErr := inv_errors.Errorf("Error downloading EN Manifest file for tag %s: %s", manifestTag, err)
		zlog.Err(invErr).Msg("")
		return nil, invErr
	}

	if artifacts == nil || len(*artifacts) == 0 {
		invErr := inv_errors.Errorf("Empty artifact data")
		zlog.Err(invErr).Msg("")
		return nil, invErr
	}

	artifact := (*artifacts)[0]
	zlog.InfraSec().Info().Msgf("Downloading artifact %s", artifact.Name)

	return artifact.Data, nil
}

// GetInfraConfig returns the current infrastructure configuration.
func GetInfraConfig() InfraConfig {
	configLock.RLock()
	defer configLock.RUnlock()
	return currentInfraConfig
}

// SetInfraConfig updates the current infrastructure configuration.
func SetInfraConfig(config InfraConfig) {
	zlog.InfraSec().Debug().Msgf("Setting infra configuration: %+v", config)
	zlog.Info().Msgf("Using EN manifest tag: %q", config.ENAgentManifestTag)

	configLock.Lock()
	defer configLock.Unlock()
	currentInfraConfig = config
}
