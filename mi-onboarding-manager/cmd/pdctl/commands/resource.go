/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/
package commands

import (
	"context"
	"strings"
	"time"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/validator"

	"github.com/spf13/cobra"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/invclient"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	inventoryv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/inventory/v1"
	osv1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/os/v1"
)

var (
	dialer     grpcDialer
	resourceID string
	uuID       string
	fields     string
)

const (
	timeDuration = 5 * time.Second
	tenant1      = "11111111-1111-1111-1111-111111111111"
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

	getIDCmd := &cobra.Command{
		Use:   "getById",
		Short: "Get instance resources by resource id",
		RunE:  getInstanceByID(context.Background(), &dialer),
	}

	getIDCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	_ = getIDCmd.MarkFlagRequired("resource-id")

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
	createCmd.Flags().StringP("osID", "o", "", "os id (required)")
	createCmd.Flags().Uint32("securityFeature", 0, "security Feature (required)")
	must(createCmd.MarkFlagRequired("kind"))
	must(createCmd.MarkFlagRequired("hostID"))
	must(createCmd.MarkFlagRequired("osID"))
	must(createCmd.MarkFlagRequired("securityFeature"))

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "update a instance",
		RunE:  updateInstance(context.Background(), &dialer),
	}
	updateCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	_ = updateCmd.MarkFlagRequired("resource-id")
	updateCmd.Flags().StringVarP(&fields, "fields", "f", "", "fields to update")
	_ = updateCmd.MarkFlagRequired("fields")

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "delete a instance",
		RunE:  deleteInstance(context.Background(), &dialer),
	}
	deleteCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	_ = deleteCmd.MarkFlagRequired("resource-id")

	instanceCmd.AddCommand(getCmd, getIDCmd, createCmd, deleteCmd, updateCmd)

	return instanceCmd
}

