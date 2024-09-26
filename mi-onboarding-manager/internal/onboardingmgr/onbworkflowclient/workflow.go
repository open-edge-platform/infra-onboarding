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
	"os"
	"time"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/common"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/env"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/fdoclient"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/onboardingmgr/utils"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/tinkerbell"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/util"
	om_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/pkg/status"
	computev1 "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/api/compute/v1"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/auth"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/logging"
	inv_status "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/v2/pkg/status"
)

const (
	//nolint:lll // keep long line for better readability
	sviInfoPayload = `[{"filedesc" : "client_id","resource" : "$(guid)_%s"},{"filedesc" : "client_secret","resource" : "$(guid)_%s"}]`
)

var (
	clientName        = "Workflow"
	zlog              = logging.GetLogger(clientName)
	actionStatusMap   = make(map[string]string)
	actionStartTimes  = make(map[string]time.Time)
	actionRunTimes    = make(map[string]time.Time)
	actionDurations   = make(map[string]time.Duration)
	workflowStartTime time.Time
	rebootEndTime     time.Time
)

//nolint:tagliatelle // Renaming the json keys may effect while unmarshalling/marshaling so, used nolint.
type ResponseData struct {
	To2CompletedOn string `json:"to2CompletedOn"`
	To0Expiry      string `json:"to0Expiry"`
}

func checkTO2StatusCompleted(_ context.Context, deviceInfo utils.DeviceInfo) (bool, error) {
	// Make an HTTP GET request
	to2URL := fmt.Sprintf("http://%s:%s/api/v1/owner/state/%s",
		env.FdoOwnerDNS, env.FdoOwnerPort, deviceInfo.FdoGUID)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, to2URL, http.NoBody)
	if err != nil {
		respErr := inv_errors.Errorf("Error making HTTP GET request %v", err)
		zlog.MiSec().MiErr(err).Msgf("")
		return false, respErr
	}
	httpClient := &http.Client{}
	resp, err := httpClient.Do(request)
	if err != nil {
		respErr := inv_errors.Errorf("Error making HTTP GET request %v", err)
		zlog.MiSec().MiErr(err).Msgf("")
		return false, respErr
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		err = inv_errors.Errorf("Failed to perform API call to %s with status code %v",
			to2URL, resp.StatusCode)
		zlog.MiErr(err).Msg("")
		return false, err
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
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
			env.FdoOwnerDNS, deviceInfo.GUID)
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

	prodWorkflowName := tinkerbell.GetProdWorkflowName(deviceInfo.GUID)
	workflow, err := getWorkflow(ctx, kubeClient, prodWorkflowName)
	if err != nil && inv_errors.IsNotFound(err) {
		// This may happen if:
		// 1) workflow for Instance is not created yet -> proceed to runProdWorkflow()
		// 2) we already finished & removed workflow for Instance -> in this case we should never get here
		runErr := runProdWorkflow(ctx, kubeClient, deviceInfo, instance)
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

	util.PopulateInstanceStatusAndCurrentState(
		instance, computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
		om_status.ProvisioningStatusInProgress)

	return handleWorkflowStatus(instance, workflow,
		om_status.ProvisioningStatusDone, om_status.ProvisioningStatusFailed)
}

