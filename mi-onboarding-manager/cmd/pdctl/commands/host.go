/*
 * SPDX-FileCopyrightText: (C) 2023 Intel Corporation
 * SPDX-License-Identifier: LicenseRef-Intel
 */
package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/xeipuuv/gojsonschema"
	"google.golang.org/grpc"

	pbinv "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/api/grpc/onboardingmgr"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type FWinstance struct {
	BIOSVendor      string `yaml:"BIOSVendor" json:"BIOSVendor"`
	BIOSDate        string `json:"BIOSDate" yaml:"BIOSDate"`
	BIOSRelease     string `json:"BIOSRelease" yaml:"BIOSRelease"`
	BIOSVersion     string `yaml:"BIOSVersion" json:"BIOSVersion"`
	BMCManufacturer string `yaml:"BMCManufacturer" json:"BMCManufacturer"`
}
type OSinstance struct {
	Machine      string `json:"Machine"`
	SysName      string `json:"SysName"`
	Release      string `json:"Release"`
	NodeName     string `json:"NodeName"`
	Version      string `json:"Version"`
	DomainName   string `json:"DomainName"`
	OsDistroName string `json:"OsDistroName"`
}
type Imageinstance struct {
	ContainerID            string `json:"ContainerID"`
	ContainerName          string `json:"ContainerName"`
	ContainerRegistryImage string `json:"ContainerRegistryImage"`
}
type NwInterface struct {
	Name    string `json:"Name"`
	Type    string `json:"Type"`
	Address string `json:"Address"`
}
type Hbom struct {
	BaseboardName   string        `json:"BaseboardName"`
	HardwareUUID    string        `json:"HardwareUUID"`
	BMCManufacturer string        `json:"BMCManufacturer"`
	BMCProductName  string        `json:"BMCProductName"`
	RAMtotalMemory  int           `json:"RAMtotalMemory"`
	Nwdevices       []NwInterface `json:"NwDevices"`
	PciDevices      []string      `json:"PciDevices"`
	//CpuCoreInfo     []cpu.CPUInfo `json:"CpuCoreInfo"` // TODO: CPU info IS AVAILABLE in inventory agent.
}

