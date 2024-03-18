/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/auth"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/fdoclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/tinkerbell"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/util"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/api/compute/v1"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/status"
)

// TODO (LPIO-1863): avoid hardcoding.
const (
	k8sNamespace = "maestro-iaas-system"

	hwPrefixName       = "machine-"
	workFlowPrefixName = "workflow-"

	//nolint:lll // keep long line for better readability
	sviInfoPayload = `[{"filedesc" : "client_id","resource" : "$(guid)_%s"},{"filedesc" : "client_secret","resource" : "$(guid)_%s"}]`
)

var (
	clientName = "Workflow"
	zlog       = logging.GetLogger(clientName)
)

//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
type ResponseData struct {
	To2CompletedOn string `json:"to2CompletedOn"`
	To0Expiry      string `json:"to0Expiry"`
}

func checkTO2StatusCompleted(_ context.Context, deviceInfo utils.DeviceInfo) (bool, error) {
	// Make an HTTP GET request
	to2URL := fmt.Sprintf("http://%s:%s/api/v1/owner/state/%s",
		deviceInfo.FdoOwnerDNS, deviceInfo.FdoOwnerPort, deviceInfo.FdoGUID)
	response, err := http.NewRequestWithContext(context.Background(), http.MethodGet, to2URL, http.NoBody)
	if err != nil {
		respErr := inv_errors.Errorf("Error making HTTP GET request %v", err)
		zlog.MiSec().MiErr(err).Msgf("")
		return false, respErr
	}
	httpClient := &http.Client{}
	resp, err := httpClient.Do(response)
	if err != nil {
		respErr := inv_errors.Errorf("Error making HTTP GET request %v", err)
		zlog.MiSec().MiErr(err).Msgf("")
		return false, respErr
	}
	defer resp.Body.Close()
	// Read response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		respErr := inv_errors.Errorf("Error while reading resp body %v", err)
		zlog.MiSec().MiErr(err).Msgf("")
		return false, respErr
	}

	if len(body) == 0 {
		zlog.Debug().Msgf("Empty TO2 status response received for host %s", deviceInfo.GUID)
		// in progress, let other FDO actions to be performed. The upper layer will return IN_PROGRESS error.
		return false, nil
	}

	// Unmarshal the JSON response
	responseData := ResponseData{}
	if jsonErr := json.Unmarshal(body, &responseData); jsonErr != nil {
		zlog.MiSec().Err(jsonErr).Msgf("")
		return false, inv_errors.Errorf("Failed to perform GET request to %s for host %s",
			deviceInfo.ProvisionerIP, deviceInfo.GUID)
	}

	if responseData.To2CompletedOn == "" {
		zlog.Debug().Msgf("TO2 process not completed yet for host %s", deviceInfo.GUID)
		// not completed yet
		return false, nil
	}

	zlog.Debug().Msgf("TO2 has completed: %s", responseData.To2CompletedOn)

	return true, nil
}

func CheckStatusOrRunProdWorkflow(ctx context.Context,
	deviceInfo utils.DeviceInfo,
	instance *computev1.InstanceResource,
) error {
	zlog.Info().Msgf("Checking status of Prod workflow for host %s", deviceInfo.GUID)

	kubeClient, err := tinkerbell.K8sClientFactory()
	if err != nil {
		return err
	}

	util.PopulateHostStatus(instance,
		computev1.HostStatus_HOST_STATUS_PROVISIONING,
		"", // no details as workflow is not started yet
		om_status.OnboardingStatusInProgress)
	// NOTE: this is not used by UI as for now, but update status for future use.
	util.PopulateInstanceStatusAndCurrentState(
		instance,
		instance.GetCurrentState(),
		computev1.InstanceStatus_INSTANCE_STATUS_PROVISIONING,
		om_status.ProvisioningStatusInProgress)

	prodWorkflowName := fmt.Sprintf("workflow-%s-prod", deviceInfo.GUID)
	workflow, err := getWorkflow(ctx, kubeClient, prodWorkflowName)
	if err != nil && inv_errors.IsNotFound(err) {
		// This may happen if:
		// 1) workflow for Instance is not created yet -> proceed to runProdWorkflow()
		// 2) we already finished & removed workflow for Instance -> in this case we should never get here
		runErr := runProdWorkflow(ctx, kubeClient, deviceInfo)
		if runErr != nil {
			return runErr
		}

		// runProdWorkflow returned no error, but we return an error here so that the upper layer can handle it appropriately
		// and reconcile until the workflow is finished.
		return inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "Prod workflow started, waiting for it to complete")
	}

	if err != nil {
		// some unexpected error, we fail to get workflow status
		return err
	}

	return handleWorkflowStatus(instance, workflow,
		computev1.HostStatus_HOST_STATUS_PROVISIONED, computev1.HostStatus_HOST_STATUS_PROVISION_FAILED,
		om_status.ProvisioningStatusDone, om_status.ProvisioningStatusFailed)
}