func runProdWorkflow(
	ctx context.Context, k8sCli client.Client, deviceInfo utils.DeviceInfo, instance *computev1.InstanceResource,
) error {
	zlog.Info().Msgf("Creating prod workflow for host %s", deviceInfo.GUID)

	if *common.FlagEnableDeviceInitialization {
		zlog.Debug().Msgf("Checking TO2 status completed for host %s", deviceInfo.GUID)

		util.PopulateHostOnboardingStatus(instance,
			om_status.OnboardingStatusInProgress)

		// we should wait until TO2 process is completed before running prod workflow
		isCompleted, err := checkTO2StatusCompleted(ctx, deviceInfo)
		if err != nil {
			return err
		}

		if !isCompleted {
			zlog.Debug().Msgf("TO2 process still not completed for host %s", deviceInfo.GUID)
			return inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "TO2 process started, waiting for it to complete")
		}

		util.PopulateHostOnboardingStatus(instance,
			om_status.OnboardingStatusDone)
	}

	if !*common.FlagEnableDeviceInitialization {
		// normally EN credentials should be created as part of the device initialization phase,
		// but we have to do it here if the DI phase is disabled.
		clientID, clientSecret, err := createENCredentialsIfNotExists(ctx, deviceInfo)
		if err != nil {
			return err
		}
		// TODO: to be removed once DI is always enabled.
		deviceInfo.AuthClientID = clientID
		deviceInfo.AuthClientSecret = clientSecret
	}

	// NOTE: IMO (Tomasz) this is a one-time operation that should be done when a host is discovered and created.
	// So it shouldn't be here (move to host reconciler?)
	if createHwErr := tinkerbell.CreateHardwareIfNotExists(ctx, k8sCli, env.K8sNamespace, deviceInfo,
		instance.GetDesiredOs().ResourceId); createHwErr != nil {
		return createHwErr
	}

	prodTemplate, err := tinkerbell.GenerateTemplateForProd(env.K8sNamespace, deviceInfo)
	if err != nil {
		zlog.MiErr(err).Msg("")
		return inv_errors.Errorf("Failed to generate Tinkerbell prod template for host %s", deviceInfo.GUID)
	}

	if createTemplErr := tinkerbell.CreateTemplateIfNotExists(ctx, k8sCli, prodTemplate); createTemplErr != nil {
		return createTemplErr
	}

	prodWorkflow := tinkerbell.NewWorkflow(
		tinkerbell.GetProdWorkflowName(deviceInfo.GUID),
		env.K8sNamespace,
		deviceInfo.HwMacID,
		tinkerbell.GetTinkHardwareName(deviceInfo.GUID),
		tinkerbell.GetProdTemplateName(deviceInfo.ImgType, deviceInfo.GUID))

	if createWFErr := tinkerbell.CreateWorkflowIfNotExists(ctx, k8sCli, prodWorkflow); createWFErr != nil {
		return createWFErr
	}

	zlog.Debug().Msgf("Prod workflow %s for host %s created successfully", prodWorkflow.Name, deviceInfo.GUID)

	return nil
}

// RunFDOActions runs all required FDO actions such as voucher extension and SVI calls.
func RunFDOActions(ctx context.Context, deviceInfo *utils.DeviceInfo) error {
	if !*common.FlagEnableDeviceInitialization {
		zlog.Warn().Msgf("enableDeviceInitialization is set to false, skipping FDO actions")
		return nil
	}

	zlog.Debug().Msgf("Running FDO actions for host %s", deviceInfo.GUID)

	// we need to upload voucher extension before checking TO2 status to get FDO GUID that is needed for TO2 status check.
	fdoGUID, err := uploadFDOVoucherScript(ctx, *deviceInfo)
	if err != nil {
		return err
	}
	deviceInfo.FdoGUID = fdoGUID

	clientID, clientSecret, err := createENCredentialsIfNotExists(ctx, *deviceInfo)
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

	zlog.Debug().Msgf("FDO actions successfully executed for host %s", deviceInfo.GUID)

	return nil
}

func CheckStatusOrRunDIWorkflow(ctx context.Context, deviceInfo utils.DeviceInfo, instance *computev1.InstanceResource) error {
	if !*common.FlagEnableDeviceInitialization {
		zlog.Warn().Msgf("enableDeviceInitialization is set to false, skipping running DI workflow")
		return nil
	}

	zlog.Debug().Msgf("Checking status of DI workflow for host %s", deviceInfo.GUID)

	kubeClient, err := tinkerbell.K8sClientFactory()
	if err != nil {
		return err
	}

	util.PopulateHostOnboardingStatus(instance,
		om_status.InitializationInProgress)

	diWorkflowName := tinkerbell.GetDIWorkflowName(deviceInfo.GUID)
	status, err := getWorkflow(ctx, kubeClient, diWorkflowName)
	if err != nil && inv_errors.IsNotFound(err) {
		zlog.Debug().Msgf("DI workflow for host %s does not yet exist.", deviceInfo.GUID)
		// This may happen if:
		// 1) workflow for Instance is not created yet -> proceed to runWorkflow()
		// 2) we already finished & removed workflow for Instance -> in this case we should never get here
		runErr := runDIWorkflow(ctx, kubeClient, deviceInfo, instance)
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
		om_status.InitializationDone, om_status.InitializationFailed)
}

