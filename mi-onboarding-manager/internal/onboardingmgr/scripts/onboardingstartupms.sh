#!/bin/bash

#########################################################################################
# INTEL CONFIDENTIAL
# Copyright (2023) Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# The source code contained or described herein and all documents related to the source
# code("Material") are owned by Intel Corporation or its suppliers or licensors. Title
# to the Material remains with Intel Corporation or its suppliers and licensors. The
# Material contains trade secrets and proprietary and confidential information of Intel
# or its suppliers and licensors. The Material is protected by worldwide copyright and
# trade secret laws and treaty provisions. No part of the Material may be used, copied,
# reproduced, modified, published, uploaded, posted, transmitted, distributed, or
# disclosed in any way without Intel's prior express written permission.
#
# No license under any patent, copyright, trade secret or other intellectual property
# right is granted to or conferred upon you by disclosure or delivery of the Materials,
# either expressly, by implication, inducement, estoppel or otherwise. Any license under
# such intellectual property rights must be express and approved by Intel in writing.
#########################################################################################
# shellcheck disable=all
SETUP_LOG_FILENAME="onboarding_logs.txt"
image_type=$1
ms_install=$2
job_name=""
namespace="tink-system"

# MS_INSTALL="false"
FILE=env_variable.txt
TINKER_CLIENT_IMG=""

