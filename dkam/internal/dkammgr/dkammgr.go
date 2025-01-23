package dkammgr

import (
	//import dependencies

	"context"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/util"
	"os"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/api/os/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-core/inventory/v2/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/curation"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/download"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/dkam/pkg/signing"
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

	downloaded, downloadErr := download.DownloadMicroOS(ctx)

	if downloadErr != nil {
		zlog.MiSec().Info().Msgf("Failed to download MicroOS %v", downloadErr)
		return downloadErr
	}
	if downloaded {
		zlog.MiSec().Info().Msg("Downloaded successfully")
	}

	return nil
}

func GetCuratedScript(ctx context.Context, os *osv1.OperatingSystemResource) error {
	scriptFileName, err := util.GetInstallerLocation(os, config.PVC)
	if err != nil {
		return err
	}

	installerExists, patherr := util.PathExists(scriptFileName)
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

	signed, err := signing.SignHookOS()
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
	dnsName := GetServerUrl()
	signed, err := signing.BuildSignIpxe(dnsName)
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

func DownloadOS(ctx context.Context, osRes *osv1.OperatingSystemResource) error {
	zlog.Info().Msgf("Inside DownloadOS...")

	if osRes.GetOsProvider() != osv1.OsProviderKind_OS_PROVIDER_KIND_EIM {
		zlog.Debug().Msgf("Skipping OS download for %s due to OS provider kind: %s",
			osRes.GetResourceId(), osRes.GetOsProvider().String())
		return nil
	}
	if osRes.GetOsType() == osv1.OsType_OS_TYPE_IMMUTABLE {
		zlog.Debug().Msgf("Skipping OS download for OS type: %s", osRes.GetOsType())
		return nil
	}

	imageURL := osRes.GetImageUrl()
	zlog.Info().Msgf("imageURL %s", imageURL)
	targetDir := config.PVC

	zlog.Info().Msgf("Download Ubuntu OS")

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

	zlog.MiSec().Info().Msg("OS Image downloaded and move to PVC")
	return nil

}
