/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onboarding

import (
	"context"
	"fmt"
	"strings"
	"time"

	tink "github.com/tinkerbell/tink/api/v1alpha1"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	computev1 "github.com/open-edge-platform/infra-core/inventory/v2/pkg/api/compute/v1"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/auth"
	inv_errors "github.com/open-edge-platform/infra-core/inventory/v2/pkg/errors"
	"github.com/open-edge-platform/infra-core/inventory/v2/pkg/logging"
	inv_status "github.com/open-edge-platform/infra-core/inventory/v2/pkg/status"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/env"
	onboarding_types "github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/onboarding/types"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/tinkerbell/templates"
	"github.com/open-edge-platform/infra-onboarding/onboarding-manager/internal/util"
	om_status "github.com/open-edge-platform/infra-onboarding/onboarding-manager/pkg/status"
)

var (
	clientName            = "Workflow"
	zlog                  = logging.GetLogger(clientName)
	actionStatusMap       = make(map[string]string)
	actionStartTimes      = make(map[string]time.Time)
	actionRuning          = make(map[string]float64)
	actionSuccessDuration = make(map[string]int64)
)

// generateWorkflowName returns workflow name in format "workflow-<UUID>".
func generateWorkflowName(uuid string) string {
	return fmt.Sprintf("workflow-%s", uuid)
}

func CheckWorkflowExist(ctx context.Context,
	deviceInfo onboarding_types.DeviceInfo,
	instance *computev1.InstanceResource,
) bool {
	zlog.Debug().Msgf("Checking status of workflow for host %s", deviceInfo.GUID)

	kubeClient, err := tinkerbell.K8sClientFactory()
	if err != nil {
		return false
	}

	prodWorkflowName := generateWorkflowName(deviceInfo.GUID)
	_, err = getWorkflow(ctx, kubeClient, prodWorkflowName, instance.Host.ResourceId)
	if err != nil || inv_errors.IsNotFound(err) {
		return false
	}
	return true
}

func CheckStatusOrRunProdWorkflow(ctx context.Context,
	deviceInfo onboarding_types.DeviceInfo,
	instance *computev1.InstanceResource,
) error {
	zlog.Debug().Msgf("Checking status of Prod workflow for host %s", deviceInfo.GUID)

	kubeClient, err := tinkerbell.K8sClientFactory()
	if err != nil {
		return err
	}

	prodWorkflowName := generateWorkflowName(deviceInfo.GUID)
	workflow, err := getWorkflow(ctx, kubeClient, prodWorkflowName, instance.Host.ResourceId)
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
	ctx context.Context, k8sCli client.Client, deviceInfo onboarding_types.DeviceInfo, instance *computev1.InstanceResource,
) error {
	zlog.Debug().Msgf("Creating prod workflow for host %s", deviceInfo.GUID)

	clientID, clientSecret, err := createENCredentialsIfNotExists(ctx, instance.GetTenantId(), deviceInfo)
	if err != nil {
		return err
	}

	deviceInfo.AuthClientID = clientID
	deviceInfo.AuthClientSecret = clientSecret
	deviceInfo.TenantID = instance.GetTenantId()

	if instance.GetLocalaccount() != nil {
		deviceInfo.LocalAccountUserName = instance.GetLocalaccount().Username
		deviceInfo.SSHKey = instance.GetLocalaccount().SshKey
	}

	if len(instance.CustomConfig) != 0 {
		deviceInfo.CustomConfigs = make(map[string]string)
		for _, customCfg := range instance.CustomConfig {
			deviceInfo.CustomConfigs[customCfg.Name] = customCfg.Config
		}
	}

	templateName, found := templates.OSTypeToTemplateName[deviceInfo.OsType]
	if !found {
		return inv_errors.Errorf("Cannot find Tinkerbell template for OS type %s", deviceInfo.OsType)
	}

	workflowHardwareMap, err := tinkerbell.GenerateWorkflowInputs(ctx, deviceInfo)
	if err != nil {
		return err
	}

	prodWorkflow := tinkerbell.NewWorkflow(
		generateWorkflowName(deviceInfo.GUID),
		env.K8sNamespace,
		tinkerbell.DummyHardwareName,
		templateName,
		workflowHardwareMap)

	if createWFErr := tinkerbell.CreateWorkflowIfNotExists(ctx, k8sCli, prodWorkflow); createWFErr != nil {
		return createWFErr
	}

	zlog.Debug().Msgf("Prod workflow %s for host %s created successfully", prodWorkflow.Name, deviceInfo.GUID)

	return nil
}

