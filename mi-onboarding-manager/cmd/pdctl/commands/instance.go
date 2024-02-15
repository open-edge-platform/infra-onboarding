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
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"

	pb "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/api"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func InstanceResourceCmd() *cobra.Command {
	var (
		name         string
		version      string
		platform     string
		category     int
		description  string
		details      *pb.Supplier
		packageURL   string
		author       string
		state        bool
		license      string
		vendor       string
		manufacturer string
		releaseDate  string
		artifactID   string
		inputFile    string
	)

	details = &pb.Supplier{}
	InstanceCmds := &cobra.Command{
		Use:   "instance",
		Short: "Manage all the Instance Resources",
		RunE:  printUsage,
	}

	var dialer grpcDialer

	// Persistent flags
	InstanceCmds.PersistentFlags().StringVar(&dialer.Addr,
		"addr", "", "Inventory Service address in `HOST[:PORT]` form (required)")
	must(InstanceCmds.MarkPersistentFlagRequired("addr"))
	InstanceCmds.PersistentFlags().BoolVar(&dialer.SkipHostVerification,
		"insecure", false, "Skip host verification for Inventory Manager")
	InstanceCmds.PersistentFlags().StringVar(&dialer.CertPath,
		"cert", "", "Path to client certificate file")
	InstanceCmds.PersistentFlags().StringVar(&dialer.KeyPath,
		"key", "", "Path to client key file")
	InstanceCmds.PersistentFlags().StringVar(&dialer.CACertPath,
		"cacert", "", "Path to CA certificate bundle")
	InstanceCmds.MarkFlagsRequiredTogether("cert", "key", "cacert")
	InstanceCmds.MarkFlagsMutuallyExclusive("insecure", "cert")
	InstanceCmds.MarkFlagsMutuallyExclusive("insecure", "key")
	InstanceCmds.MarkFlagsMutuallyExclusive("insecure", "cacert")

	// Create a new FlagSet for artifact flags
	artifactFlags := pflag.NewFlagSet("artifact", pflag.ExitOnError)

	// Define artifact flags
	artifactFlags.StringVar(&name, "name", "", "Name of the Instance")
	artifactFlags.StringVar(&version, "version", "", "Version of the Instance")
	artifactFlags.StringVar(&platform, "platform", "", "Platform of the Instance")
	artifactFlags.IntVar(&category, "category", 0,
		"Category of the artifact (0: DEFAULT, 1: BIOS, 2: OS, 3: APPLICATION, 4: CONTAINER, 5: PLATFORM)")
	artifactFlags.StringVar(&description, "description", "", "Description of the Instance")
	artifactFlags.StringVar(&packageURL, "package_url", "", "URL of the package")
	artifactFlags.StringVar(&author, "author", "", "Author of the package")
	artifactFlags.BoolVar(&state, "state", false, "State of the Instance")
	artifactFlags.StringVar(&license, "license", "", "License information")
	artifactFlags.StringVar(&vendor, "vendor", "", "Vendor details")
	artifactFlags.StringVar(&manufacturer, "manufacturer", "", "Manufacturer details")
	artifactFlags.StringVar(&releaseDate, "release_date", "", "Release date of the Instance")
	artifactFlags.StringVar(&artifactID, "resource_id", "", "Resource ID generated while creating an instance")
	artifactFlags.StringVar(&details.Name, "supplier_name", "", "Name of the supplier")
	artifactFlags.StringVar(&details.Url, "supplier_url", "", "URL of the supplier")
	artifactFlags.StringVar(&details.Contact, "supplier_contact", "", "Contact details of the supplier")
	artifactFlags.StringVar(&inputFile, "input_file", "", "Path to yaml/json file for multiple inputs")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get Instance details",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Dial inventory Manager
			cc, err := dialer.Dial(cmd.Context())
			if err != nil {
				return err
			}
			defer cc.Close()

			// Check if the category value is valid
			if category < 0 || category > 5 {
				return errors.New("invalid category value, must be between 0 and 5")
			}

			categoryEnum := pb.ArtifactData_ArtifactCategory(category)

			// Create the ArtifactData object
			artifact := &pb.ArtifactData{
				Name:        name,
				Version:     version,
				Platform:    platform,
				Category:    categoryEnum,
				Description: description,
				Details: &pb.Supplier{
					Name:    details.Name,
					Url:     details.Url,
					Contact: details.Contact,
				},
				PackageUrl:   packageURL,
				Author:       author,
				State:        state,
				License:      license,
				Vendor:       vendor,
				Manufacturer: manufacturer,
				ReleaseData:  releaseDate,
				ArtifactId:   artifactID,
			}

			data, err := getArtifacts(cmd.Context(), cc, artifact)
			if err != nil {
				if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
					return err
				}

				// If the verbose flag wasn't used, strip gRPC status code info
				// from error
				return errors.New(status.Convert(err).Message())
			}

			fmt.Printf("Payload: %+v\n", data.Payload)
			return nil
		},
	}

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an Instance Resource",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, err := dialer.Dial(cmd.Context())
			if err != nil {
				return err
			}
			defer cc.Close()

			// Check if the category value is valid
			if category < 0 || category > 5 {
				return errors.New("invalid category value, must be between 0 and 5")
			}

			categoryEnum := pb.ArtifactData_ArtifactCategory(category)

			var artifacts []*pb.ArtifactData

			// For multiple inputs.
			if inputFile != "" {
				data, err := os.ReadFile(inputFile)
				if err != nil {
					return err
				}

				var inputArtifacts []*pb.ArtifactData
				if err := yaml.Unmarshal(data, &inputArtifacts); err != nil {
					return err
				}

				for _, currArtifact := range inputArtifacts {
					if currArtifact.Category < 0 || currArtifact.Category > 5 {
						return errors.New("invalid category value, must be between 0 and 5")
					}

					artifacts = append(artifacts, currArtifact)
				}
			} else {
				artifact := &pb.ArtifactData{
					Name:        name,
					Version:     version,
					Platform:    platform,
					Category:    categoryEnum,
					Description: description,
					Details: &pb.Supplier{
						Name:    details.Name,
						Url:     details.Url,
						Contact: details.Contact,
					},
					PackageUrl:   packageURL,
					Author:       author,
					State:        state,
					License:      license,
					Vendor:       vendor,
					Manufacturer: manufacturer,
					ReleaseData:  releaseDate,
					ArtifactId:   artifactID,
				}

				artifacts = append(artifacts, artifact)
			}

			for _, currArtifact := range artifacts {
				data, err := createArtifacts(cmd.Context(), cc, currArtifact)
				if err != nil {
					if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
						return err
					}
					return errors.New(status.Convert(err).Message())
				}

				fmt.Printf("Payload: %+v\n", data.Payload)
			}

			return nil
		},
	}

	updatebyIDCmd := &cobra.Command{
		Use:   "update",
		Short: "Update an Instance by Id",
		RunE: func(cmd *cobra.Command, args []string) error {
			if artifactID == "" {
				return errors.New("artifact_id is required")
			}
			// Dial inventory Manager
			cc, err := dialer.Dial(cmd.Context())
			if err != nil {
				return err
			}
			defer cc.Close()
			// Check if the category value is valid
			if category < 0 || category > 5 {
				return errors.New("invalid category value, must be between 0 and 5")
			}

			categoryEnum := pb.ArtifactData_ArtifactCategory(category)

			// Create the ArtifactData object
			artifact := &pb.ArtifactData{
				Name:        name,
				Version:     version,
				Platform:    platform,
				Category:    categoryEnum,
				Description: description,
				Details: &pb.Supplier{
					Name:    details.Name,
					Url:     details.Url,
					Contact: details.Contact,
				},
				PackageUrl:   packageURL,
				Author:       author,
				State:        state,
				License:      license,
				Vendor:       vendor,
				Manufacturer: manufacturer,
				ReleaseData:  releaseDate,
				ArtifactId:   artifactID,
			}

			data, err := updateArtifactsByID(cmd.Context(), cc, artifactID, artifact)
			if err != nil {
				if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
					return err
				}

				// If the verbose flag wasn't used, strip gRPC status code info
				// from error
				return errors.New(status.Convert(err).Message())
			}

			fmt.Printf("Payload: %+v\n", data.Payload)
			return nil
		},
	}
	// Marking id as required flag.
	updatebyIDCmd.Flags().StringVar(&artifactID, "artifact_id", "", "Artifact ID to get details (required)")
	must(updatebyIDCmd.MarkFlagRequired("artifact_id"))

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an Instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Dial inventory Manager
			cc, err := dialer.Dial(cmd.Context())
			if err != nil {
				return err
			}
			defer cc.Close()
			// Check if the category value is valid
			if category < 0 || category > 5 {
				return errors.New("invalid category value, must be between 0 and 5")
			}

			categoryEnum := pb.ArtifactData_ArtifactCategory(category)

			// Create the ArtifactData object
			artifact := &pb.ArtifactData{
				Name:        name,
				Version:     version,
				Platform:    platform,
				Category:    categoryEnum,
				Description: description,
				Details: &pb.Supplier{
					Name:    details.Name,
					Url:     details.Url,
					Contact: details.Contact,
				},
				PackageUrl:   packageURL,
				Author:       author,
				State:        state,
				License:      license,
				Vendor:       vendor,
				Manufacturer: manufacturer,
				ReleaseData:  releaseDate,
				ArtifactId:   artifactID,
			}

			data, err := deleteArtifacts(cmd.Context(), cc, artifact)
			if err != nil {
				if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
					return err
				}

				// If the verbose flag wasn't used, strip gRPC status code info
				// from error
				return errors.New(status.Convert(err).Message())
			}

			fmt.Printf("Payload: %+v\n", data.Payload)
			return nil
		},
	}

	// Setting the flags for all subcommands.
	getCmd.Flags().AddFlagSet(artifactFlags)
	createCmd.Flags().AddFlagSet(artifactFlags)
	updatebyIDCmd.Flags().AddFlagSet(artifactFlags)
	deleteCmd.Flags().AddFlagSet(artifactFlags)

	InstanceCmds.AddCommand(getCmd, createCmd, updatebyIDCmd, deleteCmd)

	return InstanceCmds
}