func runProdWorkflow(ctx context.Context, k8sCli client.Client, deviceInfo utils.DeviceInfo) error {
	zlog.Info().Msgf("Creating prod workflow for host %s", deviceInfo.GUID)

	if !*common.FlagEnableDeviceInitialization {
		// normally EN credentials should be created as part of the device initialization phase,
		// but we have to do it here if the DI phase is disabled.
		clientID, clientSecret, err := createENCredentialsIfNotExists(ctx, deviceInfo)
		if err != nil {
			return err
		}
		// TODO: to be removed once DI is always enabled.
		deviceInfo.ClientID = clientID
		deviceInfo.ClientSecret = clientSecret
	}

	// NOTE: IMO (Tomasz) this is a one-time operation that should be done when a host is discovered and created.
	// So it shouldn't be here (move to host reconciler?)
	if createHwErr := tinkerbell.CreateHardwareIfNotExists(ctx, k8sCli, k8sNamespace, deviceInfo); createHwErr != nil {
		return createHwErr
	}

	prodTemplate, err := tinkerbell.GenerateTemplateForProd(k8sNamespace, deviceInfo)
	if err != nil {
		zlog.MiErr(err).Msg("")
		return inv_errors.Errorf("Failed to generate Tinkerbell prod template for host %s", deviceInfo.GUID)
	}

	if createTemplErr := tinkerbell.CreateTemplateIfNotExists(ctx, k8sCli, prodTemplate); createTemplErr != nil {
		return createTemplErr
	}

	prodWorkflow := tinkerbell.NewWorkflow(
		fmt.Sprintf("workflow-%s-prod", deviceInfo.GUID),
		k8sNamespace,
		deviceInfo.HwMacID,
		hwPrefixName+deviceInfo.GUID,
		fmt.Sprintf("%s-%s-prod", deviceInfo.ImType, deviceInfo.GUID))

	if createWFErr := tinkerbell.CreateWorkflowIfNotExists(ctx, k8sCli, prodWorkflow); createWFErr != nil {
		return createWFErr
	}

	zlog.Debug().Msgf("Prod workflow %s for host %s created successfully", prodWorkflow.Name, deviceInfo.GUID)

	return nil
}

// CheckTO2StatusOrRunFDOActions checks if TO2 protocol is completed.
// If it's already completed, this function returns immediately.
// If it's not yet completed, it runs a series of actions to ensure TO2 gets completed.
func CheckTO2StatusOrRunFDOActions(ctx context.Context,
	deviceInfo utils.DeviceInfo, instance *computev1.InstanceResource,
) error {
	if !*common.FlagEnableDeviceInitialization {
		zlog.Warn().Msgf("enableDeviceInitialization is set to false, skipping FDO actions")
		return nil
	}

	zlog.Info().Msgf("Checking TO2 status completed for host %s", deviceInfo.GUID)

	util.PopulateHostStatus(instance, computev1.HostStatus_HOST_STATUS_ONBOARDING,
		"",
		om_status.OnboardingStatusInProgress)

	// we need to upload voucher extension before checking TO2 status to get FDO GUID that is needed for TO2 status check.
	fdoGUID, err := uploadFDOVoucherScript(ctx, deviceInfo)
	if err != nil {
		return err
	}
	deviceInfo.FdoGUID = fdoGUID

	isCompleted, err := checkTO2StatusCompleted(ctx, deviceInfo)
	if err != nil {
		return err
	}

	if isCompleted {
		zlog.Debug().Msgf("TO2 process completed for host %s", deviceInfo.GUID)
		util.PopulateHostStatus(instance,
			computev1.HostStatus_HOST_STATUS_ONBOARDED,
			"",
			om_status.InitializationDone)

		return nil
	}

	if fdoErr := runFDOActionsIfNeeded(ctx, deviceInfo); fdoErr != nil {
		return fdoErr
	}

	// in progress
	return inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "TO2 status not completed yet for host %s",
		deviceInfo.GUID)
}

