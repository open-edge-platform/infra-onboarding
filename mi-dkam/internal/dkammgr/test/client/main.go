package main

import (
	// import dependencies
	"context"

	"google.golang.org/grpc"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
)

var zlog = logging.GetLogger("MIDKAMAuth")

// createArtifacts function
func GetArtifacts(client pb.DkamServiceClient) {
	zlog.MiSec().Info().Msg("GetArtifacts.")
	req := &pb.GetENProfileRequest{ProfileName: "AI", Platform: "Asus"}
	res, err := client.GetENProfile(context.Background(), req)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error %v", err)
	}
	// err = ioutil.WriteFile("manifest.yaml", []byte(res.ManifestFile), 0644)
	// if err != nil {
	// 	zlog.MiSec().Fatal().Err(err).Msgf("Error %v", err)
	// }

	zlog.MiSec().Info().Msgf("Result: %s", res)
	zlog.MiSec().Info().Msgf("OS url: %s", res.OsUrl)
	zlog.MiSec().Info().Msgf("Overlay script URL: %s", res.OverlayscriptUrl)
	zlog.MiSec().Info().Msgf("Tinker Actiom version: %s", res.TinkActionVersion)

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