type artifactData struct {
	Payload []*pb.ArtifactData
}

func getArtifacts(ctx context.Context, cc *grpc.ClientConn, artifact *pb.ArtifactData) (*artifactData, error) {
	fmt.Println("Getting the Instance...")

	client := pb.NewNodeArtifactServiceNBClient(cc)

	resp, err := client.GetArtifacts(ctx, &pb.ArtifactRequest{
		Payload: []*pb.ArtifactData{artifact},
	})
	if err != nil {
		return nil, err
	}

	data := &artifactData{
		Payload: resp.Payload,
	}

	return data, nil
}

func createArtifacts(ctx context.Context, cc *grpc.ClientConn, artifact *pb.ArtifactData) (*artifactData, error) {
	fmt.Println("Creating the Instance...")

	client := pb.NewNodeArtifactServiceNBClient(cc)

	resp, err := client.CreateArtifacts(ctx, &pb.ArtifactRequest{
		Payload: []*pb.ArtifactData{artifact},
	})
	if err != nil {
		return nil, err
	}

	data := &artifactData{
		Payload: resp.Payload,
	}

	return data, nil
}

func updateArtifactsByID(
	ctx context.Context,
	cc *grpc.ClientConn,
	artifactID string,
	artifact *pb.ArtifactData,
) (*artifactData, error) {
	fmt.Println("Updating the Artifact By ID...")

	client := pb.NewNodeArtifactServiceNBClient(cc)

	artifact.ArtifactId = artifactID

	resp, err := client.UpdateArtifactsById(ctx, &pb.ArtifactRequest{
		Payload: []*pb.ArtifactData{artifact},
	})
	if err != nil {
		return nil, err
	}

	data := &artifactData{
		Payload: resp.Payload,
	}

	return data, nil
}

func deleteArtifacts(ctx context.Context,
	cc *grpc.ClientConn,
	artifact *pb.ArtifactData,
) (*artifactData, error) {
	fmt.Println("Deleting the Instance...")

	client := pb.NewNodeArtifactServiceNBClient(cc)

	resp, err := client.DeleteArtifacts(ctx, &pb.ArtifactRequest{
		Payload: []*pb.ArtifactData{artifact},
	})
	if err != nil {
		return nil, err
	}

	data := &artifactData{
		Payload: resp.Payload,
	}

	return data, nil
}
