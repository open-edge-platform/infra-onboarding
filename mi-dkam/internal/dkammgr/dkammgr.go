package dkammgr

import (
	//import dependencies

	"context"
	"os"
	"path/filepath"
	"strings"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/curation"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/download"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/signing"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

type Service struct {
	pb.UnimplementedDkamServiceServer
}

var zlog = logging.GetLogger("MIDKAMgRPC")
var url string
var tag = config.Tag

func DownloadArtifacts() error {
	MODE := GetMODE()
	targetDir := config.PVC
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

func (server *Service) GetArtifacts(ctx context.Context, req *pb.GetArtifactsRequest) (*pb.GetArtifactsResponse, error) {
	//Get request
	profile := req.ProfileName
	platform := req.Platform

	zlog.MiSec().Info().Msgf("Profile Name %s", profile)
	zlog.MiSec().Info().Msgf("Platform %s", platform)

	filename, tinkeraction_version := GetCuratedScript(profile, platform)
	scriptName := strings.Split(filename, "/")
	proxyIP := "http://%host_ip%/tink-stack"
	zlog.MiSec().Info().Msgf("proxyIP %s", proxyIP)
	url = proxyIP + "/" + scriptName[len(scriptName)-1]
	zlog.MiSec().Info().Msgf("url %s", url)
	osUrl := proxyIP + "/" + config.ImageFileName
	zlog.MiSec().Info().Msgf("osUrl %s", osUrl)

	zlog.MiSec().Info().Msg("Return Manifest file.")
	return &pb.GetArtifactsResponse{StatusCode: true, OsUrl: osUrl, OverlayscriptUrl: url, TinkActionVersion: tinkeraction_version}, nil
}

func GetCuratedScript(profile string, platform string) (string, string) {
	filename, version := curation.GetCuratedScript(profile, platform)
	return filename, version
}

func GetServerUrl() string {
	return os.Getenv("DNS_NAME")
}

func SignMicroOS() (bool, error) {
	//MODE := GetMODE()
	scriptPath := GetScriptDir()
	targetDir := config.PVC
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
	targetDir := config.PVC
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

func DownloadOS() error {
	zlog.Info().Msgf("Inside DownloadOS...")
	imageURL := config.ImageUrl
	targetDir := config.PVC
	fileName := fileNameFromURL(imageURL)
	rawFileName := strings.TrimSuffix(fileName, ".img") + ".raw.gz"
	file := targetDir + "/" + rawFileName
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
