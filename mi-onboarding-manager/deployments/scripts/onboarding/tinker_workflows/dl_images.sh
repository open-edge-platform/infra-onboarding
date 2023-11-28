#!/bin/bash

#########################################################################
###  Script to apply download image configuration to tinkerbell
###  Need to be run on Provisioner
###
###   Author : Nabendu Maiti <nabendu.bikash.maiti@intel.com>
#########################################################################
#########################################################################################
# INTEL CONFIDENTIAL
# Copyright (2023) Intel Corporation
#
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
#set -x

RED='\033[0;31m'
BCyan='\033[1;36m'
NC='\033[0m' # No Color

SETUP_LOG_FILENAME="onboarding_logs.txt"

manifests_dir="./manifests/"
namespacetmp="tink-system"

INTERNAL_CTRL=${INTERNAL_CONTROL:-true}

#apply_manifests() {
main() {

	if [ "$#" -eq 1 ]; then
		image_type=$1
	else
		echo "Usage : dl_images.sh <imagetype e.g. focal/jammy/nkc>"
		exit 1
	fi

	job_name=""
	namespace=${namespacetmp:-"tink-system"}

	if [ "$image_type" == "prod_bkc" ]; then
		current_dir=$(pwd) >/dev/null
		to_download_yes=false
		#chmod + get_dkam_image_url.sh
		#source get_dkam_image_url.sh
		#TODO:will remove the hardcoded URL once we have the interface with inventory namanger
		#BKC_URL="https://af01p-png.devtools.intel.com/artifactory/hspe-edge-png-local/ubuntu-base/20230911-1844/default/ubuntu-22.04-desktop-amd64+intel-iot-37-custom.img.bz2"
		bkc_link="$BKC_URL"

		cd ${current_dir}
		if [ -z $bkc_link ]; then
			echo "bkc_link from DKAM is empty Please check" >/dev/null

			exit 0
		fi

		filename_bz2=${bkc_link##*/}
		bkc_raw_gz=${filename_bz2%.*}.raw.gz

		## No checksum based checking of new file. BKC release image names are alwys unique.
		#Before going for download check for the image present under /opt/hook, if yes do not download it.
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

			#envsubst <"$manifests_dir"/image_dload/ubuntu-download_bkc.yaml >/tmp/ubuntu_bkc_dl.yaml
			cat "$manifests_dir"/image_dload/ubuntu-download_bkc.yaml >/tmp/ubuntu_bkc_dl.yaml
			sed -i "s|BKC_IMG_LINK|$bkc_link|g" /tmp/ubuntu_bkc_dl.yaml
			kubectl delete -n tink-system job.batch/download-ubuntu-bkc >/dev/null 2>&1
			kubectl delete -n tink-system configMap download-bkc-image >/dev/null 2>&1
			kubectl apply -n "$namespace" -f /tmp/ubuntu_bkc_dl.yaml >/dev/null 2>&1
			job_name="download-ubuntu-bkc"
		else
			echo -e "${BCyan} Using old downloaded bkc ${NC}"
		fi
		export TINKER_CLIENT_IMG=$bkc_raw_gz
		echo "export TINKERBELL_CLIENT_IMG=$TINKER_CLIENT_IMG" >./.img_variable.txt
		echo "Started download the BKC image,it might take 30minutes to 1hr Based on network speed Please wait"
	## TODO add proper redirection >> $current_dir/../$SETUP_LOG_FILENAME
	elif [ "$image_type" == "prod_jammy" ]; then
		cat "$manifests_dir"/image_dload/ubuntu-download_jammy.yaml >/tmp/ubuntu_jammy_dl.yaml
		kubectl apply -n "$namespace" -f /tmp/ubuntu_jammy_dl.yaml
		job_name="download-ubuntu-jammy"

	elif [ "$image_type" == "prod_focal" ]; then
		if [ ! -f "/opt/hook/focal-server-cloudimg-amd64.raw.gz" ]; then
			echo "Started download the Focal image,it might take few minutes Based on network speed Please wait" >>../$SETUP_LOG_FILENAME
			cat "$manifests_dir"/image_dload/ubuntu-download.yaml >/tmp/ubuntu_focal_dl.yaml
			kubectl apply -n "$namespace" -f /tmp/ubuntu_focal_dl.yaml >/dev/null 2>&1
			job_name="download-ubuntu-focal"
		fi

	elif [ "$image_type" == "prod_focal-ms" ]; then
		echo "its ms-focal"
		#check if the focal image is for MS or 0.5 release

		#if MS instalation set do below changes
		#TODO:This will be changed once we have interfce with Inventory manager
		#Download the kernel pkgs to install on Focal image to suporting the ethernet drivers

		if [ ! -f /opt/hook/linux-image-5.15.96-lts.deb ] || [ ! -f /opt/hook/linux-headers-5.15.96-lts.deb ]; then
			wget http://oak-07.jf.intel.com/ikt_kernel_deb_repo/pool/main/l/linux-5.15.96-lts-230421t211918z/linux-headers-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb --no-proxy >/dev/null 2>&1
			wget http://oak-07.jf.intel.com/ikt_kernel_deb_repo/pool/main/l/linux-5.15.96-lts-230421t211918z/linux-image-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb --no-proxy /dev/null 2>&1

			sudo mv linux-image-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb /opt/hook/linux-image-5.15.96-lts.deb
			sudo mv linux-headers-5.15.96-lts-230421t211918z_5.15.96-184_amd64.deb /opt/hook/linux-headers-5.15.96-lts.deb
		fi

		if [ ! -f "/opt/hook/focal-server-cloudimg-amd64.raw.gz" ]; then
			echo "Started download the Focal image,it might take few minutes Based on network speed Please wait" >>../$SETUP_LOG_FILENAME
			cat "$manifests_dir"/image_dload/ubuntu-download.yaml >/tmp/ubuntu_focal_dl.yaml
			kubectl apply -n "$namespace" -f /tmp/ubuntu_focal_dl.yaml >/dev/null 2>&1
			job_name="download-ubuntu-focal"
		fi
	else
		echo "Unknown option $image_type"
	fi

	if [ "$INTERNAL_CTRL" = true ] && [ "$job_name" ]; then
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

	if [ "$image_type" == "prod_bkc" ]; then
		#TODO: Remove hardcode URL once we have interface with telemetry agent
		#download the bkc_overlay script for installing the base pkgs on the system

		#BKC_BASEPKG="https://ubit-artifactory-sh.intel.com/artifactory/sed-dgn-local/yocto/dev-test-image/DKAM/IAAS/ADL/installer23WW37.5_1506.sh"
		base_pkg_script_url="$BKC_BASEPKG"

		script_name=${base_pkg_script_url##*/}

		if [ -f $script_name ]; then
			rm $script_name
		fi
		wget --no-proxy $base_pkg_script_url
		sudo cp $script_name /opt/hook/base_installer.sh
		#copy edge_node_installer.sh to /opt/hook directory for downloding it on to the edge node
		sudo cp edge_node_installer.sh /opt/hook
		#copy docker-compose files for the agnets to /opt/hook/ directory for downloding it on to the edge node

		if [  -d ../../../../../docker/edge-iaas-platform/agents ]; then
		    docker_file_dir=../../../../../docker/edge-iaas-platform/agents > /dev/null 2>&1
		else
		    docker_file_dir=../../../../../docker/edge-iaas-platform/platform-manager > /dev/null 2>&1
		fi
		sudo cp $docker_file_dir/inventory-agent/docker-compose.yml /opt/hook/docker-compose-inv.yml
		sudo cp $docker_file_dir/update-agent/docker-compose.yml /opt/hook/docker-compose-upd.yml
		
		#telemetry code changes

		if [ ! -f /opt/hook/telemetry_agent_files.tar ]; then
		    if [ -d telemetry_agent_files ]; then
		       rm -rf telemetry_agent_files >/dev/null 2>&1
		       rm -rf telemetry_agent_files.tar  >/dev/null 2>&1
		    fi
		    mkdir ${current_dir}/telemetry_agent_files
		    cd ${current_dir}/telemetry_agent_files 

		    #copy the files for otelcol_agent
		    mkdir otelcol_agent
		    cp ../../../../../../bkc/edge-iaas-telemetry/platform-manager/telemetry/onboarding_deployment/deploy-iaas-telemetry                otelcol_agent
		    cp ../../../../../../docker/edge-iaas-platform/platform-director/telemetry/otelcol/config-edge-agent.yaml otelcol_agent
		    cp ../../../../../../docker/edge-iaas-platform/platform-director/telemetry/otelcol/docker-compose.yml otelcol_agent
		    #copy the files for telemetry_agent
		    mkdir telemetry_agent

		    cp ../../../../../../bkc/edge-iaas-telemetry/platform-manager/telemetry/onboarding_deployment/deploy-iaas-telemetry                telemetry_agent
		    cp ../../../../../../docker/edge-iaas-platform/platform-manager/telemetry/telemetry-agent/docker-compose.yml telemetry_agent
		    cp ../../../../../../cmd/telemetryagent/config.yaml telemetry_agent
		    cd telemetry_agent && mkdir iaas-telemetry 
		    cp ../../../../../../../docker/edge-iaas-platform/platform-manager/telemetry/fluentbit/configuration/fluent-bit-common.conf iaas-telemetry
		    cp ../../../../../../../docker/edge-iaas-platform/platform-manager/telemetry/telegraf/configuration/telegraf-iaas-default.conf iaas-telemetry
		    cd iaas-telemetry && mkdir iaas-telemetry
		    cd iaas-telemetry 
		    mkdir fluentbit
		    cp ../../../../../../../../../docker/edge-iaas-platform/platform-manager/telemetry/fluentbit/docker-compose.yaml fluentbit
		    mkdir telegraf
		    cp ../../../../../../../../../docker/edge-iaas-platform/platform-manager/telemetry/telegraf/docker-compose.yml telegraf
	            #tar the telemetry-agent files and copy to /opt/hook folder
		    cd ${current_dir}
		    tar -cvf telemetry_agent_files.tar telemetry_agent_files
		    sudo cp telemetry_agent_files.tar /opt/hook/
		fi
	fi

}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
	#	set -euxo pipefail

	main "$@"
	echo "dload done!"
fi
