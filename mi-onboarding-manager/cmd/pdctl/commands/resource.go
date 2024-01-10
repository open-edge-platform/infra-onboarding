/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package commands

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	maestro "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/maestro"
	"github.com/spf13/cobra"
)

var (
	dialer     grpcDialer
	wg         sync.WaitGroup
	resourceID string
	uuID       string
	fields     string
	termchan   bool
)

func InstanceResCmds() *cobra.Command {
	instanceCmd := &cobra.Command{
		Use:   "instance-res",
		Short: "Instance resource operations",
		RunE:  printUsage,
	}

	instanceCmd.PersistentFlags().StringVar(&dialer.Addr,
		"addr", "", " Address in HOST[:PORT] form (required)")
	must(instanceCmd.MarkPersistentFlagRequired("addr"))
	instanceCmd.PersistentFlags().BoolVar(&dialer.SkipHostVerification,
		"insecure", false, "Skip host verification")
	instanceCmd.PersistentFlags().StringVar(&dialer.CertPath,
		"cert", "", "Path to client certificate file")
	instanceCmd.PersistentFlags().StringVar(&dialer.KeyPath,
		"key", "", "Path to client key file")
	instanceCmd.PersistentFlags().StringVar(&dialer.CACertPath,
		"cacert", "", "Path to CA certificate bundle")
	instanceCmd.MarkFlagsRequiredTogether("cert", "key", "cacert")
	instanceCmd.MarkFlagsMutuallyExclusive("insecure", "cert")
	instanceCmd.MarkFlagsMutuallyExclusive("insecure", "key")
	instanceCmd.MarkFlagsMutuallyExclusive("insecure", "cacert")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get instance resources",
		RunE:  getInstanceResources(context.Background(), &dialer),
	}

	getIdCmd := &cobra.Command{
		Use:   "getById",
		Short: "Get instance resources by resource id",
		RunE:  getInstanceByID(context.Background(), &dialer),
	}

	getIdCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	getIdCmd.MarkFlagRequired("resource-id")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new instance",
		RunE:  createInstance(context.Background(), &dialer),
	}

	createCmd.Flags().StringP("kind", "k", "", "Kind of instance (required)")
	createCmd.Flags().StringP("hostID", "y", "", "host id (required)")
	createCmd.Flags().StringP("desired-state", "d", "", "Desired state of the instance")
	createCmd.Flags().StringP("current-state", "c", "", "Current state of the instance")
	createCmd.Flags().Uint64("vm-memory-bytes", 0, "Quantity of memory in the system, in bytes")
	createCmd.Flags().Uint32("vm-cpu-cores", 0, "Number of CPU cores")
	createCmd.Flags().Uint64("vm-storage-bytes", 0, "Storage quantity (primary), in bytes")
	must(createCmd.MarkFlagRequired("kind"))
	must(createCmd.MarkFlagRequired("hostID"))

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "update a instance",
		RunE:  updateInstance(context.Background(), &dialer),
	}
	updateCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	updateCmd.MarkFlagRequired("resource-id")
	updateCmd.Flags().StringVarP(&fields, "fields", "f", "", "fields to update")
	updateCmd.MarkFlagRequired("fields")

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "delete a instance",
		RunE:  deleteInstance(context.Background(), &dialer),
	}
	deleteCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	deleteCmd.MarkFlagRequired("resource-id")

	instanceCmd.AddCommand(getCmd, getIdCmd, createCmd, deleteCmd, updateCmd)

	return instanceCmd
}

func getInstanceResources(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		instanceResources, err := maestro.GetInstanceResources(ctx, client)
		if err != nil {
			return err
		}

		for _, instance := range instanceResources {
			fmt.Printf("Instance ID: %s, Data: %v\n", instance.GetResourceId(), instance)
		}

		return nil
	}
}

func getInstanceByID(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		// existing code for getting instance by ID
		inst, err := maestro.GetInstanceResourceByResourceID(ctx, client, resourceID)
		if err != nil {
			return err
		}

		fmt.Printf("Instance ID: %s, Data: %v\n", inst.GetResourceId(), inst)

		return nil
	}
}

