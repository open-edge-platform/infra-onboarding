// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package env

import (
	"os"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

const (
	envHTTPProxy     = "EN_HTTP_PROXY"
	envHTTPSProxy    = "EN_HTTPS_PROXY"
	envNoProxy       = "EN_NO_PROXY"
	envNameservers   = "EN_NAMESERVERS"
	envImageType     = "IMAGE_TYPE"
	envDiskType      = "DISK_PARTITION"
	envImgURL        = "IMG_URL"
	envProvisionerIP = "PD_IP"
	envOverlayURL    = "OVERLAY_URL"
	envFdoMfgDNS     = "FDO_MFG_URL"
	envFdoMfgPort    = "FDO_MFG_PORT"
	envFdoOwnerDNS   = "FDO_OWNER_URL"
	envFdoOwnerPort  = "FDO_OWNER_PORT"
	envK8sNamespace  = "MI_K8S_NAMESPACE"
	envDkamMode      = "EN_DKAMMODE"
	envUserName      = "EN_USERNAME"
	envPassWord      = "EN_PASSWORD"

	defaultOwnerURL     = "mi-fdo-owner"
	defaultOwnerPort    = "58042"
	defaultMfgURL       = "mi-fdo-mfg"
	defaultMfgPort      = "58039"
	defaultK8sNamespace = "maestro-iaas-system"
)

var (
	DiskType           = os.Getenv(envDiskType)
	ImgURL             = os.Getenv(envImgURL)
	ProvisionerIP      = os.Getenv(envProvisionerIP)
	InstallerScriptURL = os.Getenv(envOverlayURL)
	ENProxyHTTP        = os.Getenv(envHTTPProxy)
	ENProxyHTTPS       = os.Getenv(envHTTPSProxy)
	ENProxyNo          = os.Getenv(envNoProxy)
	ENNameservers      = os.Getenv(envNameservers)
	ENDkamMode         = os.Getenv(envDkamMode)
	ENUserName         = os.Getenv(envUserName)
	ENPassWord         = os.Getenv(envPassWord)

	ImgType      = GetEnvWithDefault(envImageType, utils.ImgTypeJammy)
	FdoMfgDNS    = GetEnvWithDefault(envFdoMfgDNS, defaultMfgURL)
	FdoMfgPort   = GetEnvWithDefault(envFdoMfgPort, defaultMfgPort)
	FdoOwnerDNS  = GetEnvWithDefault(envFdoOwnerDNS, defaultOwnerURL)
	FdoOwnerPort = GetEnvWithDefault(envFdoOwnerPort, defaultOwnerPort)
	K8sNamespace = GetEnvWithDefault(envK8sNamespace, defaultK8sNamespace)
)

var zlog = logging.GetLogger("Env")

func GetEnvWithDefault(key, defaultVal string) string {
	v, found := os.LookupEnv(key)
	if found && v != "" {
		return v
	}
	zlog.Warn().Msgf("%s env var is not set, using default image type: %s",
		envImageType, defaultVal)
	return defaultVal
}
