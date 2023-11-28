package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	pbi "github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	pb "github.com/intel-sandbox/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/provisioningproto"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50054"
)
func generateDevSerial(macID string) (string, error) {
	// Remove colons from the MAC address
	uniqueID := strings.ReplaceAll(macID, ":", "")

	// Generate a random alphanumeric string of length 5
	rand.Seed(time.Now().UnixNano())
	randID := generateRandomString(5)

	// Truncate the uniqueID to remove the first 6 characters
	truncatedID := uniqueID[6:]

	// Concatenate truncatedID and randID to create devSerial
	devSerial := truncatedID + randID

	return devSerial, nil
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}

func OnboardingTest(client pb.OnBoardingEBClient) (*pb.OnboardingResponse, error) {
	var obm pb.OnboardingRequest
	log.Printf("start onboarding")

	devSerial, dev_ser_err := generateDevSerial("1c:69:7a:0e:b2:28")
	if dev_ser_err != nil {
		fmt.Printf("Error: %v\n", dev_ser_err)
		return nil, dev_ser_err
	}
	log.Printf("Device Serial number %s", devSerial)

	obm.Hwdata = append(obm.Hwdata, &pbi.HwData{
		HwId:  devSerial,
		MacId: "",
		SutIp: "",
		CusParams: &pbi.CustomerParams{
			DpsScopeId:          "Add dps scopeid",
			DpsRegistrationId:   "registration id",
			DpsEnrollmentSymKey: "Add DpsEnrollmentSymKey from your azure portal ",
		},
		DiskPartition: "/dev/sda",
		PlatformType:  "prod_focal-ms",
	})

	obm.Hwdata = append(obm.Hwdata, &pbi.HwData{
		HwId:  devSerial,
		MacId: "",
		SutIp: "",
		CusParams: &pbi.CustomerParams{
			DpsScopeId:          "Add dps scopeid",
			DpsRegistrationId:   "registration id",
			DpsEnrollmentSymKey: "Add DpsEnrollmentSymKey from your azure portal ",
		},
		DiskPartition: "/dev/sda",
		PlatformType:  "prod_focal-ms",
	})

	obm.OnbParams = &pbi.OnboardingParams{PdIp: "", PdMac: "", LoadBalancerIp: "", DiskPartition: "/dev/sda", Env: "ZT"}
	// Add other variables in onboarding request
	res, err := client.StartOnboarding(context.Background(), &obm)
	if err != nil {
		log.Fatalf("Failed to get data: %v", err)
		return nil, err
	}
	return res, nil
}

func main() {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Did not connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewOnBoardingEBClient(conn)
	res, err := OnboardingTest(client)
	if err != nil {
		log.Fatalf("Onboarding failed: %v", err)
	}

	log.Printf("Onboarding state: %s", res.Status)
}