func CheckStatusOrRunRebootWorkflow(
	ctx context.Context, deviceInfo utils.DeviceInfo, instance *computev1.InstanceResource,
) error {
	if !*common.FlagEnableDeviceInitialization {
		zlog.Warn().Msgf("enableDeviceInitialization is set to false, skipping running reboot workflow")
		return nil
	}

	zlog.Debug().Msgf("Checking status of Reboot workflow for host %s", deviceInfo.GUID)

	kubeClient, err := tinkerbell.K8sClientFactory()
	if err != nil {
		return err
	}

	rebootWorkflowName := tinkerbell.GetRebootWorkflowName(deviceInfo.GUID)
	status, err := getWorkflow(ctx, kubeClient, rebootWorkflowName)
	if err != nil && inv_errors.IsNotFound(err) {
		zlog.Debug().Msgf("Reboot workflow for host %s does not yet exist.", deviceInfo.GUID)
		// This may happen if:
		// 1) workflow for Instance is not created yet -> proceed to runWorkflow()
		// 2) we already finished & removed workflow for Instance -> in this case we should never get here
		runErr := runRebootWorkflow(ctx, kubeClient, deviceInfo)
		if runErr != nil {
			return runErr
		}

		// runRebootWorkflow returned no error, but we return an error here so that the upper layer can handle it appropriately
		// and reconcile until the workflow is finished.
		return inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "Reboot workflow started, waiting for it to complete")
	}

	if err != nil {
		// some unexpected error, we fail to get workflow status
		return err
	}

	// keep statuses from the DI workflow on success
	return handleWorkflowStatus(instance, status,
		inv_status.New(instance.GetProvisioningStatus(), instance.GetProvisioningStatusIndicator()),
		om_status.InitializationFailed)
}

func formatDuration(d time.Duration) string {
	hours := d / time.Hour
	remainingDuration := d % time.Hour // Remainder after subtracting hours
	minutes := remainingDuration / time.Minute
	remainingDuration %= time.Minute // Remainder after subtracting minutes
	seconds := remainingDuration / time.Second
	return fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
}