func createInstance(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()

		kind, _ := cmd.Flags().GetString("kind")
		desiredState, _ := cmd.Flags().GetString("desired-state")
		currentState, _ := cmd.Flags().GetString("current-state")
		vmMemoryBytes, _ := cmd.Flags().GetUint64("vm-memory-bytes")
		vmCpuCores, _ := cmd.Flags().GetUint32("vm-cpu-cores")
		vmStorageBytes, _ := cmd.Flags().GetUint64("vm-storage-bytes")
		hostID, _ := cmd.Flags().GetString("hostID")

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		instance := &computev1.InstanceResource{
			Kind:           computev1.InstanceKind(computev1.InstanceKind_value[kind]),
			DesiredState:   computev1.InstanceState(computev1.InstanceState_value[desiredState]),
			CurrentState:   computev1.InstanceState(computev1.InstanceState_value[currentState]),
			VmMemoryBytes:  vmMemoryBytes,
			VmCpuCores:     vmCpuCores,
			VmStorageBytes: vmStorageBytes,

			// Set other fields based on parameters
		}

		// Validate the instance resource
		if err := instance.ValidateAll(); err != nil {
			return err
		}

		instanceID, err := maestro.CreateInstanceResource(ctx, client, instance, hostID)
		if err != nil {
			return err
		}

		fmt.Printf("Created Instance ID: %s\n", instanceID)
		fmt.Printf("Instance Details: %+v\n", instance)

		return nil
	}
}

func deleteInstance(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		// existing code for getting instance by ID
		inst, err := maestro.GetInstanceResourceByResourceID(ctx, client, resourceID)
		if err != nil {
			return err
		}

		err = maestro.DeleteInstanceResource(ctx, client, inst)
		if err != nil {
			return err
		}

		fmt.Printf("Deleted Instance ID: %s\n", resourceID)
		return nil
	}
}

func updateInstance(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		// existing code for getting instance by ID
		inst, err := maestro.GetInstanceResourceByResourceID(ctx, client, resourceID)
		if err != nil {
			return err
		}

		fieldSlice := strings.Split(fields, ",")

		err = maestro.UpdateInvResourceFields(ctx, client, inst, fieldSlice)
		if err != nil {
			return err
		}

		fmt.Printf("Updated Instance ID: %s, Fields: %s\n", resourceID, fields)
		return nil
	}
}