bkc_raw_gz=""
if [ "$image_type" == "bkc" ]; then
	current_dir=$(pwd) >/dev/null
	to_download_yes=false
	#chmod + get_dkam_image_url.sh
	#source get_dkam_image_url.sh
	#TODO:will remove the hardcoded URL once we have the interface with inventory namanger 
	bkc_link="https://af01p-png.devtools.intel.com/artifactory/hspe-edge-png-local/ubuntu-base/20230911-1844/default/ubuntu-22.04-desktop-amd64+intel-iot-37-custom.img.bz2"

	cd "${current_dir}" || exit
	if [ -z $bkc_link ]; then
		echo "bkc_link from DKAM is empty Please check" >/dev/null

		exit 0
	fi

	filename_bz2=${bkc_link##*/}
	bkc_raw_gz=${filename_bz2%.*}.raw.gz

	# No checksum based checking of new file. BKC release image names are alwys unique.
	# Before going for download check for the image present under /opt/hook, if yes do not download it.
	if [ -e "/opt/hook/$bkc_raw_gz" ]; then
		to_download_yes=false
	else
		#Image not present and downlod it from DKAM
		to_download_yes=true
	fi

	if [ "$to_download_yes" = true ]; then
		# todo wait if previous one completed or create new job
		echo "Started download the UBUNTU image,it might take few minutes Based on network speed Please wait" >>../$SETUP_LOG_FILENAME
		echo -e "${RED}Going to use new downloaded bkc ${NC}" >/dev/null

		export TINKER_CLIENT_IMG=$bkc_raw_gz
		export BKC_IMG_LINK=$bkc_link

		#envsubst <../workflows/manifests/image_dload/ubuntu-download_bkc.yaml >/tmp/ubuntu_bkc_dl.yaml
		cat ../workflows/manifests/image_dload/ubuntu-download_bkc.yaml >/tmp/ubuntu_bkc_dl.yaml
		sed -i "s|BKC_IMG_LINK|$bkc_link|g" /tmp/ubuntu_bkc_dl.yaml
		kubectl delete -n tink-system job.batch/download-ubuntu-bkc >/dev/null 2>&1
		kubectl delete -n tink-system configMap download-bkc-image >/dev/null 2>&1
		kubectl apply -n "$namespace" -f /tmp/ubuntu_bkc_dl.yaml >/dev/null 2>&1
		job_name="download-ubuntu-bkc"
	else
		echo -e "${BCyan} Using old downloaded bkc ${NC}"
	fi
	export TINKER_CLIENT_IMG=$bkc_raw_gz
	echo "Started download the BKC image,it might take 30minutes to 1hr Based on network speed Please wait" 
	## TODO add proper redirection >> $current_dir/../$SETUP_LOG_FILENAME
elif [ "$image_type" == "jammy" ]; then
	cat ../workflows/manifests/image_dload/ubuntu-download_jammy.yaml >/tmp/ubuntu_jammy_dl.yaml
	kubectl apply -n "$namespace" -f /tmp/ubuntu_jammy_dl.yaml >/dev/null 2>&1
	job_name="download-ubuntu-jammy"

elif [ "$image_type" == "focal" ]; then
	#check if the focal image is for MS or 0.5 release

	#if MS instalation set do below changes
	#TODO:This will be changed once we have interfce with Inventory manager
	if [ "$ms_install" = "ms" ]; then
	    #Download the kernel pkgs to install on Focal image to suporting the ethernet drivers 

	     if [ ! -f /opt/hook/linux-image-5.15.96-lts.deb ] || [ ! -f /opt/hook/linux-headers-5.15.96-lts.deb ]; then
	         wget http://oak-07.jf.intel.com/ikt_kernel_deb_repo/pool/main/l/linux-5.15.96-lts-230421t211918z/linux-headers-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb --no-proxy > /dev/null 2>&1
	         wget http://oak-07.jf.intel.com/ikt_kernel_deb_repo/pool/main/l/linux-5.15.96-lts-230421t211918z/linux-image-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb --no-proxy /dev/null 2>&1

	        sudo mv linux-image-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb /opt/hook/linux-image-5.15.96-lts.deb
	        sudo mv linux-headers-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb /opt/hook/linux-headers-5.15.96-lts.deb
	     fi
	     #copy the Iot_Edge_Installer.sh,log.sh and azure_env file to /opt/hook
         
	     if [ ! -f /opt/hook/azure_dps_installer.sh ] || [ ! -f /opt/hook/log.sh ] || [ ! -f /opt/hook/azure-credentials.env ]; then
	         sudo cp  ../azur_iot_edge_installer/azure_dps_installer.sh /opt/hook/
	         sudo cp  ../azur_iot_edge_installer/log.sh /opt/hook/

	         #Before copying the azure-env file please update the ID_SCOPE,REGISTRATION_ID,SYMMETRIC_KEY in env file
	         #get the details from IVM manager and update it
	         #TODO Remove the below once we have interface with inventory manager
	         ID_SCOP="0ne00AF26BF"
	         REGISTRATIN_ID="device-2"
	         SYMMETRC_KEY="xrlXsQwFUf1+7hqPT2wSwCKMWooST666k\/s8Z9U8H3ZeyuxMnEeqzYjeQUL77JyaEvh2j0\/DgueKa1W9C9LNdw=="
	         sed -i "s/export ID_SCOPE=\"\"/export ID_SCOPE=\"$ID_SCOP\"/g" ../azur_iot_edge_installer/azure-credentials.env
	         sed -i "s/export REGISTRATION_ID=\"\"/export REGISTRATION_ID=\"$REGISTRATIN_ID\"/g" ../azur_iot_edge_installer/azure-credentials.env 
	         sed -i "s/export SYMMETRIC_KEY=\"\"/export SYMMETRIC_KEY=\"$SYMMETRC_KEY\"/g" ../azur_iot_edge_installer/azure-credentials.env
		sudo cp  ../azur_iot_edge_installer/azure-credentials.env /opt/hook 
	     fi
	 fi
        if [ ! -f "/opt/hook/focal-server-cloudimg-amd64.raw.gz" ]; then
	     echo "Started download the Focal image,it might take few minutes Based on network speed Please wait" >>../$SETUP_LOG_FILENAME
	    cat ../workflows/manifests/image_dload/ubuntu-download.yaml >/tmp/ubuntu_focal_dl.yaml
	    kubectl apply -n "$namespace" -f /tmp/ubuntu_focal_dl.yaml >/dev/null 2>&1
	    job_name="download-ubuntu-focal"
	fi
fi

if [ "$job_name" ]; then
	# Wait until the job is completed
	while true; do
		job_status=$(kubectl get job "$job_name" -n "$namespace" -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}')
		if [[ "$job_status" == "True" ]]; then
			echo "$job_name completed successfully."
			break
		fi
		echo "$job_name is still running. Waiting..." >/dev/null
		sleep 45
	done
fi

if [ "$image_type" == "bkc" ]; then
    #TODO: Remove hardcode URL once we have interface with telemetry agent
    #download the bkc_overlay script for installing the base pkgs on the system

    base_pkg_script_url="https://ubit-artifactory-sh.intel.com/artifactory/sed-dgn-local/yocto/dev-test-image/DKAM/IAAS/ADL/installer23WW37.5_1506.sh"

    script_name=${base_pkg_script_url##*/}

    if [ -f "$script_name" ]; then
       rm "$script_name"
    fi
    wget --no-proxy $base_pkg_script_url
    sudo cp "$script_name" /opt/hook/base_installer.sh
    #copy edge_node_installer.sh to /opt/hook directory for downloding it on to the edge node
    sudo cp edge_node_installer.sh /opt/hook
    #copy docker-compose files for the agnets to /opt/hook/ directory for downloding it on to the edge node
    sudo cp ../../../docker/edge-iaas-platform/agents/inventory-agent/docker-compose.yml /opt/hook/docker-compose-inv.yml
    sudo cp ../../../docker/edge-iaas-platform/agents/update-agent/docker-compose.yml /opt/hook/docker-compose-upd.yml
fi

if [ "$image_type" == "bkc" ]; then
	export ROOTFS_PART_NO=3
else
	export ROOTFS_PART_NO=1
fi

disk_dev=$disk #"/dev/nvme0n1"
npart=$(echo "$disk_dev" | grep '.*[0-9]$')
if [ "$npart" ]; then
	export ROOTFS_PARTITION="p$ROOTFS_PART_NO"
else
	export ROOTFS_PARTITION="$ROOTFS_PART_NO"
fi

export DISK_DEVICE="$disk_dev"
export TINKERBELL_CLIENT_IP="$worker_ip"
export TINKERBELL_CLIENT_MAC="$worker_mac"
export TINKER_CLIENT_IMG_TYPE=$image_type
export TINKERBELL_HOST_IP="$loadb_ip"
export PROVISIONER_HOST_IP="$host_ip"
export TINKERBELL_CLIENT_GW="$host_ip"

PRODUCT_UUID_FILE="/sys/class/dmi/id/product_uuid"
AGENT_HARDWARE_ID=$(sudo cat "$PRODUCT_UUID_FILE" 2>/dev/null)

if [ $? -ne 0 ]; then
    echo "Error: $PRODUCT_UUID_FILE does not exist or cannot be read."
    AGENT_HARDWARE_ID="NODE-CANNOT-READ-HARDWARE-ID"
fi

if [ ! -f /opt/hook/agent_node_env.txt ]; then
    #export port numbers and host_ip for the aganets to start on edge node
	{
		echo "export MGR_HOST=$host_ip" 
		echo "export NO_PROXY=$host_ip" 
		echo "export INVMGR_PORT=31846"
		echo "export UPDATEMGR_PORT=31845"
		echo "export UPDATEMGR_HOST=$host_ip"
		echo "export AGENT_HARDWARE_ID=$AGENT_HARDWARE_ID"
	} > agent_node_env.txt
    sudo cp agent_node_env.txt /opt/hook/
    rm agent_node_env.txt
fi
echo "TINKER_CLIENT_IMG=\"$bkc_raw_gz\"" >> "$FILE"
echo "Variables added to $FILE:"