func getInstanceResources(ctx context.Context, dialer *grpcDialer) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cc, err := dialer.Dial(cmd.Context())
		if err != nil {
			return err
		}
		defer cc.Close()
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeDuration)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		instanceResources, err := client.GetInstanceResources(ctx)
		if err != nil {
			return err
		}

		for _, instance := range instanceResources {
			zlog.Debug().Msgf("Instance ID: %s, Data: %v\n", instance.GetResourceId(), instance)
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
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeDuration)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		// existing code for getting instance by ID
		inst, err := client.GetInstanceResourceByResourceID(ctx, tenant1, resourceID)
		if err != nil {
			return err
		}

		zlog.Debug().Msgf("Instance ID: %s, Data: %v\n", inst.GetResourceId(), inst)

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
		currentState, _ := cmd.Flags().GetString("current-state")
		vmMemoryBytes, _ := cmd.Flags().GetUint64("vm-memory-bytes")
		vmCPUCores, _ := cmd.Flags().GetUint32("vm-cpu-cores")
		vmStorageBytes, _ := cmd.Flags().GetUint64("vm-storage-bytes")
		hostID, _ := cmd.Flags().GetString("hostID")
		osID, _ := cmd.Flags().GetString("osID")
		securityFeature, _ := cmd.Flags().GetUint32("securityFeature")

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
			// TODO: remove this later https://jira.devtools.intel.com/browse/LPIO-1829
			invclient.WithClientKind(inventoryv1.ClientKind_CLIENT_KIND_API),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		instance := &computev1.InstanceResource{
			Kind:            computev1.InstanceKind(computev1.InstanceKind_value[kind]),
			DesiredState:    computev1.InstanceState_INSTANCE_STATE_RUNNING,
			CurrentState:    computev1.InstanceState(computev1.InstanceState_value[currentState]),
			VmMemoryBytes:   vmMemoryBytes,
			VmCpuCores:      vmCPUCores,
			VmStorageBytes:  vmStorageBytes,
			SecurityFeature: osv1.SecurityFeature(securityFeature),

			Host: &computev1.HostResource{
				ResourceId: hostID,
			},
			DesiredOs: &osv1.OperatingSystemResource{
				ResourceId: osID,
			},
			// Set other fields based on parameters
		}

		// Validate the instance resource
		if validationErr := validator.ValidateMessage(instance); validationErr != nil {
			return validationErr
		}

		_, err = client.CreateInstanceResource(ctx, instance.GetTenantId(), instance)
		if err != nil {
			return err
		}

		zlog.Debug().Msgf("Instance Details: %+v\n", instance)

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
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeDuration)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		// existing code for getting instance by ID
		inst, err := client.GetInstanceResourceByResourceID(ctx, tenant1, resourceID)
		if err != nil {
			return err
		}

		err = client.DeleteInstanceResource(ctx, inst.GetTenantId(), inst.GetResourceId())
		if err != nil {
			return err
		}

		zlog.Debug().Msgf("Deleted Instance ID: %s\n", resourceID)
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
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeDuration)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		// existing code for getting instance by ID
		inst, err := client.GetInstanceResourceByResourceID(ctx, tenant1, resourceID)
		if err != nil {
			return err
		}

		// Sanitizing the input by removing the Instance immutable fields
		fieldSlice := strings.Split(fields, ",")
		for i, v := range fieldSlice {
			if v == computev1.InstanceResourceFieldSecurityFeature {
				fieldSlice = append(fieldSlice[:i], fieldSlice[i+1:]...)
				break
			}
		}

		err = client.UpdateInvResourceFields(ctx, inst.GetTenantId(), inst, fieldSlice)
		if err != nil {
			return err
		}

		zlog.Debug().Msgf("Updated Instance ID: %s, Fields: %s\n", resourceID, fields)
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

	getIDCmd := &cobra.Command{
		Use:   "getById",
		Short: "Get Host resources by resource id",
		RunE:  getResourceByID(context.Background(), &dialer),
	}

	getIDCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	_ = getIDCmd.MarkFlagRequired("resource-id")

	getUUIDCmd := &cobra.Command{
		Use:   "getByUUID",
		Short: "Get host resources by UUID ",
		RunE:  getByuuID(context.Background(), &dialer),
	}

	getUUIDCmd.Flags().StringVarP(&uuID, "uuid", "u", "", " UUID (required)")
	_ = getUUIDCmd.MarkFlagRequired("uuid")

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
	_ = createCmd.MarkFlagRequired("uuid")
	_ = createCmd.MarkFlagRequired("hostname")
	_ = createCmd.MarkFlagRequired("sut-ip")

	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "update a Host",
		RunE:  updateHost(context.Background(), &dialer),
	}
	updateCmd.Flags().StringVarP(&resourceID, "resource-id", "r", "", "Resource ID (required)")
	_ = updateCmd.MarkFlagRequired("resource-id")
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
	_ = deleteCmd.MarkFlagRequired("resource-id")

	hostresCmd.AddCommand(createCmd, getCmd, getIDCmd, getUUIDCmd, deleteCmd, updateCmd)

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
		mgmtIP, _ := cmd.Flags().GetString("sut-ip")
		desiredState, _ := cmd.Flags().GetString("desired-state")
		currentState, _ := cmd.Flags().GetString("current-state")

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
			// TODO: remove this later https://jira.devtools.intel.com/browse/LPIO-1829
			invclient.WithClientKind(inventoryv1.ClientKind_CLIENT_KIND_API),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		hostResource := &computev1.HostResource{
			SerialNumber: serialNumber,
			Uuid:         uuid,
			BmcKind:      computev1.BaremetalControllerKind(computev1.BaremetalControllerKind_value[bmcKind]),
			BmcIp:        bmcIP,
			BmcUsername:  bmcUsername,
			BmcPassword:  bmcPassword,
			PxeMac:       pxeMAC,
			Hostname:     hostname,
			Kind:         kind,
			MgmtIp:       mgmtIP,
			DesiredState: computev1.HostState(computev1.HostState_value[desiredState]),
			CurrentState: computev1.HostState(computev1.HostState_value[currentState]),
		}

		// Validate the host resource
		if validationErr := validator.ValidateMessage(hostResource); validationErr != nil {
			return validationErr
		}

		_, err = client.CreateHostResource(ctx, hostResource.GetTenantId(), hostResource)
		if err != nil {
			return err
		}

		zlog.Debug().Msgf("Host Details: %+v\n", hostResource)

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
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeDuration)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		instanceResources, err := client.GetHostResources(ctx)
		if err != nil {
			return err
		}

		for _, instance := range instanceResources {
			zlog.Debug().Msgf("Instance ID: %s, Data: %v\n", instance.GetResourceId(), instance)
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
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeDuration)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		host, err := client.GetHostResourceByResourceID(ctx, tenant1, resourceID)
		if err != nil {
			return err
		}

		zlog.Debug().Msgf("Host ID: %s, Data: %v\n", host.GetResourceId(), host)

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
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeDuration)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		host, err := client.GetHostResourceByUUID(ctx, uuID)
		if err != nil {
			return err
		}

		zlog.Debug().Msgf("Host ID: %s, Data: %v\n", host.GetResourceId(), host)

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
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeDuration)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		// existing code for getting instance by ID
		host, err := client.GetHostResourceByResourceID(ctx, tenant1, uuID)
		if err != nil {
			return err
		}

		err = client.DeleteHostResource(ctx, host.GetTenantId(), host.GetResourceId())
		if err != nil {
			return err
		}

		zlog.Debug().Msgf("Deleted host ID: %s\n", resourceID)
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
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeDuration)
		defer cancel()

		client, err := invclient.NewOnboardingInventoryClientWithOptions(
			invclient.WithInventoryAddress(dialer.Addr),
		)
		if err != nil {
			return err
		}
		defer client.Close()

		// existing code for getting host by ID
		host, err := client.GetHostResourceByResourceID(ctx, tenant1, resourceID)
		if err != nil {
			return err
		}

		// Can add parameters for updating it
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
		if mgmtIP, _ := cmd.Flags().GetString("sut-ip"); mgmtIP != "" {
			host.MgmtIp = mgmtIP
		}
		if desiredState, _ := cmd.Flags().GetString("desired-state"); desiredState != "" {
			host.DesiredState = computev1.HostState(computev1.HostState_value[desiredState])
		}
		if currentState, _ := cmd.Flags().GetString("current-state"); currentState != "" {
			host.CurrentState = computev1.HostState(computev1.HostState_value[currentState])
		}

		err = client.UpdateHostResource(ctx, host.GetTenantId(), host)
		if err != nil {
			return err
		}

		zlog.Debug().Msgf("Updated Host ID: %s\n", resourceID)
		return nil
	}
}