//nolint:cyclop // May effect the functionality, need to simplify this in future
func getWorkflow(ctx context.Context, k8sCli client.Client, workflowName string) (*tink.Workflow, error) {
	got := &tink.Workflow{}
	clientErr := k8sCli.Get(ctx, types.NamespacedName{Namespace: env.K8sNamespace, Name: workflowName}, got)
	if clientErr != nil && errors.IsNotFound(clientErr) {
		zlog.MiSec().Debug().Msgf("%s", clientErr)
		return nil, inv_errors.Errorfc(codes.NotFound, "Workflow %s doesn't exist", workflowName)
	}

	if clientErr != nil {
		zlog.MiSec().MiErr(clientErr).Msgf("")
		// some other error that may need retry
		return nil, inv_errors.Errorf("Failed to get workflow %s status.", workflowName)
	}

	// if tinker time measurement flag is enabled
	if os.Getenv("ENABLE_ACTION_TIMESTAMPS") == "true" {
		logFilePath := os.Getenv("TIMESTAMP_LOG_PATH")
		if logFilePath == "" {
			zlog.Warn().Msg("TIMESTAMP_LOG_PATH env is not set")
		}
		utils.Init(logFilePath)
		// check if the status is not empty and
		//  if there are tasks and actions to iterate over.
		if len(got.Status.Tasks) > 0 {
			for _, task := range got.Status.Tasks {
				if len(task.Actions) > 0 {
					// Check if the task has actions to iterate over
					for _, action := range task.Actions {
						lastStatus, existsFlag := actionStatusMap[action.Name]
						// store first pending status for each action
						if !existsFlag || lastStatus != string(action.Status) {
							actionStatusMap[action.Name] = string(action.Status)
							//nolint:exhaustive //TODO WorkflowStateFailed and WorkflowStateTimeout will be handled in future
							switch action.Status {
							case tink.WorkflowStatePending:
								if _, exists := actionStartTimes[action.Name]; !exists {
									// Calculate the duration for action
									actionStartTimes[action.Name] = time.Now()
								}
							case tink.WorkflowStateRunning:
								if startTime, hasStartTime := actionStartTimes[action.Name]; hasStartTime {
									// Calculate the duration for running action
									duration := time.Since(startTime)
									formattedDuration := formatDuration(duration)
									utils.TimeStamp(
										fmt.Sprintf("action name <%s>,time duration <%s> pending to running",
											action.Name, formattedDuration))
									actionRunTimes[action.Name] = time.Now()
									if workflowStartTime.IsZero() {
										// first tinker action execution set for one time.
										// when moves from "pending" to "running"
										workflowStartTime = time.Now()
									}
								}
							case tink.WorkflowStateSuccess:
								if startTime, hasStartTime := actionRunTimes[action.Name]; hasStartTime {
									// Calculate the duration for action.
									duration := time.Since(startTime)
									// Record the duration for this action.
									actionDurations[action.Name] = duration
									// Format the duration for logging
									formattedActionDuration := formatDuration(duration)
									// Log the individual action duration.
									utils.TimeStamp(
										fmt.Sprintf("action name <%s>,time duration <%s> running to success",
											action.Name, formattedActionDuration))
									// Remove the start time from the map as it's no longer needed.
									delete(actionStartTimes, action.Name)
								}
								// reboot endTime is set when the "reboot" action reaches "success".
								if action.Name == tinkerbell.ActionReboot {
									rebootEndTime = time.Now()
									utils.TimeStamp(fmt.Sprintf("Last action name <%s>, end time <%s>",
										action.Name, rebootEndTime))
								}
							}
						}
					}
				} else {
					utils.TimeStamp("No action found in the workflow.")
				}
			}
			for actionName, actionStatus := range actionStatusMap {
				if actionName == tinkerbell.ActionReboot {
					if actionStatus == string(tink.WorkflowStateSuccess) {
						totalTinkerExecutionTime := rebootEndTime.Sub(workflowStartTime)
						formattedTotalDuration := formatDuration(totalTinkerExecutionTime)
						utils.TimeStamp(fmt.Sprintf("time duration for all tinker action execution : <%s>",
							formattedTotalDuration))
					}
				}
			}
		}
	}
	zlog.Debug().Msgf("Workflow %s state: %s", got.Name, got.Status.State)
	return got, nil
}

func runDIWorkflow(ctx context.Context, k8sCli client.Client, deviceInfo utils.DeviceInfo,
	instance *computev1.InstanceResource,
) error {
	zlog.Info().Msgf("Creating DI workflow for host %s", deviceInfo.GUID)

	if err := tinkerbell.CreateHardwareIfNotExists(ctx, k8sCli, env.K8sNamespace, deviceInfo,
		instance.GetDesiredOs().ResourceId); err != nil {
		return err
	}

	diTemplate, err := tinkerbell.GenerateTemplateForDI(env.K8sNamespace, deviceInfo)
	if err != nil {
		return err
	}

	if err := tinkerbell.CreateTemplateIfNotExists(ctx, k8sCli, diTemplate); err != nil {
		return err
	}

	diWorkflow := tinkerbell.NewWorkflow(
		tinkerbell.GetDIWorkflowName(deviceInfo.GUID),
		env.K8sNamespace, deviceInfo.HwMacID,
		tinkerbell.GetTinkHardwareName(deviceInfo.GUID),
		tinkerbell.GetDITemplateName(deviceInfo.GUID))

	if err := tinkerbell.CreateWorkflowIfNotExists(ctx, k8sCli, diWorkflow); err != nil {
		return err
	}

	zlog.Debug().Msgf("DI workflow %s for host %s created successfully", diWorkflow.Name, deviceInfo.GUID)

	return nil
}