func HostResourceCmd() *cobra.Command {
	hostCmd := &cobra.Command{
		Use:   "host",
		Short: "Manages all Host Resources.",
	}

	var dialer grpcDialer

	// Persistent flags
	hostCmd.PersistentFlags().StringVar(&dialer.Addr,
		"addr", "", "Inventory Service address in `HOST[:PORT]` form (required)")
	must(hostCmd.MarkPersistentFlagRequired("addr"))
	hostCmd.PersistentFlags().BoolVar(&dialer.SkipHostVerification,
		"insecure", false, "Skip host verification")
	hostCmd.PersistentFlags().StringVar(&dialer.CertPath,
		"cert", "", "Path to client certificate file")
	hostCmd.PersistentFlags().StringVar(&dialer.KeyPath,
		"key", "", "Path to client key file")
	hostCmd.PersistentFlags().StringVar(&dialer.CACertPath,
		"cacert", "", "Path to CA certificate bundle")
	hostCmd.MarkFlagsRequiredTogether("cert", "key", "cacert")
	hostCmd.MarkFlagsMutuallyExclusive("insecure", "cert")
	hostCmd.MarkFlagsMutuallyExclusive("insecure", "key")
	hostCmd.MarkFlagsMutuallyExclusive("insecure", "cacert")

	var (
		hwID            string
		platformType    string
		fwArtifactID    string
		osArtifactID    string
		appArtifactID   string
		platArtifactID  string
		deviceType      string
		deviceInfoAgent string
		deviceStatus    string
		updateStatus    string
		updateAvailable string
		nodeId          string
		inputFile       string
	)

	// Create a new FlagSet for addNodeCmd flags
	nodeFlags := pflag.NewFlagSet("node", pflag.ExitOnError)

	// Define addNodeCmd flags
	nodeFlags.StringVar(&hwID, "hw-id", "", "HW ID (required)")
	nodeFlags.StringVar(&nodeId, "host-id", "", "Host ID (required)")
	nodeFlags.StringVar(&platformType, "platform-type", "", "Platform details of the Host (required)")
	nodeFlags.StringVar(&fwArtifactID, "fw-instance-id", "", "Node FW Instance ID")
	nodeFlags.StringVar(&osArtifactID, "os-instance-id", "", "Node OS Instance ID")
	nodeFlags.StringVar(&appArtifactID, "app-instance-id", "", "Node App Instance ID")
	nodeFlags.StringVar(&platArtifactID, "plat-instance-id", "", "Node Platform Instance ID")
	nodeFlags.StringVar(&deviceType, "device-type", "", "Host type (physical, virtual, or container)")
	nodeFlags.StringVar(&deviceInfoAgent, "device-info-agent", "", "Inventory Agent update SBOM & HBOM details during bootup")
	nodeFlags.StringVar(&deviceStatus, "device-status", "", "Device status (READY, UNCLAIMED, etc.)")
	nodeFlags.StringVar(&updateStatus, "update-status", "", "Update status from Update Manager")
	nodeFlags.StringVar(&updateAvailable, "update-available", "", "Update availability status from Update Manager")
	nodeFlags.StringVar(&inputFile, "input_file", "", "Path to yaml/json file for Multiple inputs")

	addNodeCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a Host",
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, err := dialer.Dial(cmd.Context())
			if err != nil {
				return err
			}
			defer cc.Close()

			var nodes []*pbinv.NodeData

			if inputFile != "" {
				data, err := ioutil.ReadFile(inputFile)
				if err != nil {
					return err
				}

				if err := yaml.Unmarshal(data, &nodes); err != nil {
					fmt.Println("Error unmarshaling YAML:", err)
					return err
				}

			} else {
				nodeData := &pbinv.NodeData{
					HwId:            hwID,
					PlatformType:    platformType,
					FwArtifactId:    fwArtifactID,
					OsArtifactId:    osArtifactID,
					AppArtifactId:   appArtifactID,
					PlatArtifactId:  platArtifactID,
					DeviceType:      deviceType,
					DeviceInfoAgent: deviceInfoAgent,
					DeviceStatus:    deviceStatus,
					UpdateStatus:    updateStatus,
					UpdateAvailable: updateAvailable,
					NodeId:          nodeId,
				}

				nodes = append(nodes, nodeData)
			}

			for _, currNode := range nodes {
				resp, err := addNodes(cmd.Context(), cc, currNode)
				if err != nil {
					return err
				}
				// Handle the response data as needed
				fmt.Printf("Added Host: %+v\n", resp.Payload)
			}

			return nil
		},
	}

	getNodeCmd := &cobra.Command{
		Use:   "get",
		Short: "get Host Resource details",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Dial Inventory Manager
			cc, err := dialer.Dial(cmd.Context())
			if err != nil {
				return err
			}
			defer cc.Close()

			// Create the NodeData object from inventorymgr's Protobuf package
			nodeData := &pbinv.NodeData{
				HwId:            hwID,
				PlatformType:    platformType,
				FwArtifactId:    fwArtifactID,
				OsArtifactId:    osArtifactID,
				AppArtifactId:   appArtifactID,
				PlatArtifactId:  platArtifactID,
				DeviceType:      deviceType,
				DeviceInfoAgent: deviceInfoAgent,
				DeviceStatus:    deviceStatus,
				UpdateStatus:    updateStatus,
				UpdateAvailable: updateAvailable,
				NodeId:          nodeId,
			}

			resp, err := getNodes(cmd.Context(), cc, nodeData)
			if err != nil {
				return err
			}

			fwart_schemaData, err := ioutil.ReadFile("../../json_schemas/FwArtifact_schema.json")
			if err != nil {
				log.Fatalf("FwArt: Error reading fw schema file: %v", err)
			}

			fwart_schemaLoader := gojsonschema.NewStringLoader(string(fwart_schemaData))

			if resp.Payload[0].FwArtifactId == "" {
				log.Fatalf("No Data for Fw art ")
			}
			fwart_documentLoader := gojsonschema.NewStringLoader(resp.Payload[0].FwArtifactId)

			resultfwart, err := gojsonschema.Validate(fwart_schemaLoader, fwart_documentLoader)
			if err != nil {
				log.Fatalf("Error validating JSON against fwart schema: %v", err)
			}

			if !resultfwart.Valid() {
				fmt.Println("JSON is not valid against the fwart schema.")
				for _, err := range resultfwart.Errors() {
					fmt.Printf("- %s\n", err)
				}
			}

			var fwartinfo FWinstance
			if err := json.Unmarshal([]byte(resp.Payload[0].FwArtifactId), &fwartinfo); err != nil {
				log.Fatalf("Error unmarshaling fwart JSON: %v", err)
			}

			osart_schemaData, err := ioutil.ReadFile("../../json_schemas/OsArtifact_schema.json")
			if err != nil {
				log.Fatalf("OsArt: Error reading schema file: %v", err)
			}

			osart_schemaLoader := gojsonschema.NewStringLoader(string(osart_schemaData))

			if resp.Payload[0].OsArtifactId == "" {
				log.Fatalf("No Data")
			}
			osart_documentLoader := gojsonschema.NewStringLoader(resp.Payload[0].OsArtifactId)

			resultosart, err := gojsonschema.Validate(osart_schemaLoader, osart_documentLoader)
			if err != nil {
				log.Fatalf("Error validating JSON against osart schema: %v", err)
			}

			if !resultosart.Valid() {
				fmt.Println("JSON is not valid against the osart schema.")
				for _, err := range resultosart.Errors() {
					fmt.Printf("- %s\n", err)
				}
			}
			var osartinfo OSinstance
			if err := json.Unmarshal([]byte(resp.Payload[0].OsArtifactId), &osartinfo); err != nil {
				log.Fatalf("Error unmarshaling osart JSON: %v", err)
			}

			imageart_schemaData, err := ioutil.ReadFile("../../json_schemas/ImageArtifact_schema.json")
			if err != nil {
				log.Fatalf("ImgArt: Error reading image schema file: %v", err)
			}

			imageart_schemaLoader := gojsonschema.NewStringLoader(string(imageart_schemaData))
			//TODO: since a member is not added for Image artifact, AppArtifactId is used, this has to be changed as and when
			//it is added
			if resp.Payload[0].AppArtifactId == "" {
				log.Fatalf("No Data for Image art ")
			}
			imageart_documentLoader := gojsonschema.NewStringLoader(resp.Payload[0].AppArtifactId)

			resultimageart, err := gojsonschema.Validate(imageart_schemaLoader, imageart_documentLoader)
			if err != nil {
				fmt.Println("Error validating JSON against imageart schema:", err)
			}

			if !resultimageart.Valid() {
				fmt.Println("JSON is not valid against the imageart schema.")
				for _, err := range resultimageart.Errors() {
					fmt.Printf("- %s\n", err)
				}
			}

			var imageartinfo []Imageinstance
			if err := json.Unmarshal([]byte(resp.Payload[0].AppArtifactId), &imageartinfo); err != nil {
				fmt.Println("Error unmarshaling imageart JSON:", err)
			}
			schemaData, err := ioutil.ReadFile("../../json_schemas/Hbom_schema.json")
			if err != nil {
				log.Fatalf("Hbomsch: Error reading schema file: %v", err)
			}

			schemaLoader := gojsonschema.NewStringLoader(string(schemaData))
			documentLoader := gojsonschema.NewStringLoader(string(resp.Payload[0].DeviceInfoAgent))

			result, err := gojsonschema.Validate(schemaLoader, documentLoader)
			if err != nil {
				log.Fatalf("Error validating JSON against schema: %v", err)
			}

			if !result.Valid() {
				fmt.Println("JSON is not valid against the schema.")
				for _, err := range result.Errors() {
					fmt.Printf("- %s\n", err)
				}

			}
			var devinfo Hbom
			if err := json.Unmarshal([]byte(resp.Payload[0].DeviceInfoAgent), &devinfo); err != nil {
				log.Fatalf("Error unmarshaling JSON: %v", err)

			}

			// Iterate through the collection and print specific fields from each NodeData
			fmt.Println("--------------------------Host Resource details-----------------------------")
			for _, node := range resp.Payload {
				// Create copies of the fields you intend to use
				hwID := node.HwId
				platformType := node.PlatformType
				platArtifactID := node.PlatArtifactId
				deviceStatus := node.DeviceStatus
				deviceType := node.DeviceType
				updateStatus := node.UpdateStatus
				updateAvailable := node.UpdateAvailable

				// Use the copied fields
				fmt.Printf("HW id: %s\n", hwID)
				fmt.Printf("Platform Type: %s\n", platformType)
				fmt.Printf("-------FW Details-------- \n")
				fmt.Printf("BIOSVendor\t: %s\n", fwartinfo.BIOSVendor)
				fmt.Printf("BIOSDate\t: %s\n", fwartinfo.BIOSDate)
				fmt.Printf("BIOSRelease\t: %s\n", fwartinfo.BIOSRelease)
				fmt.Printf("BIOSVersion\t: %s\n", fwartinfo.BIOSVersion)
				fmt.Printf("BMCManufacturer\t: %s\n", fwartinfo.BMCManufacturer)
				fmt.Printf("-------OS Details-------- \n")
				fmt.Println("Machine\t\t:", osartinfo.Machine)
				fmt.Println("SysName\t\t:", osartinfo.SysName)
				fmt.Println("Release\t\t:", osartinfo.Release)
				fmt.Println("NodeName\t:", osartinfo.NodeName)
				fmt.Println("Version\t\t:", osartinfo.Version)
				fmt.Println("DomainName\t:", osartinfo.DomainName)
				fmt.Println("OsDistroName\t:", osartinfo.OsDistroName)
				fmt.Printf("-------Image Details-------- \n")
				for _, imart := range imageartinfo {
					fmt.Println("ContainerID\t:", imart.ContainerID)
					fmt.Println("ContainerName\t:", imart.ContainerName)
					fmt.Println("ContainerRegistryImage\t:", imart.ContainerRegistryImage)

				}
				fmt.Printf("-------HBOM Details-------- \n")
				fmt.Println("BaseboardName:", devinfo.BaseboardName)
				fmt.Println("HardwareUUID:", devinfo.HardwareUUID)
				fmt.Println("BMCManufacturer:", devinfo.BMCManufacturer)
				fmt.Println("BMCProductName:", devinfo.BMCProductName)
				fmt.Println("RAMtotalMemory:", devinfo.RAMtotalMemory)

				fmt.Printf("-------Host IP Address-------- \n")
				for _, nw := range devinfo.Nwdevices {
					fmt.Printf("Name: %s, Type: %s, Address: %s\n", nw.Name, nw.Type, nw.Address)
				}
				fmt.Printf("-------lspci output-------- \n")
				for _, printPcidev := range devinfo.PciDevices {
					fmt.Printf("%s \n", printPcidev)
				}
				fmt.Printf("-------CPU Info-------- \n")
				/* for _, cpuinfo := range devinfo.CpuCoreInfo {
					  fmt.Printf(" cpu id = %d\n", cpuinfo.ID)
					  fmt.Printf(" cpu family = %s\n", cpuinfo.Family)
					  fmt.Printf(" cpu vendor id = %s\n", cpuinfo.VendorID)
					  fmt.Printf(" cpu Model = %s\n", cpuinfo.Model)
					  fmt.Printf(" cpu ModelName = %s\n", cpuinfo.ModelName)
					  fmt.Printf(" cpu Mhz = %f\n", cpuinfo.Mhz)
				  } */
				fmt.Printf("plat Instance ID: %s\n", platArtifactID)
				fmt.Printf("Device status: %s\n", deviceStatus)
				fmt.Printf("Device Type: %s\n", deviceType)
				fmt.Printf("Update status: %s\n", updateStatus)
				fmt.Printf("Update available: %s\n", updateAvailable)

				fmt.Printf("-----------------------------------------------------------------\n")
			}

			return nil
		},
	}

	updateNodeCmd := &cobra.Command{
		Use:   "update",
		Short: "Update a host",
		RunE: func(cmd *cobra.Command, args []string) error {
			if hwID == "" {
				return errors.New("hw-id is required")
			}
			// Dial Inventory Manager
			cc, err := dialer.Dial(cmd.Context())
			if err != nil {
				return err
			}
			defer cc.Close()

			// Create the NodeData object from inventorymgr's Protobuf package
			nodeData := &pbinv.NodeData{
				HwId:            hwID,
				PlatformType:    platformType,
				FwArtifactId:    fwArtifactID,
				OsArtifactId:    osArtifactID,
				AppArtifactId:   appArtifactID,
				PlatArtifactId:  platArtifactID,
				DeviceType:      deviceType,
				DeviceInfoAgent: deviceInfoAgent,
				DeviceStatus:    deviceStatus,
				UpdateStatus:    updateStatus,
				UpdateAvailable: updateAvailable,
				NodeId:          nodeId,
			}

			resp, err := updateNodes(cmd.Context(), cc, nodeData)
			if err != nil {
				return err
			}

			// Handle the response data as needed
			fmt.Printf("Updating the Host resource : %+v\n", resp.Payload)
			return nil
		},
	}

	delNodeCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a Host",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Dial Inventory Manager
			cc, err := dialer.Dial(cmd.Context())
			if err != nil {
				return err
			}
			defer cc.Close()

			// Parse the flags for node information
			hwID, _ := cmd.Flags().GetString("hw-id")
			platformType, _ := cmd.Flags().GetString("platform-type")
			fwArtifactID, _ := cmd.Flags().GetString("fw-artifact-id")
			osArtifactID, _ := cmd.Flags().GetString("os-artifact-id")
			appArtifactID, _ := cmd.Flags().GetString("app-artifact-id")
			platArtifactID, _ := cmd.Flags().GetString("plat-artifact-id")
			deviceType, _ := cmd.Flags().GetString("device-type")
			deviceInfoAgent, _ := cmd.Flags().GetString("device-info-agent")
			deviceStatus, _ := cmd.Flags().GetString("device-status")
			updateStatus, _ := cmd.Flags().GetString("update-status")
			updateAvailable, _ := cmd.Flags().GetString("update-available")

			// Create the NodeData object from inventorymgr's Protobuf package
			nodeData := &pbinv.NodeData{
				HwId:            hwID,
				PlatformType:    platformType,
				FwArtifactId:    fwArtifactID,
				OsArtifactId:    osArtifactID,
				AppArtifactId:   appArtifactID,
				PlatArtifactId:  platArtifactID,
				DeviceType:      deviceType,
				DeviceInfoAgent: deviceInfoAgent,
				DeviceStatus:    deviceStatus,
				UpdateStatus:    updateStatus,
				UpdateAvailable: updateAvailable,
			}

			resp, err := deleteNodes(cmd.Context(), cc, nodeData)
			if err != nil {
				return err
			}

			// Handle the response data as needed
			fmt.Printf("Deleted the Host resource: %+v\n", resp.Payload)
			return nil
		},
	}

	addNodeCmd.Flags().AddFlagSet(nodeFlags)
	getNodeCmd.Flags().AddFlagSet(nodeFlags)
	updateNodeCmd.Flags().AddFlagSet(nodeFlags)
	delNodeCmd.Flags().AddFlagSet(nodeFlags)

	// Add addNodeCmd to hostCmd
	hostCmd.AddCommand(addNodeCmd, getNodeCmd, updateNodeCmd, delNodeCmd)

	return hostCmd
}

