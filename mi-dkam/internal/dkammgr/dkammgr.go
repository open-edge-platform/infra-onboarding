package dkammgr

import (
	//import dependencies

	"context"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/client-go/rest"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/config"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/curation"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/download"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/logging"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/signing"
)

type Service struct {
	pb.UnimplementedDkamServiceServer
}

var zlog = logging.GetLogger("MIDKAMgRPC")
var url string

var fileServer = config.ProdFileServer
var harborServer = config.ProdHarbor
var tag = config.Tag

func DownloadArtifacts() error {
	MODE := GetMODE()
	//MODE := "dev"
	zlog.MiSec().Info().Msgf("Mode of deployment: %s", MODE)
	if MODE == "dev" || MODE == "preint" {
		fileServer = config.DevFileServer
		harborServer = config.DevHarbor
	}

	if MODE == "preint" {
		tag = config.Tag
	}
	zlog.MiSec().Info().Msg("Download artifacts")

	_, k8err := rest.InClusterConfig()
	if k8err == nil {
		//Running inside Kubernetes cluster
		zlog.MiSec().Info().Msgf("Running inside k8 cluster")

		err := download.DownloadArtifacts(fileServer, harborServer, GetScriptDir(), tag)
		if err != nil {
			zlog.MiSec().Info().Msgf("Failed to download manifest file: %v", err)
			return err
		}

		downloaded, downloadErr := download.DownloadMicroOS(GetScriptDir())

		if downloadErr != nil {
			zlog.MiSec().Info().Msgf("Failed to download MicroOS %v", downloadErr)
			return downloadErr
		}
		if downloaded {
			zlog.MiSec().Info().Msg("Downloaded successfully")
		}

	} else {
		// Running outside Kubernetes cluster
		zlog.MiSec().Info().Msgf("Running outside k8 cluster")
		zlog.MiSec().Info().Msgf("read local file")
	}

	return nil
}

func (server *Service) GetArtifacts(ctx context.Context, req *pb.GetArtifactsRequest) (*pb.GetArtifactsResponse, error) {
	//Get request
	profile := req.ProfileName
	platform := req.Platform

	zlog.MiSec().Info().Msgf("Profile Name %s", profile)
	zlog.MiSec().Info().Msgf("Platform %s", platform)

	filename := GetCuratedScript(profile, platform)
	scriptName := strings.Split(filename, "/")
	proxyIP := "http://%host_ip%/tink-stack"
	zlog.MiSec().Info().Msgf("proxyIP %s", proxyIP)
	url = proxyIP + "/" + scriptName[len(scriptName)-1]
	zlog.MiSec().Info().Msgf("url %s", url)
	osUrl := proxyIP + "/" + config.ImageFileName
	zlog.MiSec().Info().Msgf("osUrl %s", osUrl)

	zlog.MiSec().Info().Msg("Return Manifest file.")
	return &pb.GetArtifactsResponse{StatusCode: true, OsUrl: osUrl, OverlayscriptUrl: url}, nil
}

func GetCuratedScript(profile string, platform string) string {
	filename := curation.GetCuratedScript(profile, platform)
	return filename
}

func GetServerUrl() string {
	return os.Getenv("DNS_NAME")
}

func SignMicroOS() (bool, error) {
	//MODE := GetMODE()
	scriptPath := GetScriptDir()
	signed, err := signing.SignHookOS(scriptPath)
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
	dnsName := GetServerUrl()
	signed, err := signing.BuildSignIpxe(scriptPath, dnsName)
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

	// Command-line arguments
	imageURL := config.ImageUrl
	targetDir := GetScriptDir()
	fileName := fileNameFromURL(imageURL)
	rawFileName := strings.TrimSuffix(fileName, ".img") + ".raw.gz"
	file := targetDir + "/" + rawFileName

	// Check if the compressed raw image file already exists
	if _, err := os.Stat(file); os.IsNotExist(err) {
		// Download the image
		if err := download.DownloadUbuntuImage(imageURL, "image.img", file, rawFileName); err != nil {
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