func runFDOActionsIfNeeded(ctx context.Context, deviceInfo utils.DeviceInfo) error {
	zlog.Info().Msgf("Running FDO actions for host %s", deviceInfo.GUID)

	clientID, clientSecret, err := createENCredentialsIfNotExists(ctx, deviceInfo)
	if err != nil {
		return err
	}

	clientIDFilename := fmt.Sprintf("%s_%s", deviceInfo.FdoGUID, "client_id")
	err = fdoclient.SendFileToOwner(ctx, clientIDFilename, clientID)
	if err != nil {
		zlog.MiErr(err).Msg("")
		return inv_errors.Errorf("Failed to send client_id file to FDO owner: %v", err)
	}

	clientSecretFilename := fmt.Sprintf("%s_%s", deviceInfo.FdoGUID, "client_secret")
	err = fdoclient.SendFileToOwner(ctx, clientSecretFilename, clientSecret)
	if err != nil {
		zlog.MiErr(err).Msg("")
		return inv_errors.Errorf("Failed to send client_secret file to FDO owner: %v", err)
	}

	payload := fmt.Sprintf(sviInfoPayload, "client_id", "client_secret")
	// doing svi for secret Transfer
	err = fdoclient.ExecuteSVI(ctx, payload)
	if err != nil {
		return inv_errors.Errorf("Failed to initiate secure transfer of FDO files: %v", err)
	}

	return nil
}

func CheckStatusOrRunDIWorkflow(ctx context.Context, deviceInfo utils.DeviceInfo, instance *computev1.InstanceResource) error {
	if !*common.FlagEnableDeviceInitialization {
		zlog.Warn().Msgf("enableDeviceInitialization is set to false, skipping running DI workflow")
		return nil
	}

	zlog.Info().Msgf("Checking status of DI workflow for host %s", deviceInfo.GUID)

	kubeClient, err := tinkerbell.K8sClientFactory()
	if err != nil {
		return err
	}

	util.PopulateHostStatus(instance, computev1.HostStatus_HOST_STATUS_INITIALIZING,
		"", // no details as workflow is not started yet
		om_status.InitializationInProgress)

	diWorkflowName := "workflow-" + deviceInfo.GUID
	status, err := getWorkflow(ctx, kubeClient, diWorkflowName)
	if err != nil && inv_errors.IsNotFound(err) {
		zlog.Debug().Msgf("DI workflow for host %ss does not yet exist.", deviceInfo.GUID)
		// This may happen if:
		// 1) workflow for Instance is not created yet -> proceed to runWorkflow()
		// 2) we already finished & removed workflow for Instance -> in this case we should never get here
		runErr := runDIWorkflow(ctx, kubeClient, deviceInfo)
		if runErr != nil {
			return runErr
		}

		// runDIWorkflow returned no error, but we return an error here so that the upper layer can handle it appropriately
		// and reconcile until the workflow is finished.
		return inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "DI workflow started, waiting for it to complete")
	}

	if err != nil {
		// some unexpected error, we fail to get workflow status
		// This may be some network connection error, so keep host status set to INITIALIZING
		return err
	}

	return handleWorkflowStatus(instance, status,
		computev1.HostStatus_HOST_STATUS_INITIALIZED, computev1.HostStatus_HOST_STATUS_INIT_FAILED,
		om_status.InitializationDone, om_status.InitializationFailed)
}

func getWorkflow(ctx context.Context, k8sCli client.Client, workflowName string) (*tink.Workflow, error) {
	got := &tink.Workflow{}
	clientErr := k8sCli.Get(ctx, types.NamespacedName{Namespace: k8sNamespace, Name: workflowName}, got)
	if clientErr != nil && errors.IsNotFound(clientErr) {
		zlog.MiSec().MiErr(clientErr).Msg("")
		return nil, inv_errors.Errorfc(codes.NotFound, "Cannot get workflow %s status", workflowName)
	}

	if clientErr != nil {
		zlog.MiSec().MiErr(clientErr).Msgf("")
		// some other error that may need retry
		return nil, inv_errors.Errorf("Failed to get workflow %s status.", workflowName)
	}

	zlog.Debug().Msgf("Workflow %s state: %s", got.Name, got.Status.State)

	return got, nil
}

