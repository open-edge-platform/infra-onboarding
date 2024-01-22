package dkammgr

import (
	//import dependencies

	"context"
	"os"
	"strings"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/curation"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/logging"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/rest"
)

type Service struct {
	pb.UnimplementedDkamServiceServer
}

type Data struct {
	OsUrl            string
	OverlayScriptUrl string
}

var zlog = logging.GetLogger("MIDKAMgRPC")
var url string

func DownloadArtifacts() error {
	zlog.MiSec().Info().Msg("Download artifacts")
	_, err := rest.InClusterConfig()
	if err == nil {
		// Running inside Kubernetes cluster
		zlog.MiSec().Info().Msgf("Running inside k8 cluster")
		err := curation.DownloadArtifacts()
		if err != nil {
			zlog.MiSec().Info().Msgf("Get File from Local.")
			return err
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
	zlog.MiSec().Info().Msgf("url %s", os.Getenv("ServerUrl"))
	url = GetServerUrl() + "/" + scriptName[len(scriptName)-1]
	zlog.MiSec().Info().Msgf("url %s", url)

	data := GetData()

	yamlContent, err := yaml.Marshal(data)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error... %v", err)
		return nil, err
	}

	zlog.MiSec().Info().Msg("Return Manifest file.")
	return &pb.GetArtifactsResponse{StatusCode: true, ManifestFile: string(yamlContent)}, nil
}

func GetData() Data {
	return Data{
		OsUrl:            "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img",
		OverlayScriptUrl: url,
	}
}

func GetCuratedScript(profile string, platform string) string {
	filename := curation.GetCuratedScript(profile, platform)
	return filename
}

func GetServerUrl() string {
	return os.Getenv("ServerUrl")
}