//nolint:cyclop // May effect the functionality, need to simplify this in future
func getWorkflow(ctx context.Context, k8sCli client.Client, workflowName, hostResourceID string) (*tink.Workflow, error) {
	got := &tink.Workflow{}
	clientErr := k8sCli.Get(ctx, types.NamespacedName{Namespace: env.K8sNamespace, Name: workflowName}, got)
	if clientErr != nil && errors.IsNotFound(clientErr) {
		zlog.InfraSec().Debug().Msgf("%s", clientErr)
		return nil, inv_errors.Errorfc(codes.NotFound, "Workflow %s doesn't exist", workflowName)
	}

	if clientErr != nil {
		zlog.InfraSec().InfraErr(clientErr).Msgf("")
		// some other error that may need retry
		return nil, inv_errors.Errorf("Failed to get workflow %s status.", workflowName)
	}

	// Enable Instrumentation code in Debug mode
	// Time measurements for various provisioning tinker action
	//  if there are tasks and actions to iterate over.
	if len(got.Status.Tasks) > 0 {
		for _, task := range got.Status.Tasks {
			if len(task.Actions) > 0 {
				// Check if the task has actions to iterate over
				for _, action := range task.Actions {
					actionStatusMap[workflowName+action.Name] = string(action.Status)
					switch action.Status {
					case tink.WorkflowStatePending:
						if _, ok := actionStartTimes[workflowName+action.Name]; !ok &&
							action.Name == tinkerbell.ActionSecureBootStatusFlagRead {
							actionStartTimes[workflowName+action.Name] = time.Now()
						}
					case tink.WorkflowStateRunning:
						if _, ok := actionRuning[workflowName+action.Name]; !ok &&
							action.Name == tinkerbell.ActionSecureBootStatusFlagRead {
							actionRuning[workflowName+action.Name] = time.Since(actionStartTimes[workflowName+action.Name]).
								Seconds()
						}
					case tink.WorkflowStateSuccess:
						actionSuccessDuration[workflowName+action.Name] = action.Seconds
					default:
						zlog.Debug().Msgf("workflow %s for action %s status %s", workflowName, action.Name, string(action.Status))
						for actionName := range actionStatusMap {
							delete(actionStartTimes, actionName)
							delete(actionRuning, actionName)
							delete(actionSuccessDuration, actionName)
							delete(actionStatusMap, actionName)
						}
					}
				}
			}
		}
	}

	for actionName, actionStatus := range actionStatusMap {
		if actionName == workflowName+tinkerbell.ActionReboot {
			if actionStatus == string(tink.WorkflowStateSuccess) {
				// Total Time for all tinker actions
				var totalDuration int64
				for actionN, actionSuccessTime := range actionSuccessDuration {
					if actionN == workflowName+tinkerbell.ActionSecureBootStatusFlagRead {
						msg := fmt.Sprintf(
							"Instrumentation Info for workflow %s: action name %s pending to running time %.2f, "+
								"host resource ID: %s",
							workflowName,
							"secure-boot-status-flag-read",
							actionRuning[workflowName+tinkerbell.ActionSecureBootStatusFlagRead],
							hostResourceID,
						)
						zlog.Info().Msg(msg)
						delete(actionStartTimes, actionN)
						delete(actionRuning, actionN)
					}
					if strings.Contains(actionN, workflowName) {
						totalDuration += actionSuccessTime
						zlog.Info().Msgf(
							"Instrumentation Info for workflow %s actionName %s time for running to success %d, "+
								"for host resource ID: %s",
							workflowName,
							strings.Split(actionN, workflowName)[1],
							actionSuccessTime,
							hostResourceID,
						)
						delete(actionSuccessDuration, actionN)
						delete(actionStatusMap, actionN)
					}
				}
				zlog.Info().Msgf(
					"Instrumentation Info for workflow %s, for host resource ID: %s: Total Time for all TinkerActions %d",
					workflowName,
					hostResourceID,
					totalDuration,
				)
			}
		}
	}
	zlog.Debug().Msgf("Workflow %s state: %s", got.Name, got.Status.State)
	return got, nil
}

