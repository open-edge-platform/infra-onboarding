/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package commands

import (
	"context"
	"fmt"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/internal/invclient"
	"time"

	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/os/v1"
	"github.com/spf13/cobra"
)

func OsCmds() *cobra.Command {
	var updateSources []string

	osResCmd := &cobra.Command{
		Use:   "os-res",
		Short: "os Resource operations",
		RunE:  printUsage,
	}
	osResCmd.PersistentFlags().StringVar(&dialer.Addr,
		"addr", "", " Address in HOST[:PORT] form (required)")
	must(osResCmd.MarkPersistentFlagRequired("addr"))
	osResCmd.PersistentFlags().BoolVar(&dialer.SkipHostVerification,
		"insecure", false, "Skip host verification")
	osResCmd.PersistentFlags().StringVar(&dialer.CertPath,
		"cert", "", "Path to client certificate file")
	osResCmd.PersistentFlags().StringVar(&dialer.KeyPath,
		"key", "", "Path to client key file")
	osResCmd.PersistentFlags().StringVar(&dialer.CACertPath,
		"cacert", "", "Path to CA certificate bundle")
	osResCmd.MarkFlagsRequiredTogether("cert", "key", "cacert")
	osResCmd.MarkFlagsMutuallyExclusive("insecure", "cert")
	osResCmd.MarkFlagsMutuallyExclusive("insecure", "key")
	osResCmd.MarkFlagsMutuallyExclusive("insecure", "cacert")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an Os Resource",
		RunE:  createOsResource(&dialer),
	}
	createCmd.Flags().StringP("profileName", "p", "", "profileName")
	createCmd.MarkFlagRequired("profileName")
	createCmd.Flags().StringArrayVarP(&updateSources, "update_sources", "u", []string{}, "UpdateSources")
	createCmd.Flags().StringP("repo_url", "l", "", "RepoUrl")
	createCmd.Flags().StringP("sha256", "s", "", "Sha256")
	getIdCmd := &cobra.Command{
		Use:   "getById",
		Short: "Get os resources by resource id",
		RunE:  getByID(&dialer),
	}
	getIdCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	getIdCmd.MarkFlagRequired("resource-id")
	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get Os resources",
		RunE:  getOsResources(&dialer),
	}
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "delete a os resource",
		RunE:  deleteOsResource(&dialer),
	}
	deleteCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	deleteCmd.MarkFlagRequired("resource-id")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "update a os resource",
		RunE:  updateOsResource(&dialer),
	}
	updateCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	updateCmd.MarkFlagRequired("resource-id")
	updateCmd.Flags().StringArrayVarP(&updateSources, "update_sources", "u", []string{}, "UpdateSources")
	updateCmd.Flags().StringP("repo_url", "l", "", "RepoUrl")
	updateCmd.Flags().StringP("sha256", "s", "", "Sha256")
	updateCmd.Flags().StringP("profile_name", "p", "", "ProfileName")
	osResCmd.AddCommand(createCmd, getIdCmd, deleteCmd, updateCmd, getCmd)
	return osResCmd
}

func createOsResource(dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		repoUrl, _ := cmd.Flags().GetString("repo_url")
		sha256, _ := cmd.Flags().GetString("sha256")
		profileName, _ := cmd.Flags().GetString("profileName")
		updateSources, _ := cmd.Flags().GetStringArray("update_sources")

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		osResource := &osv1.OperatingSystemResource{
			UpdateSources: updateSources,
			RepoUrl:       repoUrl,
			Sha256:        sha256,
			ProfileName:   profileName,
		}
		if err := osResource.ValidateAll(); err != nil {
			return err
		}

		_, err = client.CreateOSResource(cmd.Context(), osResource)
		if err != nil {
			return err
		}

		fmt.Printf("OS Details : %+v\n", osResource)
		return nil
	}
}

func getByID(dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		_, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		osRes, err := client.GetOSResourceByResourceID(cmd.Context(), resourceID)
		if err != nil {
			return err
		}

		fmt.Printf("Os Resorce ID: %s, Data: %v\n", osRes.GetResourceId(), osRes)

		return nil
	}
}

func deleteOsResource(dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		_, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		inst, err := client.GetOSResourceByResourceID(cmd.Context(), resourceID)
		if err != nil {
			return err
		}

		err = client.DeleteResource(cmd.Context(), inst.GetResourceId())
		if err != nil {
			return err
		}

		fmt.Printf("Deleted os resource ID: %s\n", resourceID)
		return nil
	}
}

func updateOsResource(dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		_, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		osRes, err := client.GetOSResourceByResourceID(cmd.Context(), resourceID)
		if err != nil {
			return err
		}
		var updateSources []string
		if updatesource, _ := cmd.Flags().GetStringSlice("update_sources"); len(updatesource) > 0 {
			updateSources = updatesource
		}
		osRes.UpdateSources = updateSources
		if repourl, _ := cmd.Flags().GetString("repo_url"); repourl != "" {
			osRes.RepoUrl = repourl
		}
		if sha256, _ := cmd.Flags().GetString("sha256"); sha256 != "" {
			osRes.Sha256 = sha256
		}
		if profilename, _ := cmd.Flags().GetString("profile_name"); profilename != "" {
			osRes.ProfileName = profilename
		}
		err = client.UpdateInvResourceFields(cmd.Context(), osRes, []string{
			osv1.OperatingSystemResourceFieldUpdateSources,
			osv1.OperatingSystemResourceFieldRepoUrl,
			osv1.OperatingSystemResourceFieldSha256,
			osv1.OperatingSystemResourceFieldProfileName,
		})
		if err != nil {
			return err
		}

		fmt.Printf("Updated os resource ID: %s\n", resourceID)
		return nil
	}
}

func getOsResources(dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		_, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		instanceResources, err := client.GetOSResources(cmd.Context())
		if err != nil {
			return err
		}

		for _, instance := range instanceResources {
			fmt.Printf("Os Resources ID: %s, Data: %v\n", instance.GetResourceId(), instance)
		}

		return nil
	}
}