func HostResCmds() *cobra.Command {
	hostresCmd := &cobra.Command{
		Use:   "host-res",
		Short: "host Resource operations",
		RunE:  printUsage,
	}

	hostresCmd.PersistentFlags().StringVar(&dialer.Addr,
		"addr", "", " Address in HOST[:PORT] form (required)")
	must(hostresCmd.MarkPersistentFlagRequired("addr"))
	hostresCmd.PersistentFlags().BoolVar(&dialer.SkipHostVerification,
		"insecure", false, "Skip host verification")
	hostresCmd.PersistentFlags().StringVar(&dialer.CertPath,
		"cert", "", "Path to client certificate file")
	hostresCmd.PersistentFlags().StringVar(&dialer.KeyPath,
		"key", "", "Path to client key file")
	hostresCmd.PersistentFlags().StringVar(&dialer.CACertPath,
		"cacert", "", "Path to CA certificate bundle")
	hostresCmd.MarkFlagsRequiredTogether("cert", "key", "cacert")
	hostresCmd.MarkFlagsMutuallyExclusive("insecure", "cert")
	hostresCmd.MarkFlagsMutuallyExclusive("insecure", "key")
	hostresCmd.MarkFlagsMutuallyExclusive("insecure", "cacert")

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get Host resources",
		RunE:  getHostResources(context.Background(), &dialer),
	}
	getCmd.Flags().StringP("kind", "k", "", "Kind of instance")

	getIdCmd := &cobra.Command{
		Use:   "getById",
		Short: "Get Host resources by resource id",
		RunE:  getResourceByID(context.Background(), &dialer),
	}

	getIdCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	getIdCmd.MarkFlagRequired("resource-id")

	getUuidCmd := &cobra.Command{
		Use:   "getByUUID",
		Short: "Get host resources by UUID ",
		RunE:  getByuuID(context.Background(), &dialer),
	}

	getUuidCmd.Flags().StringVarP(&uuID, "uuid", "u", "", " UUID (required)")
	getUuidCmd.MarkFlagRequired("uuid")

	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new host",
		RunE:  createResource(context.Background(), &dialer),
	}

	createCmd.Flags().StringP("bmc-kind", "b", "", "BMC Kind")
	createCmd.Flags().StringP("bmc-ip", "i", "", "BMC IP")
	createCmd.Flags().StringP("bmc-username", "u", "", "BMC Username")
	createCmd.Flags().StringP("bmc-password", "m", "", "BMC Password")
	createCmd.Flags().StringP("pxe-mac", "x", "", "PXE MAC")
	createCmd.Flags().StringP("hostname", "w", "", "Hostname")
	createCmd.Flags().StringP("kind", "k", "", "Kind of instance")
	createCmd.Flags().StringP("uuid", "d", "", "UUID (required)")
	createCmd.Flags().StringP("serial-number", "s", "", "serial number (required)")
	createCmd.Flags().StringP("sut-ip", "t", "", "Sut-ip")
	createCmd.Flags().StringP("desired-state", "e", "", "Desired state of the host")
	createCmd.Flags().StringP("current-state", "c", "", "Current state of the host")
	createCmd.MarkFlagRequired("uuid")
	createCmd.MarkFlagRequired("hostname")
	createCmd.MarkFlagRequired("sut-ip")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "update a Host",
		RunE:  updateHost(context.Background(), &dialer),
	}
	updateCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	updateCmd.MarkFlagRequired("resource-id")
	updateCmd.Flags().StringP("bmc-kind", "b", "", "BMC Kind")
	updateCmd.Flags().StringP("bmc-ip", "i", "", "BMC IP")
	updateCmd.Flags().StringP("bmc-username", "u", "", "BMC Username")
	updateCmd.Flags().StringP("bmc-password", "m", "", "BMC Password")
	updateCmd.Flags().StringP("pxe-mac", "x", "", "PXE MAC")
	updateCmd.Flags().StringP("hostname", "w", "", "Hostname")
	updateCmd.Flags().StringP("kind", "k", "", "Kind of instance")
	updateCmd.Flags().StringP("current-state", "c", "", "Current state")
	updateCmd.Flags().StringP("desired-state", "d", "", "desired state")

	updateCmd.Flags().StringP("sut-ip", "t", "", "Sut ip")

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "delete a host",
		RunE:  deleteHost(context.Background(), &dialer),
	}
	deleteCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	deleteCmd.MarkFlagRequired("resource-id")

	hostresCmd.AddCommand(createCmd, getCmd, getIdCmd, getUuidCmd, deleteCmd, updateCmd)

	return hostresCmd
}

func createResource(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()

		bmcKind, _ := cmd.Flags().GetString("bmc-kind")
		bmcIP, _ := cmd.Flags().GetString("bmc-ip")
		bmcUsername, _ := cmd.Flags().GetString("bmc-username")
		bmcPassword, _ := cmd.Flags().GetString("bmc-password")
		pxeMAC, _ := cmd.Flags().GetString("pxe-mac")
		hostname, _ := cmd.Flags().GetString("hostname")
		kind, _ := cmd.Flags().GetString("kind")
		uuid, _ := cmd.Flags().GetString("uuid")
		serialNumber, _ := cmd.Flags().GetString("serial-number")
		mgmtIp, _ := cmd.Flags().GetString("sut-ip")
		desiredState, _ := cmd.Flags().GetString("desired-state")
		currentState, _ := cmd.Flags().GetString("current-state")

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		hostResource := &computev1.HostResource{
			SerialNumber: serialNumber,
			BmcKind:      computev1.BaremetalControllerKind(computev1.BaremetalControllerKind_value[bmcKind]),
			BmcIp:        bmcIP,
			BmcUsername:  bmcUsername,
			BmcPassword:  bmcPassword,
			PxeMac:       pxeMAC,
			Hostname:     hostname,
			Kind:         kind,
			MgmtIp:       mgmtIp,
			DesiredState: computev1.HostState(computev1.HostState_value[desiredState]),
			CurrentState: computev1.HostState(computev1.HostState_value[currentState]),
		}

		// Validate the host resource
		if err := hostResource.ValidateAll(); err != nil {
			return err
		}

		hostID, err := maestro.CreateHostResource(ctx, client, uuid, hostResource)

		if err != nil {
			return err
		}

		fmt.Printf("Created Host ID: %s\n", hostID)
		fmt.Printf("Host Details: %+v\n", hostResource)

		return nil
	}
}

