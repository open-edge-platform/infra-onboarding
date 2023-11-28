#!/bin/bash

########################################################################################################
#
#  This file is for automating the complete onboarding installtion for SUT's for IaaS  Project WS3.
#  This covers Setup Host with
#     Proxy Settings, Kubernetes, RKE2 Cluster, tinkerbell stack ,FIDO,OS provisiong  ..
#  Author: Jayanthi, Shankar Srinivas  <shankar.srinivas.jayanthi@intel.com>
#
########################################################################################################
#set -x

SETUP_STATUS_FILENAME=".onboarding_installation_status"
SETUP_LOG_FILENAME="onboarding_logs.txt"
SCRIPT_DIR=$(pwd)

touch $SCRIPT_DIR/$SETUP_STATUS_FILENAME
touch $SCRIPT_DIR/$SETUP_LOG_FILENAME

####Array for the device serial numbers
declare -a DevSerial=()

###global counter variable
g_pt_count=0
g_di_count=0
g_to_count=0
g_vochaer_count=0
tink_stack_name=""
boot_pod=""
device_count=0
total_devices_to_provsion=0
host_ip=""
pub_interface_name=""
load_balancer_ip=""
fdo_client="CLIENT-SDK-TPM"

######Functions

#check all pre conditions before start device onboarding  serivices
function check_pre_condition() {
	#check first rke2 and tinkerbell stack running on the PD system,if not exit the script
	rke2_status=$(sudo systemctl status rke2-server.service | grep -i "Active:" | awk '{print $3}')
	if [ "$rke2_status" != "(running)" ]; then
		echo "rke2 cluster is not running on the PD system,check it before procced!!"
		exit 0
	fi
	tink_stack=$(kubectl get all --all-namespaces | grep -i "service/boots" | awk '{print $1}')
	if [ -z $tink_stack ]; then
		echo "looks tinkerbell stack not running, please check.if required run setup_tinkerbell_stack_with_intel_network.sh"
		exit 0
	elif [[ $(kubectl get all --all-namespaces | grep -i "service/boots" | awk '{print $4}') = "" ]]; then
		echo "looks tinkerbell stack not running properly,please check.if required run setup_tinkerbell_stack_with_intel_network.sh"
		exit 0
	elif [[ $(kubectl get all --all-namespaces | grep -i "service/boots" | awk '{print $5}') = "<pending>" ]]; then
		echo "looks tinkerbell stack not running properly,please check.if required run setup_tinkerbell_stack_with_intel_network.sh"
		exit 0
	elif [[ $(kubectl get all --all-namespaces | grep -i "service/tink-stack" | awk '{print $4}') = "" ]]; then
		echo "looks tinkerbell stack not running properly,please check.if required run setup_tinkerbell_stack_with_intel_network.sh"
		exit 0
	elif [[ $(kubectl get all --all-namespaces | grep -i "service/tink-stack" | awk '{print $5}') = "<pending>" ]]; then
		echo "looks tinkerbell stack not running properly,please check.if required run setup_tinkerbell_stack_with_intel_network.sh"
		exit 0
	fi

	#check for sut onboarding list file data,if it empty or not present exit
	if [ ! -s $SCRIPT_DIR/sut_onboarding_list.txt ] || [ ! -f $SCRIPT_DIR/sut_onboarding_list.txt ]; then
		echo "sut_onboarding_list.txt empty or not present,please check. Its required for onboarding provision"
		exit 0
	fi
}
#This is to check if FDO services already installed on the system,if yes don not install else install
function check_for_fdo_server_setup() {
	#check for the DB status
	local fdo_db_status=$(docker inspect --format "{{json .State.Status}}" db-fdo-db-1)
	local fdo_rv_status=$(docker inspect --format "{{json .State.Status}}" pri-fdo-rv)
	local fdo_mfg_status=$(docker inspect --format "{{json .State.Status}}" pri-fdo-mfg)
	local fdo_owner_status=$(docker inspect --format "{{json .State.Status}}" pri-fdo-owner)

	if [ "$fdo_db_status" != "\"running"\" ]; then
		echo false
	elif [ "$fdo_rv_status" != "\"running"\" ]; then
		echo false
	elif [ "$fdo_mfg_status" != "\"running"\" ]; then
		echo false
	elif [ "$fdo_owner_status" != "\"running"\" ]; then
		echo false
	else
		echo true
	fi
}

#Install the FDO server on provisiong Server
function setup_FDO_services_on_pd() {
	#Install FDO services on provosining server
	if grep -q "FDO Services installtion done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping FDO Services installtion as its already Done"
	else
		local status=$(check_for_fdo_server_setup)
		if [ $status = false ]; then

			#give exec permissions
			chmod +x $SCRIPT_DIR/setup_scripts/setup_FDO_services.sh
			chmod +x $SCRIPT_DIR/setup_scripts/cleanup_delete_FDO_services.sh
			#clean up the FDO services if exists
			pushd $SCRIPT_DIR/setup_scripts
			./cleanup_delete_FDO_services.sh
			./setup_FDO_services.sh $host_ip
			if [ $? -eq 0 ]; then
				echo "FDO Services installtion Done on Provisioning Server"
				echo "FDO Services installtion done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
				popd
			else
				echo "FDO Services installtion Failed on Provisioning Server"
				echo "FDO Services installtion Failed" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
				popd
				exit 0
			fi
		else
			echo "FDO Services installtion done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
		fi
	fi
}

check_for_action_registry() {
	local docker_images=$(curl -X GET https://localhost:5015/v2/_catalog --insecure 2>&1)
	# ["create_partition","efibootset","fdoclient_action","store_alpine"]}
	local strings_to_check=("create_partition" "efibootset" "fdoclient_action" "store_alpine")
	local found=false

	for str in $strings_to_check; do
		if [ ! "$(
			echo "$docker_images" | grep -F -q "$str"
			echo $?
		)" -eq 0 ]; then
			echo "String '$str' is missing in the docker_images."
			found=true
		fi
	done

	if [ "$found" = true ]; then
		echo "false"
	else
		echo "true"
	fi
}

#Install the FDO server on provisiong Server
function setup_FDO_actions_on_pd() {
	#Install FDO services on provosining server
	if grep -q "FDO Actions installtion done" $SCRIPT_DIR/$SETUP_STATUS_FILENAME; then
		echo "Skipping FDO Action build & installtion as its already Done"
	else
		local status=$(check_for_action_registry)
		if [ $status = false ]; then

			#give exec permissions
			chmod +x $SCRIPT_DIR/setup_scripts/setup_actions.sh

			pushd $SCRIPT_DIR/setup_scripts
			./setup_actions.sh $host_ip $load_balancer_ip
			if [ $? -eq 0 ]; then
				echo "FDO Actions local registry setup & installtion done on Provisioning Server"
				echo "FDO Actions installtion done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
				popd
			else
				echo "FDO Actions local registry setup & installtion Failed on Provisioning Server"
				echo "FDO Actions installtion Failed" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
				popd
				exit 0
			fi
		else
			echo "FDO Actions installtion done" >>$SCRIPT_DIR/$SETUP_STATUS_FILENAME
		fi
	fi
}

#Install device initialization (di),T01,T02 and production on the suts
function start_device_onboarding() {
	#Get the cluster namespce
	tink_stack_name=$(kubectl get all --all-namespaces | grep -i "service/boots" | awk '{print $1}')

	#Get the boot-pod name
	boot_pod=$(kubectl get all -n $tink_stack_name | grep pod/boots- | awk '{ print $1 }')

	#apply Device initialization,T01/T02 Flows and production for the sut list

	apply_fdo_Di_flows_to_sut sut_onboarding_list.txt

	#wait for all the devices execution for onboarding
	wait_for_all_devices_onboarding &
	wait
}
#extend the voucher for successful di SUTS
function extend_voucher_for_successful_di_suts() {
	Device_serial=$1

	cd $HOME/pri-fidoiot/component-samples/demo/scripts
	unset http_proxy
	unset https_proxy
	unset HTTP_PROXY
	unset HTTPS_PROXY
	chmod +x extend_upload.sh

	./extend_upload.sh -m sh -c ./secrets/ -e mtls -m $host_ip -o $host_ip -s $Device_serial >$SCRIPT_DIR/.debug_extend_upload.txt 2>/dev/null
	cd -

	if [[ $(cat $SCRIPT_DIR/.debug_extend_upload.txt | grep -F -c "Success in triggering TO0 for $dev_serial") -ge 1 ]]; then

		mac=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $Device_serial | awk '{print $1}')
		echo "Voucher extesion success for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
		#Start the svi script for successful extension voucher
		start_fdo_svi_script $Device_serial &

	elif [[ $(cat $SCRIPT_DIR/.debug_extend_upload.txt | grep -c "Failure in getting extended voucher for device with serial number $Device_serial") -ge 1 ]]; then

		mac=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $Device_serial | awk '{print $1}')
		echo "Voucher extesion failed for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
		update_status_for_onboarding $mac started failed
		update_status_for_onboarding $mac waiting_for_TO_success stopped
		device_count=$(cat $SCRIPT_DIR/.debug_device_count | awk '{print $1}') >/dev/null 2>&1
		device_count=$((device_count + 1))
		echo $device_count >$SCRIPT_DIR/.debug_device_count
		delete_work_flow_sut $mac
	fi
}

#run svi script for the for upload the files in maria DB
function start_fdo_svi_script {
	device_serial_voucher+=$1

	while [ $g_vochaer_count -lt ${#device_serial_voucher[@]} ]; do
		for dev_serial in "${device_serial_voucher[@]}"; do
			#call svi script for upload files in maria DB
			chmod +x $SCRIPT_DIR/fdo-scripts/svi_script.sh
			chmod +x $SCRIPT_DIR/fdo-scripts/test.sh
			unset http_proxy
			unset https_proxy
			unset HTTP_PROXY
			unset HTTPS_PROXY

			#eliminate duplicate serial number for executing same script one more time!
			if echo "${success_svi_list[@]}" | grep -F -qw $dev_serial || echo "${failed_svi_list[@]}" | grep -F -qw $dev_serial; then
				continue
			fi
			cd $SCRIPT_DIR/fdo-scripts
			bash svi_script.sh -c /home/$USER/pri-fidoiot/component-samples/demo/owner/secrets -o $host_ip -p 8043 -s test.sh >$SCRIPT_DIR/.debug_svi_status 2>/dev/null
			cd -

			if [[ $(cat $SCRIPT_DIR/.debug_svi_status | grep -F -c "Owner svi API is success 200") -ge 1 ]]; then

				mac=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $dev_serial | awk '{print $1}')
				sutip=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $dev_serial | awk '{print $5}')
				disktype=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $dev_serial | awk '{print $7}')
				imatype=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $dev_serial | awk '{print $9}')

				echo "svi script success for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
				sleep 20
				success_svi_list+=($dev_serial)
				g_vochaer_count=$((g_vochaer_count + 1))

				#execute the TO2 flow for successful svi run
				apply_fdo_TO_flows_to_sut $mac $sutip $disktype $imatype $dev_serial &
				break
			else

				g_vochaer_count=$((g_vochaer_count + 1))
				mac=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $dev_serial | awk '{print $1}')
				echo "svi script failed for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
				update_status_for_onboarding $mac started failed
				update_status_for_onboarding $mac waiting_for_TO_success stopped
				failed_svi_list+=($dev_serial)
				device_count=$(cat $SCRIPT_DIR/.debug_device_count | awk '{print $1}') >/dev/null 2>&1
				device_count=$((device_count + 1))
				echo $device_count >$SCRIPT_DIR/.debug_device_count
				delete_work_flow_sut $mac
				break
			fi
		done
		sleep 5
	done
}
#Check for the FDO DI Device partion type status
function verify_fdo_di_disk_partion_status() {
	dev_mac+=$1

	while [ $g_pt_count -lt ${#dev_mac[@]} ]; do
		for mac in "${dev_mac[@]}"; do
			#convert mac with out ":" as  boot logs has only plain string for mac
			mac_raw=$(echo "$mac" | sed 's/://g')

			if [[ $(kubectl logs -n $tink_stack_name $boot_pod --since=1m | grep -F workflow-$mac_raw | grep -F "actionName" | grep -F "store-Alpine" | grep -F "os-installation-di" | grep -F -c "STATE_RUNNING") -ge 1 ]] && ! echo "${mac_di_partion_start[@]}" | grep -F -qw $mac_raw; then

				update_status_for_onboarding $mac initiated started
				mac_di_partion_start+=($mac_raw)
				echo "started the Partition creation for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME

			elif [[ $(kubectl logs -n $tink_stack_name $boot_pod --since=1m | grep -F workflow-$mac_raw | grep -F "actionName" | grep -F "store-Alpine" | grep -F "os-installation-di" | grep -F "status" | grep -F -c "STATE_FAILED") -ge 1 ]] && ! echo "${mac_di_partion_failed[@]}" | grep -F -qw $mac_raw; then

				update_status_for_onboarding $mac started failed
				update_status_for_onboarding $mac waiting_for_DI_success stopped
				update_status_for_onboarding $mac waiting_for_TO_success stopped
				mac_di_partion_failed+=($mac_raw)

				device_count=$(cat $SCRIPT_DIR/.debug_device_count | awk '{print $1}') >/dev/null 2>&1
				device_count=$((device_count + 1))
				echo $device_count >$SCRIPT_DIR/.debug_device_count
				echo "Partitions creation failed for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
				delete_work_flow_sut $mac
				g_pt_count=$((g_pt_count + 1))
				break
			elif [[ $(kubectl logs -n $tink_stack_name $boot_pod --since=1m | grep -F workflow-$mac_raw | grep -F "actionName" | grep -F "store-Alpine" | grep -F "os-installation-di" | grep -F "status" | grep -F -c "STATE_SUCCESS") -ge 1 ]] && ! echo "${mac_di_partion_success[@]}" | grep -F -qw $mac_raw; then

				mac_di_partion_success+=($mac_raw)

				dev_serial=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $mac | awk '{print $3}')
				#check for the di status for a successful partition creation
				verify_fdo_di_flow_status $dev_serial &
				g_pt_count=$((g_pt_count + 1))
				echo "Partitions creation success for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
				break
			else
				continue
			fi

		done
		sleep 2
	done
}
#Check for the FDO Device installation status (di)
function verify_fdo_di_flow_status {
	Device_serial+=$1

	while [ $g_di_count -lt ${#Device_serial[@]} ]; do
		for dev_serial in "${Device_serial[@]}"; do
			echo "$Device_serial --------$dev_serial          ---"
			if [[ $(kubectl logs -n $tink_stack_name $boot_pod | grep -F -c "$dev_serial CLIENT_SDK_TPM_DI_SUCCESSFUL") -ge 1 ]] && ! echo "${Success_devserial_list[@]}" | grep -F -qw $dev_serial; then

				Success_devserial_list+=($dev_serial)
				mac=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $dev_serial | awk '{print $1}')
				update_status_for_onboarding $mac started success
				update_status_for_onboarding $mac waiting_for_DI_success started
				echo "device initialization success for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
				#extend the vouche for successful di SUTS
				extend_voucher_for_successful_di_suts $dev_serial &
				g_di_count=$((g_di_count + 1))
				break

			elif [[ $(kubectl logs -n $tink_stack_name $boot_pod | grep -F -c "$dev_serial CLIENT_SDK_TPM_DI_FAILED") -ge 1 ]] && ! echo "${Faild_devserial_list[@]}" | grep -F -qw $dev_serial; then

				Faild_devserial_list+=("$dev_serial")
				mac=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $dev_serial | awk '{print $1}')
				update_status_for_onboarding $mac started failed
				update_status_for_onboarding $mac waiting_for_DI_success stopped
				update_status_for_onboarding $mac waiting_for_TO_success stopped
				g_di_count=$((g_di_count + 1))
				device_count=$(cat $SCRIPT_DIR/.debug_device_count | awk '{print $1}') >/dev/null 2>&1
				device_count=$((device_count + 1))
				echo $device_count >$SCRIPT_DIR/.debug_device_count
				echo "device initialization failed for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
				delete_work_flow_sut $mac
				break
			else
				continue
			fi
		done
		sleep 25
	done
}
function fdo_check_to_completed() {
	local device_serial=$1

	sleep 2

	# get the GUID from the file device_serial_guid.txt
	guid_value=$(cat "${device_serial}_guid.txt")

	# {"to2CompletedOn" : "2023-07-28 16:46:17.246","to0Expiry" : "2023-07-28 16:46:11.597"}
	response=$(curl --location --request GET "https://$host_ip:8043/api/v1/owner/state/$guid_value" \
		--cacert /home/$USER/pri-fidoiot/component-samples/demo/owner/secrets/ca-cert.pem \
		--cert /home/$USER/pri-fidoiot/component-samples/demo/owner/secrets/api-user.pem 2>&1)
	value=$(grep -o "\"to2CompletedOn\" : \".*\"," <<<$response | awk ' { print $3" "$4 } ')
	# echo $response
	if [ "$value" != '' ]; then
		echo "$dev_serial CLIENT_SDK_TPM_TO2_SUCCESSFUL"
	fi

	export http_proxy=$temp_http
	export https_proxy=$temp_https
	popd >/dev/null
}

#Check for the FDO T01/T02 status
function verify_fdo_to_flow_status {
	device_serial_map+=$1
	#print the Boot logs and search for the T01/T02 sucess messages
	pushd $HOME/pri-fidoiot/component-samples/demo/scripts >/dev/null
	temp_http=$http_proxy
	temp_https=$http_proxy

	unset http_proxy
	unset https_proxy

	while [ $g_to_count -lt ${#device_serial_map[@]} ]; do
		for dev_serial in "${device_serial_map[@]}"; do

			if [[ $(fdo_check_to_completed $dev_serial | grep -F -c "$dev_serial CLIENT_SDK_TPM_TO2_SUCCESSFUL") -ge 1 ]] && ! echo "${Success_TO2_list[@]}" | grep -F -qw $dev_serial; then

				Success_TO2_list+=($dev_serial)

				mac=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $dev_serial | awk '{print $1}')

				update_status_for_onboarding $mac started success
				update_status_for_onboarding $mac waiting_for_TO_success started

				echo "T02 success for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME

				g_to_count=$((g_to_count + 1))
				break
				###			elif [[ $(kubectl logs -n $tink_stack_name $boot_pod | grep -F -c "$dev_serial CLIENT_SDK_TPM_TO2_FAILED") -ge 1 ]] && ! echo "${Faild_TO2_list[@]}" | grep -F -qw $dev_serial; then
				###
				###				Faild_TO2_list+=($dev_serial)
				###
				###				mac=$(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep $dev_serial | awk '{print $1}')
				###				update_status_for_onboarding $mac started failed
				###				update_status_for_onboarding $mac waiting_for_TO_success stopped
				###				g_to_count=$((g_to_count + 1))
				###				device_count=$(cat $SCRIPT_DIR/.debug_device_count | awk '{print $1}') >/dev/null 2>&1
				###				device_count=$((device_count + 1))
				###				echo $device_count >$SCRIPT_DIR/.debug_device_count
				###				echo "T02 failed for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
				###				delete_work_flow_sut $mac
				###				break
			else
				continue
			fi
		done
		sleep 15
	done
	export http_proxy=$temp_http
	export https_proxy=$temp_https
	popd >/dev/null
}
#This is to check the final OS installation on SUT
function check_production_work_flow_success() {
	mac=$1
	mac_fmt=$(echo "$mac" | sed 's/://g')

	while [ 1 ]; do
		if [[ $(kubectl get workflow -n $tink_stack_name | grep -F -i workflow-${mac_fmt}-prod | awk '{print $3}') = "STATE_SUCCESS" ]]; then

			update_status_for_onboarding $mac started success
			device_count=$(cat $SCRIPT_DIR/.debug_device_count | awk '{print $1}') >/dev/null 2>&1
			device_count=$((device_count + 1))
			echo $device_count >$SCRIPT_DIR/.debug_device_count
			echo "onboarding success  for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
			delete_work_flow_sut $mac
			break

		elif [[ $(kubectl get workflow -n $tink_stack_name | grep -F -i workflow-${mac_fmt}-prod | awk '{print $3}') = "STATE_FAILED" ]]; then

			update_status_for_onboarding $mac started failed
			device_count=$(cat $SCRIPT_DIR/.debug_device_count | awk '{print $1}') >/dev/null 2>&1
			device_count=$((device_count + 1))
			echo $device_count >$SCRIPT_DIR/.debug_device_count
			echo "onboarding failed  for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
			delete_work_flow_sut $mac
			break
		else
			sleep 40
			continue
		fi
	done
	sleep 10
}

#This is to delete the work-flow if its already exist for the sut that we are going to provision
function delete_work_flow_sut() {
	sut_mac_raw=$1
	sut_mac=$(echo "$sut_mac_raw" | sed 's/://g')
	#delete Template work-flows

	kubectl delete template fdodi-$sut_mac -n tink-system >/dev/null 2>&1
	kubectl delete template fdoto-$sut_mac -n tink-system >/dev/null 2>&1
	kubectl delete template focal-ms-$sut_mac-prod -n tink-system >/dev/null 2>&1

	#delete workflow
	kubectl delete workflow workflow-$sut_mac -n tink-system >/dev/null 2>&1
	kubectl delete workflow workflow-$sut_mac-prod -n tink-system >/dev/null 2>$1


	#delete hardware
	kubectl delete hardware machine-$sut_mac -n tink-system >/dev/null 2>$1
}

#Apply the device initialization(FDO di) to user provided SUT list
function apply_fdo_Di_flows_to_sut() {
	sut_onboarding_list=$1
	while read line; do
		[[ "$line" =~ ^#.*$ ]] || [[ "$line" = "" ]] && continue
		sut_name=$(echo $line | awk '{ print $1 }')
		mac=$(echo $line | awk '{ print $2 }')
		load_balancer_ip=$(echo $line | awk '{ print $3 }')
		sut_ip=$(echo $line | awk '{ print $4 }')
		disktype=$(echo $line | awk '{ print $5 }')
		bkctype=$(echo $line | awk '{ print $6 }')
		ms_id_scope=$(echo $line | awk '{ print $7 }')
		ms_reg_id=$(echo $line | awk '{ print $8 }')
		ms_smy_key=$(echo $line | awk '{ print $9 }')
		ms_smy_key=$(echo "$ms_smy_key" | sed 's/\//\\\//g')
		
		#check the if azure credetails passed as part of the sut list file for MS 
		if [[ $bkctype = "prod_focal-ms" ]]; then 
		    if [ -z $ms_id_scope ] || [ -z $ms_reg_id ] || [ -z $ms_smy_key ]; then
		        echo "looks MS credeatils are missing please check for the mac :$mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
		        continue
		    else
			if [ ! -f /opt/hook/azure_dps_installer.sh ] || [ ! -f /opt/hook/log.sh ];  then
                            sudo cp azur_iot_edge_installer/azure_dps_installer.sh /opt/hook/
                            sudo cp azur_iot_edge_installer/log.sh /opt/hook/
			 fi

			#update the azure-credential env file 
		        cp azur_iot_edge_installer/azure-credentials.env azure-credentials.env_$mac
		        sed -i "s/export ID_SCOPE=\"\"/export ID_SCOPE=\"$ms_id_scope\"/g" azure-credentials.env_$mac
			sed -i "s/export REGISTRATION_ID=\"\"/export REGISTRATION_ID=\"$ms_reg_id\"/g" azure-credentials.env_$mac
			sed -i "s/export SYMMETRIC_KEY=\"\"/export SYMMETRIC_KEY=\"$ms_smy_key\"/g" azure-credentials.env_$mac
			sudo cp azure-credentials.env_$mac /opt/hook
			rm azure-credentials.env_$mac
		    fi	
	        fi

		#check for the valid MAC,Disk type from the input list
		if [ ! -z $sut_name ] && [ ! -z $mac ] && [ ! -z $load_balancer_ip ] && [ ! -z $sut_ip ] && [ ! -z $disktype ] && [ ! -z $bkctype ] && [ ! -z $host_ip ]; then
			devicebool=true
			#check mac is proper or not before procced
			if [[ $mac =~ ^([[:xdigit:]][[:xdigit:]]:){5}[[:xdigit:]][[:xdigit:]]$ ]]; then
				macbool=true
			else
				macbool=false
				echo "Looks invalid macaddress  provided  for the sut:- $sut_name mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
			fi
			#check if the sutip is porper or not before procced 
			if [[ $sut_ip =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
		           	ipbool=true
			else
				ipbool=false
				echo "Looks invalid sutip provided  for the sut_name:- $sut_name sut_ip:- $sut_ip" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
			fi

			#check for the load balancer ip correct or not
			lb_ip=$(hostname -I | tr ' ' '\n' | grep -v "^127" | head -n 2 | awk 'NR==2 {print $1}')
			if [[ "$load_balancer_ip" = "$lb_ip" ]]; then
				lbipbool=true
			else
				lbipbool=false
				echo "Looks invalid load balancer ip provided  for the sut_name:- $sut_name load_balancer_ip:- $load_balancer_ip" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME

			fi

			case "$disktype" in
			/dev/sda*)
				diskbool=true
				;;

			/dev/sdb*)
				diskbool=true
				;;

			/dev/nvme*)
				diskbool=true
				;;
			*)
				diskbool=false
				echo "Looks invalid disk type-> $disktype  provided  for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
				;;

			esac

			#if we found successful mac,load_balancer_ip,sut_ip and disk_type proceed for the device initialization
			if [ ! -z "${macbool}" ] && [ ! -z "${diskbool}" ] && [ "${macbool}" = "true" ] && [ "${diskbool}" = "true" ] && [ "${devicebool}" = "true" ] && [ "${ipbool}" = "true" ] && [ "${lbipbool}" = "true" ]; then

				cd $SCRIPT_DIR/tinker_workflows
				chmod +x install_manifests_intel.sh
				chmod 755 *

				#check for dupilcate mac and process only one time
				if [ -f $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt ] && [ $(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep -c $mac) -ge 1 ]; then
					echo "dupilicate mac id provided  for the sut_name:- $sut_name , $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
					continue
				fi

				#check for dupilcate sut ip and process only one time
                                if [ -f $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt ] && [ $(cat $SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt | grep -c $sut_ip) -ge 1 ]; then
                                        echo "dupilicate sut_ip provided  for the sut_name:- $sut_name" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
                                        continue
                                fi
				#check if work-flow already exist for the given sut mac , if yes delete it
				delete_work_flow_sut $mac

				#Initiate the persistent fdo disk setup and DI workflow
				devSerial=$(./install_manifests_intel.sh $host_ip $load_balancer_ip $sut_ip $mac di $disktype $fdo_client | grep -i Serial | awk '{ print $6 }')

				if [ -z $devSerial ]; then
					echo "DI command not successful for for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
					continue
				fi

				echo "DI command Initiated successful for for the mac:- $mac" >>$SCRIPT_DIR/$SETUP_LOG_FILENAME
				DevSerial+=("$devSerial")

				echo "$sut_name $mac "acquireing_ip"  "initiated" "waiting_for_DI_success" "waiting_for_TO_success"" >>$SCRIPT_DIR/.debug_status.txt

				#Store the details for which mac the serail number generated.
				echo "$mac : $devSerial : $sut_ip : $disktype : $bkctype" >>$SCRIPT_DIR/.debug_mac_to_serial_map_for_diflow.txt

				#verify FDO disk partion status
				verify_fdo_di_disk_partion_status $mac &

				total_devices_to_provsion=$((total_devices_to_provsion + 1))
				echo $total_devices_to_provsion >$SCRIPT_DIR/.debug_total_sut
				sleep 3
				update_status_for_onboarding $mac acquireing_ip $sut_ip
				cd -
			fi
		else
			devicebool=false
			continue
		fi

	done <$sut_onboarding_list
}
#Apply the T01 && T02 flows for the successful Di devices
function apply_fdo_TO_flows_to_sut() {
	mac_id=$1
	sut_ip=$2
	disk_type=$3
	img_type=$4
	success_dev_serial=$5

	cd $SCRIPT_DIR/tinker_workflows

	#executing the T02 workflow
	##  	./install_manifests_intel.sh $host_ip $load_balancer_ip $sut_ip $mac_id to $disk_type $img_type
	verify_fdo_to_flow_status "$success_dev_serial" &
	wait

	# do zproduction flow sstart  and check
	./install_manifests_intel.sh $host_ip $load_balancer_ip $sut_ip $mac_id $img_type $disk_type $fdo_client
	check_production_work_flow_success $mac &

	cd -
	sleep 2
}
#This is get the system ip details
function get_system_ip_details() {
	sudo apt install net-tools -y >/dev/null 2>&1
	pub_inerface_name=$(route | grep '^default' | grep -o '[^ ]*$')
	host_ip=$(ifconfig "${pub_inerface_name}" | grep 'inet ' | awk '{print $2}')

}
#Delete the debug text files which were created previosuly
function clenup_files() {
	rm $SCRIPT_DIR/.debug*
	>$SCRIPT_DIR/$SETUP_LOG_FILENAME
}
#Update the flow status for DI,T01,T02 && on Boarding task for the sut
function update_status_for_onboarding() {
	mac=$1
	present_state=$2
	update_state=$3

	cat $SCRIPT_DIR/.debug_status.txt | sed -i "/$mac/s/$present_state/$update_state/g" $SCRIPT_DIR/.debug_status.txt >/dev/null 2>&1
}
#Setup the Print Screen for the status
function setup_print_screen() {
	echo "SUT_ID  MAC   SUT_IP  DI-FLOW  TO-FLOW   DEV-ONBOARD" >>$SCRIPT_DIR/.debug_status.txt
	print_to_screen $SCRIPT_DIR/.debug_status.txt &
}
#Kill back ground processer and remove all unwanted files created
function clean_up_setup() {
	echo "cleanup"
	killall zero_touch_onboarding_installation.sh
}
#This is to print the ongoig status for DI,TO1,TO2,Onboarding status to User
function print_to_screen() {
	out_file=$1
	# Use awk to create a tabular format with lines
	while [ 1 ]; do
		awk 'BEGIN { print "+----------+----------+----------+----------+----------+----------+----------+---------+-------------+-------------------+" }
     	           { printf("| %-12s | %-18s | %-18s | %-12s | %-22s | %-22s| \n", $1, $2, $3, $4, $5, $6) }
     	END       { print   "+----------+----------+----------+----------+----------+----------+----------+---------+-------------+-------------------+" }' $out_file
		cat $SCRIPT_DIR/$SETUP_LOG_FILENAME
		sleep 5
		clear
	done

}
#wait for all devices onboarding status, once all sut's done exit from the script
function wait_for_all_devices_onboarding {
	echo 0 >$SCRIPT_DIR/.debug_device_count
	while [ 1 ]; do
		dev_count=$(cat $SCRIPT_DIR/.debug_device_count | awk '{print $1}') >/dev/null 2>&1
		total_sut=$(cat $SCRIPT_DIR/.debug_total_sut | awk '{print $1}') >/dev/null 2>&1
		#Break the loop once all devices on boarding success or faild .
		if [[ $dev_count -eq $total_sut ]]; then
			sleep 60
			clean_up_setup
		else
			sleep 10
		fi
	done
}
#This is start the onboarding services on the PD system
function start_device_onboarding_services() {
	check_pre_condition

	clenup_files

	get_system_ip_details

	setup_FDO_services_on_pd

	setup_FDO_actions_on_pd

	setup_print_screen

	start_device_onboarding
}
###MAIN####
#Parse the arguments from commmand line
trap 'clean_up_setup' EXIT

start_device_onboarding_services

echo "Done with the installation,Please check  .onboarding_installation_status file for full details"
echo ""
echo "Please Check onboarding_logs.txt file for device onboarding status"
