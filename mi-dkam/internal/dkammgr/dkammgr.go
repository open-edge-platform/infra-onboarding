package dkammgr

import (
	//import dependencies

	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/util"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/invclient"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/curation"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/download"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/signing"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/policy/rbac"
)

var zlog = logging.GetLogger("DKAM-Mgr")
var file string

func DownloadArtifacts(ctx context.Context) error {
	MODE := GetMODE()
	manifestTag := os.Getenv("MANIFEST_TAG")
	//MODE := "dev"
	zlog.MiSec().Info().Msgf("Mode of deployment: %s", MODE)
	zlog.MiSec().Info().Msgf("Manifest Tag: %s", manifestTag)

	zlog.MiSec().Info().Msg("Download artifacts")

	err := download.DownloadArtifacts(ctx, manifestTag)
	if err != nil {
		zlog.MiSec().Info().Msgf("Failed to download manifest file: %v", err)
		return err
	}

	downloaded, downloadErr := download.DownloadMicroOS(ctx, GetScriptDir())

	if downloadErr != nil {
		zlog.MiSec().Info().Msgf("Failed to download MicroOS %v", downloadErr)
		return downloadErr
	}
	if downloaded {
		zlog.MiSec().Info().Msg("Downloaded successfully")
	}

	return nil
}

func RemoveDir(path string) error {
	if _, err := os.Stat(path); err == nil {
		// Directory exists, remove it
		err := os.RemoveAll(path)
		if err != nil {
			zlog.MiSec().Info().Msg("Error removing directory")
		}
		zlog.MiSec().Info().Msg("Directory removed successfully")
	} else if os.IsNotExist(err) {
		// Directory does not exist, nothing to do
		zlog.MiSec().Info().Msg("Directory does not exist")
	} else {
		// Some other error occurred
		zlog.MiSec().Info().Msg("Error checking directory")
	}
	return nil
}

func GetCuratedScript(ctx context.Context, os *osv1.OperatingSystemResource) error {
	scriptFileName, err := util.GetInstallerLocation(os, config.PVC)
	if err != nil {
		return err
	}

	installerExists, patherr := download.PathExists(scriptFileName)
	if patherr != nil {
		zlog.MiSec().Info().Msgf("Error checking installer file path %v", patherr)
	}
	if installerExists {
		zlog.MiSec().Info().Msg("Installer exists. Skip curation.")
	} else {
		if os.GetOsType() == osv1.OsType_OS_TYPE_MUTABLE {
			err := download.DownloadPrecuratedScript(ctx, os.GetProfileName())
			if err != nil {
				zlog.MiSec().Info().Msgf("Failed to download Profile script: %v", err)
				return err
			}
		}

		err := curation.CurateScript(os)
		if err != nil {
			zlog.MiSec().Info().Msgf("Failed curate %v", err)
			return err
		}

	}
	return nil
}

func GetServerUrl() string {
	return os.Getenv("DNS_NAME")
}

func SignMicroOS() (bool, error) {
	//MODE := GetMODE()
	scriptPath := GetScriptDir()
	targetDir := config.DownloadPath
	signed, err := signing.SignHookOS(scriptPath, targetDir)
	if err != nil {
		zlog.MiSec().Info().Msgf("Failed to sign MicroOS %v", err)
		return false, err
	}
	if signed {
		zlog.MiSec().Info().Msgf("Signed MicroOS and moved to PVC")
	}

	return true, nil
}

func BuildSignIpxe() (bool, error) {
	scriptPath := GetScriptDir()
	targetDir := config.DownloadPath
	dnsName := GetServerUrl()
	signed, err := signing.BuildSignIpxe(targetDir, scriptPath, dnsName)
	if err != nil {
		zlog.MiSec().Info().Msgf("Failed to build and sign iPXE %v", err)
		return false, err
	}
	if signed {
		zlog.MiSec().Info().Msgf("Build, Signed iPXE and moved to PVC")
	}
	return true, nil
}

func GetMODE() string {
	return os.Getenv("MODE")
}

func GetScriptDir() string {
	scriptPath := ""
	currentDir, err := os.Getwd()
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error getting current working directory: %v", err)
		return err.Error()
	}
	zlog.MiSec().Info().Msgf("Current dir %s", currentDir)
	// Navigate two levels up

	path := filepath.Join(currentDir, "pkg", "script")
	_, geterr := os.Stat(path)
	if geterr == nil {
		scriptPath = path
	}
	if os.IsNotExist(geterr) {
		parentDir := filepath.Join(currentDir, "..", "..")
		zlog.MiSec().Info().Msgf("Root dir %s", parentDir)
		scriptPath = filepath.Join(parentDir, "pkg", "script")
	}
	zlog.MiSec().Info().Msgf("scriptPath dir %s", scriptPath)
	return scriptPath
}

