#!/bin/bash

#########################################################################
###  Script to apply Ov and sv scripts
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

namespace="tink-system" # Replace 'your-namespace' with your actual namespace

device_serial=""
macid=""
pri_svc_ip=""
fdo_pir_script_dir="/home/$USER/pri-fidoiot/component-samples/demo/scripts"


fdo_ov_extend() {

	sleep 2
	pushd $fdo_pir_script_dir >/dev/null
	temp_http=$http_proxy
	temp_https=$http_proxy

	unset http_proxy
	unset https_proxy
	# extending to the same ip because we have rv owner and mfg co-located in the same server
	ret=$(bash ./extend_upload.sh -m sh -c ./secrets/ -e mtls -m $pri_svc_ip -o $pri_svc_ip -s $device_serial)

	#    : $(bash ./extend_upload.sh  -m sh -c ./secrets/ -e mtls -m $pri_svc_ip -o pri_svc_ip -s $device_serial 2>&1)
	sleep 3
	export http_proxy=$temp_http
	export https_proxy=$temp_https
	retval=$(echo $ret | grep "Success in triggering TO0 for $device_serial")

	popd >/dev/null

	if [ -z "$retval" ]; then
		echo "Failure in getting extended voucher for device with serial number $device_serial"
	else
		echo "Success in triggering TO0 for $device_serial"
	fi
}

function start_fdo_svi_script {

	cur_dir=$(pwd)
	pushd ../fdo-scripts/ >/dev/null

	temp_http=$http_proxy
	temp_https=$http_proxy
	unset http_proxy
	unset https_proxy

	chmod +x *.sh
	#bash  svi_script.sh -c $fdo_pir_script_dir/secrets -o $pri_svc_ip -p 8043 -s test.sh

	ret=$(bash svi_script.sh -c $fdo_pir_script_dir/secrets -o $pri_svc_ip -p 8043 -s test.sh)
	sleep 3
	export http_proxy=$temp_http
	export https_proxy=$temp_https
	popd >/dev/null

	retval=$(echo $ret | grep "Owner svi API is success 200")
	if [ -z "$retval" ]; then
		echo "$device_serial Owner svi API failure"
	else
		echo "$device_serial Owner svi API success 200"
	fi

}

main() {
	local arglen=$#

	if [ $arglen -eq 3 ]; then
		pri_svc_ip=$3
	elif [ $arglen -eq 2 ]; then
		pub_inerface_name=$(route | grep '^default' | grep -o '[^ ]*$')
		pri_svc_ip=$(ip addr show "${pub_inerface_name}" | grep 'inet ' | awk 'NR==1 {print $2}' | cut -d'/' -f1)
	else
		echo "Usage : apply_ov_sv.sh <serialNo> <macid> <fdo_service_ip (optional)> <fdo_pri_scripts dir (optional>)"
		exit 1
	fi

	if [ $arglen -eq 4 ]; then
		fdo_pir_dir=$4
	fi

	device_serial=$1
	macid=$2
	mac_str=$(echo $macid | tr -d ':')

	fdo_ov_extend
	start_fdo_svi_script
	## TODO add proper return value based on above functions

	exit 0
}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
	#	set -x #-euxo pipefail
	main "$@"
	echo "ov_sv done!"
fi