func getHostResources(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		instanceResources, err := maestro.GetHostResources(ctx, client)
		if err != nil {
			return err
		}

		for _, instance := range instanceResources {
			fmt.Printf("Instance ID: %s, Data: %v\n", instance.GetResourceId(), instance)
		}

		return nil
	}
}
func getResourceByID(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		host, err := maestro.GetHostResourceByResourceID(ctx, client, resourceID)
		if err != nil {
			return err
		}

		fmt.Printf("Host ID: %s, Data: %v\n", host.GetResourceId(), host)

		return nil
	}
}

func getByuuID(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		host, err := maestro.GetHostResourceByUUID(ctx, client, uuID)
		if err != nil {
			return err
		}

		fmt.Printf("Host ID: %s, Data: %v\n", host.GetResourceId(), host)

		return nil
	}
}

func deleteHost(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		// existing code for getting instance by ID
		inst, err := maestro.GetHostResourceByResourceID(ctx, client, resourceID)
		if err != nil {
			return err
		}

		err = maestro.DeleteHostResource(ctx, client, inst)
		if err != nil {
			return err
		}

		fmt.Printf("Deleted host ID: %s\n", resourceID)
		return nil
	}
}

func updateHost(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		client, eventCh, err := maestro.NewInventoryClient(&wg, dialer.Addr)
		if err != nil {
			return err
		}
		defer close(eventCh)

		// existing code for getting host by ID
		host, err := maestro.GetHostResourceByResourceID(ctx, client, resourceID)
		if err != nil {
			return err
		}

		//Can add parameters for updating it
		if bmcKind, _ := cmd.Flags().GetString("bmc-kind"); bmcKind != "" {
			host.BmcKind = computev1.BaremetalControllerKind(computev1.BaremetalControllerKind_value[bmcKind])
		}
		if bmcIP, _ := cmd.Flags().GetString("bmc-ip"); bmcIP != "" {
			host.BmcIp = bmcIP
		}
		if bmcUsername, _ := cmd.Flags().GetString("bmc-username"); bmcUsername != "" {
			host.BmcUsername = bmcUsername
		}
		if bmcPassword, _ := cmd.Flags().GetString("bmc-password"); bmcPassword != "" {
			host.BmcPassword = bmcPassword
		}
		if pxeMAC, _ := cmd.Flags().GetString("pxe-mac"); pxeMAC != "" {
			host.PxeMac = pxeMAC
		}
		if hostname, _ := cmd.Flags().GetString("hostname"); hostname != "" {
			host.Hostname = hostname
		}
		if mgmtIp, _ := cmd.Flags().GetString("sut-ip"); mgmtIp != "" {
			host.MgmtIp = mgmtIp
		}
		if desiredState, _ := cmd.Flags().GetString("desired-state"); desiredState != "" {
			host.DesiredState = computev1.HostState(computev1.HostState_value[desiredState])
		}
		if currentState, _ := cmd.Flags().GetString("current-state"); currentState != "" {
			host.CurrentState = computev1.HostState(computev1.HostState_value[currentState])
		}

		err = maestro.UpdateHostResource(ctx, client, host)
		if err != nil {
			return err
		}

		fmt.Printf("Updated Host ID: %s\n", resourceID)
		return nil
	}
}
