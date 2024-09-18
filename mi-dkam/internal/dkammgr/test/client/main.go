package main

import (
	// import dependencies
	"context"

	"google.golang.org/grpc"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/pkg/api/dkammgr/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
)

var zlog = logging.GetLogger("MIDKAMAuth")

// createArtifacts function
func GetArtifacts(client pb.DkamServiceClient) {
	zlog.MiSec().Info().Msg("GetArtifacts.")
	req := &pb.GetENProfileRequest{ProfileName: "TiberOS", Platform: "Asus", Sha256: "de04d58dc5ccc4b9671c3627fb8d626fe4a15810bc1fe3e724feea761965f666", OsType: "OS_TYPE_IMMUTABLE"}
	res, err := client.GetENProfile(context.Background(), req)
	if err != nil {
		zlog.MiSec().Fatal().Err(err).Msgf("Error %v", err)
	}

	zlog.MiSec().Info().Msgf("Result: %s", res)
	zlog.MiSec().Info().Msgf("OS url: %s", res.OsUrl)
	zlog.MiSec().Info().Msgf("Overlay script URL: %s", res.OverlayscriptUrl)
	zlog.MiSec().Info().Msgf("Tinker Actiom version: %s", res.TinkActionVersion)
	zlog.MiSec().Info().Msgf("OS image sha: %s", res.OsImageSha256)

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
