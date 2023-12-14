package dkammgr

import (
	//import dependencies
	"context"
	"log"
	"gopkg.in/yaml.v2"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	//"fmt"
	//"google.golang.org/grpc"
)


type Service struct {
    pb.UnimplementedDkamServiceServer
}

type Data struct {
 OsUrl string
 OverlayScriptUrl string
}


func (server *Service) GetArtifacts(ctx context.Context, req *pb.GetArtifactsRequest) (*pb.GetArtifactsResponse, error) {
    //Get request
    profile := req.ProfileName
    platform := req.Platform

    log.Println("Profile Name:", profile)
    log.Println("Platform:", platform)

    data := Data{
		OsUrl: "https://af01p-png.devtools.intel.com/artifactory/hspe-edge-png-local/ubuntu-base/20230911-1844/default/ubuntu-22.04-desktop-amd64+intel-iot-37-custom.qcow2.bz2",
		OverlayScriptUrl:"https://ubit-artifactory-sh.intel.com/artifactory/sed-dgn-local/yocto/dev-test-image/DKAM/IAAS/ADL/installer23WW44.4_2148.sh",
	}

	yamlContent, err := yaml.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}

        log.Println("Return Manifest file.")
	return &pb.GetArtifactsResponse{StatusCode: true, ManifestFile: string(yamlContent)}, nil
}
