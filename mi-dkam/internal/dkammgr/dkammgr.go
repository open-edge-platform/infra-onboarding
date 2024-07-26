package dkammgr

import (
	//import dependencies

	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/internal/invclient"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/curation"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/download"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/signing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/policy/rbac"
)

type Service struct {
	pb.UnimplementedDkamServiceServer
	invClient   *invclient.DKAMInventoryClient
	rbac        *rbac.Policy
	authEnabled bool
}

var zlog = logging.GetLogger("MIDKAMgRPC")
var url string
var tag = config.Tag

func NewDKAMService(invClient *invclient.DKAMInventoryClient, _ string, _ bool,
	enableAuth bool, rbacRules string,
) (*Service, error) {
	if invClient == nil {
		return nil, errors.New("invClient is nil in DKAMService")
	}

	var rbacPolicy *rbac.Policy
	var err error
	if enableAuth {
		zlog.Info().Msgf("Authentication is enabled, starting RBAC server for DKAMService")
		// start OPA server with policies
		rbacPolicy, err = rbac.New(rbacRules)
		if err != nil {
			zlog.Fatal().Msg("Failed to start RBAC OPA server")
		}
	}

	return &Service{
		invClient:   invClient,
		rbac:        rbacPolicy,
		authEnabled: enableAuth,
	}, nil
}

func DownloadArtifacts() error {
	MODE := GetMODE()
	targetDir := config.DownloadPath
	manifestTag := os.Getenv("MANIFEST_TAG")
	//MODE := "dev"
	zlog.MiSec().Info().Msgf("Mode of deployment: %s", MODE)
	zlog.MiSec().Info().Msgf("Manifest Tag: %s", manifestTag)

	if MODE == "preint" {
		tag = config.Tag
	}
	zlog.MiSec().Info().Msg("Download artifacts")

	err := download.DownloadArtifacts(targetDir, tag, manifestTag)
	if err != nil {
		zlog.MiSec().Info().Msgf("Failed to download manifest file: %v", err)
		return err
	}

	downloaded, downloadErr := download.DownloadMicroOS(targetDir, GetScriptDir())

	if downloadErr != nil {
		zlog.MiSec().Info().Msgf("Failed to download MicroOS %v", downloadErr)
		return downloadErr
	}
	if downloaded {
		zlog.MiSec().Info().Msg("Downloaded successfully")
	}

	return nil
}

func (server *Service) GetENProfile(ctx context.Context, req *pb.GetENProfileRequest) (*pb.GetENProfileResponse, error) {
	//Get request
	profile := req.ProfileName
	platform := req.Platform

	zlog.MiSec().Info().Msgf("Profile Name %s", profile)
	zlog.MiSec().Info().Msgf("Platform %s", platform)

	proxyIP := "http://%host_ip%/tink-stack"
	zlog.MiSec().Info().Msgf("proxyIP %s", proxyIP)
	url = proxyIP + "/" + "installer.sh"
	zlog.MiSec().Info().Msgf("url %s", url)
	osUrl := proxyIP + "/" + config.ImageFileName
	zlog.MiSec().Info().Msgf("osUrl %s", osUrl)
	tinkeraction_version := curation.TinkerAction
	zlog.MiSec().Info().Msgf("tinkeraction_version %s", tinkeraction_version)

	if !PathExists(config.PVC+"/installer.sh") && !PathExists(config.PVC+"/"+config.ImageFileName) {
		zlog.MiSec().Info().Msg("Path exists:")
		zlog.MiSec().Info().Msg("Return Manifest file.")
		return &pb.GetENProfileResponse{StatusCode: true, OsUrl: osUrl, OverlayscriptUrl: url, TinkActionVersion: tinkeraction_version}, nil
	} else {
		zlog.MiSec().Info().Msg("Path not exists:")
		zlog.MiSec().Info().Msg("Return Error Message.")
		return &pb.GetENProfileResponse{StatusCode: true, OsUrl: osUrl, OverlayscriptUrl: url, TinkActionVersion: tinkeraction_version}, nil
	}

}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true // path exists
	}
	if os.IsNotExist(err) {
		return false // path does not exist
	}
	return false // an error occurred (other than not existing)
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

func GetCuratedScript(profile string) error {
	err := curation.GetCuratedScript(profile)
	if err != nil {
		zlog.MiSec().Info().Msgf("Failed curate %v", err)
		return err
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

func DownloadOS(osUrl string) error {
	zlog.Info().Msgf("Inside DownloadOS...")
	imageURL := osUrl
	zlog.Info().Msgf("imageURL %s", imageURL)
	targetDir := config.DownloadPath
	fileName := fileNameFromURL(imageURL)
	rawFileName := strings.TrimSuffix(fileName, ".img") + ".raw.gz"
	file := config.PVC + "/" + rawFileName
	// Check if the compressed raw image file already exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		// Download the image
		if err := download.DownloadUbuntuImage(imageURL, "image.img", rawFileName, targetDir); err != nil {
			zlog.MiSec().Fatal().Err(err).Msgf("Error downloading image:%v", err)
			return err
		}

	} else {
		zlog.MiSec().Info().Msgf("Compressed raw image file already exists: %s", file)
	}

	zlog.MiSec().Info().Msg("Ubuntu OS downloaded and move to PVC")
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
	Tag := config.Tag
	PreintTag := config.PreintTag
	Artifact := config.Artifact
	ImageUrl := config.ImageUrl
	ImageFileName := config.ImageFileName
	RSProxy := config.RSProxy
	RSProxyManifest := config.RSProxyManifest
	OrchCACertificateFile := config.OrchCACertificateFile
	BootsCaCertificateFile := config.BootsCaCertificateFile

	return ServerAddress + "\n" + ServerAddressDescription + "\n" + Port + "\n" + Ubuntuversion + "\n" + Arch + "\n" + Release + "\n" + ProdHarbor + "\n" + DevHarbor + "\n" + AuthServer + "\n" + ReleaseVersion + "\n" + PVC + "\n" + Tag + "\n" + PreintTag + "\n" + Artifact + "\n" + ImageUrl + "\n" + ImageFileName + "\n" + RSProxy + "\n" + RSProxyManifest + "\n" + OrchCACertificateFile + "\n" + BootsCaCertificateFile
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
