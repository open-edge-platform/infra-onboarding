package main

import (
	// import dependencies
	"context"
	"io/ioutil"
	"log"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/logging"
	"google.golang.org/grpc"
)

var zlog = logging.GetLogger("MIDKAMAuth")

// createArtifacts function
func GetArtifacts(client pb.DkamServiceClient) {
	log.Println("GetArtifacts.")
	req := &pb.GetArtifactsRequest{ProfileName: "AI", Platform: "Asus"}
	res, err := client.GetArtifacts(context.Background(), req)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error %v", err)
	}
	err = ioutil.WriteFile("manifest.yaml", []byte(res.ManifestFile), 0644)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error %v", err)
	}

	log.Printf("Result: %s", res)
}

func main() {
	//connect to dkam manager
	conn, err := grpc.Dial("localhost:5581", grpc.WithInsecure())
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error %v", err)
	}
	defer conn.Close()

	client := pb.NewDkamServiceClient(conn)
	GetArtifacts(client)

}