func runDIWorkflow(ctx context.Context, k8sCli client.Client, deviceInfo utils.DeviceInfo) error {
	zlog.Info().Msgf("Creating DI workflow for host %s", deviceInfo.GUID)

	if err := tinkerbell.CreateHardwareIfNotExists(ctx, k8sCli, k8sNamespace, deviceInfo); err != nil {
		return err
	}

	diTemplate, err := tinkerbell.GenerateTemplateForDI(k8sNamespace, deviceInfo)
	if err != nil {
		return err
	}

	if err := tinkerbell.CreateTemplateIfNotExists(ctx, k8sCli, diTemplate); err != nil {
		return err
	}

	diWorkflow := tinkerbell.NewWorkflow(workFlowPrefixName+deviceInfo.GUID,
		k8sNamespace, deviceInfo.HwMacID,
		hwPrefixName+deviceInfo.GUID,
		"fdodi-"+deviceInfo.GUID)

	if err := tinkerbell.CreateWorkflowIfNotExists(ctx, k8sCli, diWorkflow); err != nil {
		return err
	}

	zlog.Debug().Msgf("DI workflow %s for host %s created successfully", diWorkflow.Name, deviceInfo.GUID)

	return nil
}

func uploadFDOVoucherScript(ctx context.Context, deviceInfo utils.DeviceInfo) (string, error) {
	zlog.Info().Msgf("Uploading FDO voucher script for host %s", deviceInfo.GUID)

	fdoGUID, err := fdoclient.DoVoucherExtension(ctx, deviceInfo)
	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("Failed to upload FDO voucher extensions for host %s", deviceInfo.GUID)
		return "", err
	}

	zlog.Debug().Msgf("FDO voucher script for host %s uploaded successfully. "+
		"Generated FDO GUID: %s", deviceInfo.GUID, fdoGUID)

	return fdoGUID, nil
}

// TODO (LPIO-1865).
func createENCredentialsIfNotExists(ctx context.Context, deviceInfo utils.DeviceInfo) (string, string, error) {
	authService, err := auth.AuthServiceFactory(ctx)
	if err != nil {
		return "", "", err
	}
	defer authService.Logout(ctx)

	clientID, clientSecret, err := authService.GetCredentialsByUUID(ctx, deviceInfo.GUID)
	if err != nil && inv_errors.IsNotFound(err) {
		return authService.CreateCredentialsWithUUID(ctx, deviceInfo.GUID)
	}

	if err != nil {
		zlog.MiSec().MiErr(err).Msgf("")
		// some other error that may need retry
		return "", "", inv_errors.Errorf("Failed to check if EN credentials for host %s exist.", deviceInfo.GUID)
	}

	zlog.Debug().Msgf("EN credentials for host %s already exists.", deviceInfo.GUID)

	return clientID, clientSecret, nil
}

func DeleteTinkHardwareForHostIfExist(ctx context.Context, hostUUID string) error {
	return tinkerbell.DeleteHardwareForHostIfExist(ctx, k8sNamespace, hostUUID)
}

func DeleteDIWorkflowResourcesIfExist(ctx context.Context, hostUUID string) error {
	return tinkerbell.DeleteDIWorkflowResourcesIfExist(ctx, k8sNamespace, hostUUID)
}

func DeleteProdWorkflowResourcesIfExist(ctx context.Context, hostUUID string) error {
	return tinkerbell.DeleteProdWorkflowResourcesIfExist(ctx, k8sNamespace, hostUUID)
}

func handleWorkflowStatus(instance *computev1.InstanceResource, workflow *tink.Workflow,
	onSuccessStatus, onFailureStatus computev1.HostStatus,
	onSuccessOnboardingStatus, onFailureOnboardingStatus inv_status.ResourceStatus,
) error {
	intermediateWorkflowState := tinkerbell.GenerateStatusDetailFromWorkflowState(workflow)

	zlog.Debug().Msgf("Workflow %s status for host %s is %s. Workflow state: %q", workflow.Name,
		instance.GetHost().GetUuid(),
		workflow.Status.State,
		intermediateWorkflowState)

	switch workflow.Status.State {
	case tink.WorkflowStateSuccess:
		// success, proceed further
		util.PopulateHostStatus(instance, onSuccessStatus, "",
			onSuccessOnboardingStatus)
		return nil
	case tink.WorkflowStateFailed, tink.WorkflowStateTimeout:
		// indicates unrecoverable error, we should update current_state = ERROR
		util.PopulateHostStatus(instance, onFailureStatus, intermediateWorkflowState,
			onFailureOnboardingStatus)
		return inv_errors.Errorfc(codes.Aborted, "")
	case "", tink.WorkflowStateRunning, tink.WorkflowStatePending:
		// not started yet or in progress
		util.PopulateHostStatusDetail(instance, intermediateWorkflowState)
		return inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "")
	default:
		zlog.MiSec().MiError("Unknown workflow state %s", workflow.Status.State)
		return inv_errors.Errorf("Unknown workflow state %s", workflow.Status.State)
	}
}