func DownloadOS(ctx context.Context, osRes *osv1.OperatingSystemResource) error {
	zlog.Info().Msgf("Inside DownloadOS...")

	if osRes.GetOsProvider() != osv1.OsProviderKind_OS_PROVIDER_KIND_EIM {
		zlog.Debug().Msgf("Skipping OS download for %s due to OS provider kind: %s",
			osRes.GetResourceId(), osRes.GetOsProvider().String())
		return nil
	}

	imageURL := osRes.GetImageUrl()
	zlog.Info().Msgf("imageURL %s", imageURL)
	targetDir := config.PVC
	if osRes.GetOsType() == osv1.OsType_OS_TYPE_IMMUTABLE && osRes.GetOsType() != osv1.OsType_OS_TYPE_UNSPECIFIED {
		zlog.Info().Msgf("Inside Download Tiber OS")

		file = util.GetOSImageLocation(osRes, targetDir)
		// Check if the compressed raw image file already exists
		if _, err := os.Stat(file); os.IsNotExist(err) {
			// Download the image
			if err := download.DownloadTiberOSImage(ctx, osRes, targetDir); err != nil {
				zlog.MiSec().Error().Err(err).Msgf("Error downloading image:%v", err)
				return err
			}

		} else {
			zlog.MiSec().Info().Msgf("Compressed raw image file already exists: %s", file)
		}

	} else {
		zlog.Info().Msgf("Inside Download Ubuntu OS")

		file = util.GetOSImageLocation(osRes, targetDir)
		// Check if the compressed raw image file already exists
		if _, err := os.Stat(file); os.IsNotExist(err) {
			// Download the image
			if err := download.DownloadUbuntuImage(ctx, osRes, targetDir); err != nil {
				zlog.MiSec().Error().Err(err).Msgf("Error downloading image:%v", err)
				return err
			}

		} else {
			zlog.MiSec().Info().Msgf("Compressed raw image file already exists: %s", file)
		}
	}

	zlog.MiSec().Info().Msg("OS Image downloaded and move to PVC")
	return nil

}

// Extract filename from URL
func fileNameFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}

func AccessConfigs() string {
	ServerAddress := config.ServerAddress
	ServerAddressDescription := config.ServerAddressDescription
	Port := config.Port
	Ubuntuversion := config.Ubuntuversion
	Arch := config.Arch
	Release := config.Release
	ProdHarbor := config.ProdHarbor
	DevHarbor := config.DevHarbor
	AuthServer := config.AuthServer
	ReleaseVersion := config.ReleaseVersion
	PVC := config.PVC
	PreintTag := config.PreintTag
	Artifact := config.Artifact
	ImageUrl := config.ImageUrl
	ImageFileName := config.ImageFileName
	RSProxy := config.HookOSRepo
	RSProxyManifest := config.ENManifestRepo
	OrchCACertificateFile := config.OrchCACertificateFile
	BootsCaCertificateFile := config.BootsCaCertificateFile

	return ServerAddress + "\n" + ServerAddressDescription + "\n" + Port + "\n" + Ubuntuversion + "\n" + Arch + "\n" + Release + "\n" + ProdHarbor + "\n" + DevHarbor + "\n" + AuthServer + "\n" + ReleaseVersion + "\n" + PVC + "\n" + PreintTag + "\n" + Artifact + "\n" + ImageUrl + "\n" + ImageFileName + "\n" + RSProxy + "\n" + RSProxyManifest + "\n" + OrchCACertificateFile + "\n" + BootsCaCertificateFile
}

func InitOnboarding(invClient *invclient.DKAMInventoryClient, enableAuth bool, rbacRules string) {
	if invClient == nil {
		zlog.Debug().Msgf("Warning: invClient is nil")
		return
	}

	var err error
	if enableAuth {
		zlog.Info().Msgf("Authentication is enabled, starting RBAC server for DKAM manager")
		// start OPA server with policies
		_, err = rbac.New(rbacRules)
		if err != nil {
			zlog.Fatal().Msg("Failed to start RBAC OPA server")
		}
	}
}