// TODO (ITEP-1865).
func createENCredentialsIfNotExists(
	ctx context.Context,
	tenantID string,
	deviceInfo onboarding_types.DeviceInfo,
) (clientID, clientSecret string, err error) {
	authService, err := auth.AuthServiceFactory(ctx)
	if err != nil {
		return "", "", err
	}
	defer authService.Logout(ctx)

	clientID, clientSecret, err = authService.GetCredentialsByUUID(ctx, tenantID, deviceInfo.GUID)
	if err != nil && inv_errors.IsNotFound(err) {
		return authService.CreateCredentialsWithUUID(ctx, tenantID, deviceInfo.GUID)
	}

	if err != nil {
		zlog.InfraSec().InfraErr(err).Msgf("")
		// some other error that may need retry
		return "", "", inv_errors.Errorf("Failed to check if EN credentials for host %s exist.", deviceInfo.GUID)
	}

	zlog.Debug().Msgf("EN credentials for host %s already exists.", deviceInfo.GUID)

	return clientID, clientSecret, nil
}

func DeleteTinkerbellWorkflowIfExists(ctx context.Context, hostUUID string) error {
	return tinkerbell.DeleteWorkflowIfExists(ctx, env.K8sNamespace, generateWorkflowName(hostUUID))
}

func handleWorkflowStatus(instance *computev1.InstanceResource, workflow *tink.Workflow,
	onSuccessProvisioningStatus, onFailureProvisioningStatus inv_status.ResourceStatus,
) error {
	intermediateWorkflowState := tinkerbell.GenerateStatusDetailFromWorkflowState(workflow)

	zlog.Debug().Msgf("Workflow %s status for host %s is %s. Workflow state: %q", workflow.Name, instance.GetHost().GetUuid(),
		workflow.Status.State, intermediateWorkflowState)

	switch workflow.Status.State {
	case tink.WorkflowStateSuccess:
		// success, proceed further
		util.PopulateInstanceStatusAndCurrentState(
			instance, computev1.InstanceState_INSTANCE_STATE_RUNNING,
			om_status.NewStatusWithDetails(onSuccessProvisioningStatus, intermediateWorkflowState))

		// FIXME: We retrieve original desired OS from workflow's template parameters (hardware map).
		//  This is temporary, until we implement Day2 refactoring.
		osResourceID, exists := workflow.Spec.HardwareMap["DeviceInfoOSResourceID"]
		if !exists {
			return inv_errors.Errorf("OS resource ID not found in Tinkerbell workflow %s", workflow.Name)
		}

		util.PopulateCurrentOS(instance, osResourceID)
		// don't set Rebooting for Standalone ENs as we don't have agents that will converge to Running eventually
		isStandalone, err := util.IsStandalone(instance)
		if err != nil {
			return err
		}
		if !isStandalone {
			// set host status to "rebooting" since every successful workflow ends with a reboot
			util.PopulateHostStatus(instance, om_status.HostStatusRebooting)
		}
		return nil
	case tink.WorkflowStateFailed, tink.WorkflowStateTimeout:
		ProvisioningStatusFailed := om_status.NewStatusWithDetails(onFailureProvisioningStatus,
			intermediateWorkflowState)
		// report error provisioning status
		util.PopulateInstanceProvisioningStatus(instance, ProvisioningStatusFailed)
		return inv_errors.Errorfc(codes.Aborted, "Workflow failed or timed out")
	case "", tink.WorkflowStateRunning, tink.WorkflowStatePending:
		ProvisioningStatusInProgress := om_status.NewStatusWithDetails(om_status.ProvisioningStatusInProgress,
			intermediateWorkflowState)
		util.PopulateInstanceStatusAndCurrentState(
			instance, computev1.InstanceState_INSTANCE_STATE_UNSPECIFIED, ProvisioningStatusInProgress)

		return inv_errors.Errorfr(inv_errors.Reason_OPERATION_IN_PROGRESS, "")
	default:
		zlog.InfraSec().InfraError("Unknown workflow state %s", workflow.Status.State)
		return inv_errors.Errorf("Unknown workflow state %s", workflow.Status.State)
	}
}
