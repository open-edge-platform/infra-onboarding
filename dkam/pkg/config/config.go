// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

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
	DefaultTimeout = 3 * time.Second

	DownloadPath           = "/tmp"
	BootsCaCertificateFile = "/etc/ssl/boots-ca-cert/ca.crt"
)

//nolint:tagliatelle // field names must be in line with charts values
type InfraConfig struct {
	ENDebianPackagesRepo string `mapstructure:"enDebianPackagesRepo"`
	ENFilesRsRoot        string `mapstructure:"enFilesRsRoot"`

	ENManifestRepo     string `mapstructure:"enManifestRepo"`
	ENAgentManifestTag string `mapstructure:"enAgentManifestTag"`

	InfraURL                string `mapstructure:"orchInfra"`
	ClusterURL              string `mapstructure:"orchCluster"`
	UpdateURL               string `mapstructure:"orchUpdate"`
	ReleaseServiceURL       string `mapstructure:"orchRelease"`
	LogsObservabilityURL    string `mapstructure:"orchPlatformObsLogs"`
	MetricsObservabilityURL string `mapstructure:"orchPlatformObsMetrics"`
	ManageabilityURL        string `mapstructure:"orchDeviceManager"`
	RPSAddress              string `mapstructure:"orchRPSHost"`
	KeycloakURL             string `mapstructure:"orchKeycloak"`
	TelemetryURL            string `mapstructure:"orchTelemetry"`
	RegistryURL             string `mapstructure:"orchRegistry"`
	FileServerURL           string `mapstructure:"orchFileServer"`
	RSType                  string `mapstructure:"rsType"`

	ProvisioningService string `mapstructure:"provisioningSvc"`
	// ProvisioningServerURL full URL to the provisioning server, including prefixes and subpaths
	ProvisioningServerURL string `mapstructure:"provisioningServerURL"`
	TinkServerURL         string `mapstructure:"tinkerSvc"`
	OnboardingURL         string `mapstructure:"omSvc"`
	OnboardingStreamURL   string `mapstructure:"omStreamSvc"`
	CDN                   string `mapstructure:"cdnSvc"`

	SystemConfigFsInotifyMaxUserInstances uint32 `mapstructure:"systemConfigFsInotifyMaxUserInstances"`
	//nolint:revive,stylecheck // keep the name in sync with charts values
	SystemConfigVmOverCommitMemory uint32 `mapstructure:"systemConfigVmOverCommitMemory"`
	SystemConfigKernelPanicOnOops  uint32 `mapstructure:"systemConfigKernelPanicOnOops"`
	SystemConfigKernelPanic        uint32 `mapstructure:"systemConfigKernelPanic"`

	ENProxyHTTP    string `mapstructure:"enProxyHTTP"`
	ENProxyHTTPS   string `mapstructure:"enProxyHTTPS"`
	ENProxyFTP     string `mapstructure:"enProxyFTP"`
	ENProxyNoProxy string `mapstructure:"enProxyNoProxy"`
	ENProxySocks   string `mapstructure:"enProxySocks"`

	NetIP      string   `mapstructure:"netIp"`
	NTPServers []string `mapstructure:"ntpServer"`
	DNSServers []string `mapstructure:"nameServers"`

	FirewallReqAllow string `mapstructure:"firewallReqAllow"`
	FirewallCfgAllow string `mapstructure:"firewallCfgAllow"`

	ENManifest ENManifest

	EMBImageURL string `mapstructure:"embImageUrl"`

	// Disable AOCO config
	DisableCOProfile   bool `mapstructure:"disableCoProfile" yaml:"disableCoProfile"`
	DisableO11YProfile bool `mapstructure:"disableO11yProfile" yaml:"disableO11yProfile"`
	SkipOSProvisioning bool `mapstructure:"skipOSProvisioning" yaml:"skipOSProvisioning"`
}

// Edge Node Agents release manifest.
type ENManifest struct {
	Repository Repository      `yaml:"repository"`
	Packages   []AgentsVersion `yaml:"packages"`
}

type Repository struct {
	Codename  string `yaml:"codename"`
	Component string `yaml:"component"`
}

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

	PVC                   = "/data"
	OrchCACertificateFile = "/etc/ssl/orch-ca-cert/ca.crt"
	ScriptPath            = "/home/appuser/pkg/script"
)

func Read() error {
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

func GetInfraConfig() InfraConfig {
	configLock.RLock()
	defer configLock.RUnlock()
	return currentInfraConfig
}

func SetInfraConfig(config InfraConfig) {
	zlog.InfraSec().Debug().Msgf("Setting infra configuration: %+v", config)
	zlog.Info().Msgf("Using EN manifest tag: %q", config.ENAgentManifestTag)

	configLock.Lock()
	defer configLock.Unlock()
	currentInfraConfig = config
}