type nodeData struct {
	Payload []*pbinv.NodeData
}

func addNodes(ctx context.Context, cc *grpc.ClientConn, node *pbinv.NodeData) (*nodeData, error) {
	fmt.Println("PDCTL entry point - Add Host...")

	resp, err := pbinv.NewNodeArtifactServiceNBClient(cc).CreateNodes(ctx, &pbinv.NodeRequest{Payload: []*pbinv.NodeData{node}})
	if err != nil {
		return nil, err
	}

	return &nodeData{
		Payload: resp.Payload,
	}, nil
}

func getNodes(ctx context.Context, cc *grpc.ClientConn, node *pbinv.NodeData) (*nodeData, error) {
	fmt.Println("PDCTL entry point - Get Host...")

	resp, err := pbinv.NewNodeArtifactServiceNBClient(cc).GetNodes(ctx, &pbinv.NodeRequest{Payload: []*pbinv.NodeData{node}})
	if err != nil {
		return nil, err
	}

	return &nodeData{
		Payload: resp.Payload,
	}, nil
}

func updateNodes(ctx context.Context, cc *grpc.ClientConn, node *pbinv.NodeData) (*nodeData, error) {
	fmt.Println("PDCTL entry point - Update Host by ID - INV ...")

	resp, err := pbinv.NewNodeArtifactServiceNBClient(cc).UpdateNodes(ctx, &pbinv.NodeRequest{Payload: []*pbinv.NodeData{node}})
	if err != nil {
		return nil, err
	}

	return &nodeData{
		Payload: resp.Payload,
	}, nil
}

func deleteNodes(ctx context.Context, cc *grpc.ClientConn, node *pbinv.NodeData) (*nodeData, error) {
	fmt.Println("PDCTL entry point - Delete host...")

	resp, err := pbinv.NewNodeArtifactServiceNBClient(cc).DeleteNodes(ctx, &pbinv.NodeRequest{Payload: []*pbinv.NodeData{node}})
	if err != nil {
		return nil, err
	}

	return &nodeData{
		Payload: resp.Payload,
	}, nil
}