func runRebootWorkflow(ctx context.Context, k8sCli client.Client, deviceInfo utils.DeviceInfo) error {
	zlog.Info().Msgf("Creating Reboot workflow for host %s", deviceInfo.GUID)

	rebootTemplate, err := tinkerbell.GenerateTemplateForNodeReboot(env.K8sNamespace, deviceInfo)
	if err != nil {
		return err
	}

	if err := tinkerbell.CreateTemplateIfNotExists(ctx, k8sCli, rebootTemplate); err != nil {
		return err
	}

	rebootWorkflow := tinkerbell.NewWorkflow(
		tinkerbell.GetRebootWorkflowName(deviceInfo.GUID),
		env.K8sNamespace, deviceInfo.HwMacID,
		tinkerbell.GetTinkHardwareName(deviceInfo.GUID),
		tinkerbell.GetRebootTemplateName(deviceInfo.GUID))

	if err := tinkerbell.CreateWorkflowIfNotExists(ctx, k8sCli, rebootWorkflow); err != nil {
		return err
	}

	zlog.Debug().Msgf("Reboot workflow %s for host %s created successfully", rebootWorkflow.Name, deviceInfo.GUID)

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
	return tinkerbell.DeleteHardwareForHostIfExist(ctx, env.K8sNamespace, hostUUID)
}

func DeleteDIWorkflowResourcesIfExist(ctx context.Context, hostUUID string) error {
	return tinkerbell.DeleteDIWorkflowResourcesIfExist(ctx, env.K8sNamespace, hostUUID)
}

func DeleteRebootWorkflowResourcesIfExist(ctx context.Context, hostUUID string) error {
	return tinkerbell.DeleteRebootWorkflowResourcesIfExist(ctx, env.K8sNamespace, hostUUID)
}

func DeleteProdWorkflowResourcesIfExist(ctx context.Context, hostUUID, imgType string) error {
	return tinkerbell.DeleteProdWorkflowResourcesIfExist(ctx, env.K8sNamespace, hostUUID, imgType)
}

func handleWorkflowStatus(instance *computev1.InstanceResource, workflow *tink.Workflow,
	onSuccessProvisioningStatus, onFailureProvisioningStatus inv_status.ResourceStatus,
) error {
	intermediateWorkflowState := tinkerbell.GenerateStatusDetailFromWorkflowState(workflow)

	zlog.Debug().Msgf("Workflow %s status for host %s is %s. Workflow state: %q", workflow.Name,
		instance.GetHost().GetUuid(),
		workflow.Status.State,
		intermediateWorkflowState)

	k8sCli, err := tinkerbell.K8sClientFactory()
	if err != nil {
		return err
	}

	switch workflow.Status.State {
	case tink.WorkflowStateSuccess:
		// success, proceed further
		util.PopulateInstanceStatusAndCurrentState(
			instance, computev1.InstanceState_INSTANCE_STATE_RUNNING,
			onSuccessProvisioningStatus)

		// Retrieve the Tinkerbell hardware resource to get the OS resource ID
		hardwareName := tinkerbell.GetTinkHardwareName(instance.GetHost().GetUuid())
		hardware := &tink.Hardware{}
		err := k8sCli.Get(context.Background(), client.ObjectKey{Name: hardwareName, Namespace: env.K8sNamespace}, hardware)
		if err != nil {
			return inv_errors.Errorf("Failed to retrieve Tinkerbell hardware %s: %v", hardwareName, err)
		}

		if hardware.Spec.Metadata.Instance.OperatingSystem != nil {
			osResourceID := hardware.Spec.Metadata.Instance.OperatingSystem.OsSlug // Use OsSlug as a unique identifier

			util.PopulateCurrentOS(instance, osResourceID)
		} else {
			return inv_errors.Errorf("OS resource ID not found in Tinkerbell hardware %s", hardwareName)
		}
		return nil
	case tink.WorkflowStateFailed, tink.WorkflowStateTimeout:
		// indicates unrecoverable error, we should update current_state = ERROR
		util.PopulateInstanceStatusAndCurrentState(instance, computev1.InstanceState_INSTANCE_STATE_ERROR,
			onFailureProvisioningStatus)
		return inv_errors.Errorfc(codes.Aborted, "")
	case "", tink.WorkflowStateRunning, tink.WorkflowStatePending:
		// not started yet or in progress
		/* TODO: extend the modern status to add detailed intermediateWorkflowState in below ticket
		https://jira.devtools.intel.com/browse/NEX-11962 */
		util.PopulateInstanceStatusAndCurrentState(
			instance, computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED,
			om_status.ProvisioningStatusInProgress)
		return inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "")
	default:
		zlog.MiSec().MiError("Unknown workflow state %s", workflow.Status.State)
		return inv_errors.Errorf("Unknown workflow state %s", workflow.Status.State)
	}
}
