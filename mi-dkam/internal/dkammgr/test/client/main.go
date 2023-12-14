package main

import (
	// import dependencies
	"context"
	"log"
	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.dkam-service/api/grpc/dkammgr"
	"google.golang.org/grpc"
	"io/ioutil"
)

//createArtifacts function
func GetArtifacts(client pb.DkamServiceClient) {
	log.Println("GetArtifacts.")
	req := &pb.GetArtifactsRequest{ProfileName: "AI", Platform: "Asus"}
	res, err := client.GetArtifacts(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed: %d\n", res)
	}
	 err = ioutil.WriteFile("manifest.yaml", []byte(res.ManifestFile), 0644)
        if err != nil {
                log.Printf("Error writing to test.yaml: %v", err)
        }


	log.Printf("Result: %s", res)
}



func main() {
	//connect to dkam manager
	conn, err := grpc.Dial("localhost:5581", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dials: %v", err)
	}
	defer conn.Close()

	client := pb.NewDkamServiceClient(conn)
	GetArtifacts(client)

}
