/*
SPDX-FileCopyrightText: (C) 2023 Intel Corporation
SPDX-License-Identifier: LicenseRef-Intel
*/
package commands

import (
	"context"
	"errors"
	"fmt"
	"os"

	"google.golang.org/grpc"

	pbinv "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/api"
	"gopkg.in/yaml.v2"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

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
	// CpuCoreInfo     []cpu.CPUInfo `json:"CpuCoreInfo"` // TODO: CPU info IS AVAILABLE in inventory agent.
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
		nodeID          string
		inputFile       string
		mac             string
		sutip           string
		serialnum       string
		uuid            string
		bmcip           string
		bmcintfceval    *bool
		hostNicDevName  string
	)

	// Create a new FlagSet for addNodeCmd flags
	nodeFlags := pflag.NewFlagSet("node", pflag.ExitOnError)

	// Define addNodeCmd flags
	nodeFlags.StringVar(&hwID, "hw-id", "", "HW ID (required)")
	nodeFlags.StringVar(&nodeID, "host-id", "", "Host ID (required)")
	nodeFlags.StringVar(&platformType, "platform-type", "", "Platform details of the Host (required)")
	nodeFlags.StringVar(&fwArtifactID, "fw-instance-id", "", "Node FW Instance ID")
	nodeFlags.StringVar(&osArtifactID, "os-instance-id", "", "Node OS Instance ID")
	nodeFlags.StringVar(&appArtifactID, "app-instance-id", "", "Node App Instance ID")
	nodeFlags.StringVar(&platArtifactID, "plat-instance-id", "", "Node Platform Instance ID")
	nodeFlags.StringVar(&deviceType, "device-type", "", "Host type (physical, virtual, or container)")
	nodeFlags.StringVar(&deviceInfoAgent, "device-info-agent", "",
		"Inventory Agent update SBOM & HBOM details during bootup")
	nodeFlags.StringVar(&deviceStatus, "device-status", "", "Device status (READY, UNCLAIMED, etc.)")
	nodeFlags.StringVar(&updateStatus, "update-status", "", "Update status from Update Manager")
	nodeFlags.StringVar(&updateAvailable, "update-available", "", "Update availability status from Update Manager")
	nodeFlags.StringVar(&inputFile, "input_file", "", "Path to yaml/json file for Multiple inputs")
	nodeFlags.StringVar(&mac, "mac", "", "mac address of the node")
	nodeFlags.StringVar(&sutip, "sutip", "", "sutip address or node ip")
	nodeFlags.StringVar(&serialnum, "serial-number", "", "sutip address or node ip")
	nodeFlags.StringVar(&uuid, "uuid", "", "uuid of the node")
	nodeFlags.StringVar(&bmcip, "bmc-ip", "", "bmc ip")
	bmcintfceval = nodeFlags.Bool("bmc-interface", true, "set bmc interface true/false")
	nodeFlags.StringVar(&hostNicDevName, "host-nic-dev-name", "", "host nic dev name")

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
				data, err := os.ReadFile(inputFile)
				if err != nil {
					return err
				}

				if err := yaml.Unmarshal(data, &nodes); err != nil {
					fmt.Println("Error unmarshaling YAML:", err)
					return err
				}
			} else {
				hwdat := &pbinv.HwData{
					MacId:          mac,
					SutIp:          sutip,
					Serialnum:      serialnum,
					Uuid:           uuid,
					BmcIp:          bmcip,
					BmcInterface:   *bmcintfceval,
					HostNicDevName: hostNicDevName,
				}
				var hwdata []*pbinv.HwData
				hwdata = append(hwdata, hwdat)

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
					NodeId:          nodeID,
					Hwdata:          hwdata,
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
			hwdat := &pbinv.HwData{
				MacId:          mac,
				SutIp:          sutip,
				Serialnum:      serialnum,
				Uuid:           uuid,
				BmcIp:          bmcip,
				BmcInterface:   *bmcintfceval,
				HostNicDevName: hostNicDevName,
			}
			var hwdata []*pbinv.HwData
			hwdata = append(hwdata, hwdat)
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
				NodeId:          nodeID,
				Hwdata:          hwdata,
			}

			resp, err := getNodes(cmd.Context(), cc, nodeData)
			if err != nil {
				return err
			}
			fmt.Printf("%v\n", resp)
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
			hwdat := &pbinv.HwData{
				MacId:          mac,
				SutIp:          sutip,
				Serialnum:      serialnum,
				Uuid:           uuid,
				BmcIp:          bmcip,
				BmcInterface:   *bmcintfceval,
				HostNicDevName: hostNicDevName,
			}
			var hwdata []*pbinv.HwData
			hwdata = append(hwdata, hwdat)
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
				NodeId:          nodeID,
				Hwdata:          hwdata,
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

			hwdat := &pbinv.HwData{
				MacId:          mac,
				SutIp:          sutip,
				Serialnum:      serialnum,
				Uuid:           uuid,
				BmcIp:          bmcip,
				BmcInterface:   *bmcintfceval,
				HostNicDevName: hostNicDevName,
			}
			var hwdata []*pbinv.HwData
			hwdata = append(hwdata, hwdat)
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
				Hwdata:          hwdata,
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

	resp, err := pbinv.NewNodeArtifactServiceNBClient(cc).CreateNodes(ctx,
		&pbinv.NodeRequest{Payload: []*pbinv.NodeData{node}})
	if err != nil {
		return nil, err
	}

	return &nodeData{
		Payload: resp.Payload,
	}, nil
}

func getNodes(ctx context.Context, cc *grpc.ClientConn, node *pbinv.NodeData) (*nodeData, error) {
	fmt.Println("PDCTL entry point - Get Host...")

	resp, err := pbinv.NewNodeArtifactServiceNBClient(cc).GetNodes(ctx,
		&pbinv.NodeRequest{Payload: []*pbinv.NodeData{node}})
	if err != nil {
		return nil, err
	}

	return &nodeData{
		Payload: resp.Payload,
	}, nil
}

func updateNodes(ctx context.Context, cc *grpc.ClientConn, node *pbinv.NodeData) (*nodeData, error) {
	fmt.Println("PDCTL entry point - Update Host by ID - INV ...")

	resp, err := pbinv.NewNodeArtifactServiceNBClient(cc).UpdateNodes(ctx,
		&pbinv.NodeRequest{Payload: []*pbinv.NodeData{node}})
	if err != nil {
		return nil, err
	}

	return &nodeData{
		Payload: resp.Payload,
	}, nil
}

func deleteNodes(ctx context.Context, cc *grpc.ClientConn, node *pbinv.NodeData) (*nodeData, error) {
	fmt.Println("PDCTL entry point - Delete host...")

	resp, err := pbinv.NewNodeArtifactServiceNBClient(cc).DeleteNodes(ctx,
		&pbinv.NodeRequest{Payload: []*pbinv.NodeData{node}})
	if err != nil {
		return nil, err
	}

	return &nodeData{
		Payload: resp.Payload,
	}, nil
}
