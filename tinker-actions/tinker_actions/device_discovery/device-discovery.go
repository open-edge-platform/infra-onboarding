package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	pb_om "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"

	"google.golang.org/grpc"
)

func grpcMaestroOnboardNodeJWT(ctx context.Context, address string, port int, mac string, ip string, uuid string, serial string) error {
	target := fmt.Sprintf("%s:%d", address, port)
	fmt.Println("\n address", target)
	conn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithBlock(),
		grpc.WithInsecure(), // Use insecure connection
	)
	if err != nil {
		return fmt.Errorf("could not dial server %s: %v", target, err)
	} else {
		fmt.Println("Connection success")
	}
	defer conn.Close()
	fmt.Println("Dial Complete")
	cli := pb_om.NewNodeArtifactServiceNBClient(conn)
	// Create a NodeData object
	nodeData := &pb_om.NodeData{
		// HwId: "host-e6b916b3",
		Hwdata: []*pb_om.HwData{
			{
				MacId:        mac,
				SutIp:        ip,
				Uuid:         uuid,
				Serialnum:    serial,
				BmcInterface: false,
			},
		},
	}
	// Create a NodeRequest object and set the Payload field
	nodeRequest := &pb_om.NodeRequest{
		Payload: []*pb_om.NodeData{nodeData},
	}
	// Call the gRPC endpoint with the NodeRequest
	if _, err := cli.CreateNodes(ctx, nodeRequest); err != nil {
		return fmt.Errorf("could not call gRPC endpoint for server %s: %v", target, err)
	}
	return nil
}

func main() {
	requiredEnvVars := []string{"OBM_ADDRESS", "MAC_ID"}
	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			fmt.Printf("Environment variable %s is not set.\n", envVar)
			os.Exit(1)
		}
	}
	macAddr := os.Getenv("MAC_ID")
	obm_addr := os.Getenv("OBM_ADDRESS")
	// Split the string into IP and port using ":" as the delimiter
	parts := strings.Split(obm_addr, ":")

	// Ensure there are exactly two parts (IP and port)
	if len(parts) != 2 {
		fmt.Println("Invalid obm_addr format.")
		os.Exit(1)
	}

	// Assign the split parts to respective variables
	obm_ip := parts[0]
	// Convert port to an integer
	obm_port, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Println("Error converting port to integer:", err)
		os.Exit(1)
	}
	// Print the results
	fmt.Printf("OBM IP: %s\n", obm_ip)
	fmt.Printf("OBM Port: %d\n", obm_port)

	// logic to detect ip, serial, and uuid based on mac starts here
	serialNumber, err := getSerialNumber()
	if err != nil {
		fmt.Printf("Error getting serial number: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("Serial Number: %s\n", serialNumber)
	}

	uuid, err := getUUID()
	if err != nil {
		fmt.Printf("Error getting UUID: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("UUID: %s\n", uuid)
	}

	ipAddress, err := getIPAddress(macAddr)
	if err != nil {
		fmt.Printf("Error getting IP address: %v\n", err)
		os.Exit(1)
	} else {
		fmt.Printf("IP Address for MAC %s: %s\n", macAddr, ipAddress)
	}
	// logic to detect ip, serial, and uuid based on mac ends here

	retryDelay := 2 * time.Second // Delay between retries
	for {
		err := grpcMaestroOnboardNodeJWT(context.Background(), obm_ip, obm_port, macAddr, ipAddress, uuid, serialNumber)
		if err == nil {
			fmt.Println("Device discovery done")
			return
		}

		fmt.Printf("There was an error in updating the edge-node details with the onboarding manager, may be caddy is down: %v\n", err)
		fmt.Printf("Retrying update in %v...\n", retryDelay)
		time.Sleep(retryDelay)
	}

}
