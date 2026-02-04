// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
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
	// BootsCaCertificateFile is the path to the CA certificate file.
	BootsCaCertificateFile = "/etc/ssl/boots-ca-cert/ca.crt"
)

// InfraConfig holds the infrastructure configuration settings.
type InfraConfig struct {
	ENDebianPackagesRepo string `mapstructure:"en-debian-packages-repo"`
	ENFilesRsRoot        string `mapstructure:"en-files-rs-root"`

	ENManifestRepo     string `mapstructure:"en-manifest-repo"`
	ENAgentManifestTag string `mapstructure:"en-agent-manifest-tag"`

	InfraURL                string `mapstructure:"orch-infra"`
	ClusterURL              string `mapstructure:"orch-cluster"`
	UpdateURL               string `mapstructure:"orch-update"`
	ReleaseServiceURL       string `mapstructure:"orch-release"`
	LogsObservabilityURL    string `mapstructure:"orch-platform-obs-logs"`
	MetricsObservabilityURL string `mapstructure:"orch-platform-obs-metrics"`
	ManageabilityURL        string `mapstructure:"orch-device-manager"`
	RPSAddress              string `mapstructure:"orch-rps-host"`
	KeycloakURL             string `mapstructure:"orch-keycloak"`
	TelemetryURL            string `mapstructure:"orch-telemetry"`
	RegistryURL             string `mapstructure:"orch-registry"`
	FileServerURL           string `mapstructure:"orch-file-server"`
	RSType                  string `mapstructure:"rs-type"`

	ProvisioningService string `mapstructure:"provisioning-svc"`
	// ProvisioningServerURL full URL to the provisioning server, including prefixes and subpaths
	ProvisioningServerURL string `mapstructure:"provisioning-server-url"`
	TinkServerURL         string `mapstructure:"tinker-svc"`
	OnboardingURL         string `mapstructure:"om-svc"`
	OnboardingStreamURL   string `mapstructure:"om-stream-svc"`
	CDN                   string `mapstructure:"cdn-svc"`

	SystemConfigFsInotifyMaxUserInstances uint32 `mapstructure:"system-config-fs-inotify-max-user-instances"`
	//nolint:revive,stylecheck // keep the name in sync with charts values
	SystemConfigVmOverCommitMemory uint32 `mapstructure:"system-config-vm-over-commit-memory"`
	SystemConfigKernelPanicOnOops  uint32 `mapstructure:"system-config-kernel-panic-on-oops"`
	SystemConfigKernelPanic        uint32 `mapstructure:"system-config-kernel-panic"`

	ENProxyHTTP    string `mapstructure:"en-proxy-http"`
	ENProxyHTTPS   string `mapstructure:"en-proxy-https"`
	ENProxyFTP     string `mapstructure:"en-proxy-ftp"`
	ENProxyNoProxy string `mapstructure:"en-proxy-no-proxy"`
	ENProxySocks   string `mapstructure:"en-proxy-socks"`

	NetIP      string   `mapstructure:"net-ip"`
	NTPServers []string `mapstructure:"ntp-server"`
	DNSServers []string `mapstructure:"name-servers"`

	FirewallReqAllow string `mapstructure:"firewall-req-allow"`
	FirewallCfgAllow string `mapstructure:"firewall-cfg-allow"`

	ENManifest ENManifest

	EMBImageURL string `mapstructure:"emb-image-url"`

	// Disable AOCO config
	DisableCOProfile   bool `mapstructure:"disable-co-profile" yaml:"disable_co_profile"`
	DisableO11YProfile bool `mapstructure:"disable-o11y-profile" yaml:"disable_o11y_profile"`
	SkipOSProvisioning bool `mapstructure:"skip-os-provisioning" yaml:"skip_os_provisioning"`
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

	PVC                   = "/data"
	OrchCACertificateFile = "/etc/ssl/orch-ca-cert/ca.crt"
	ScriptPath            = "/home/appuser/pkg/script"
)

// Read reads and validates the configuration from the config file.
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
