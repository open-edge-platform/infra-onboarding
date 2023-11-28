#!/bin/bash
#########################################################################################
#  Script to stop &  clean FDO pri services and installed docker-compose components
#  Need to be run on Provisioner
#  It run Mnufacturing service, Owner service, Rv service
#   Author : Nabendu Maiti <nabendu.bikash.maiti@intel.com>
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


set -x


dcv1=$(docker-compose)
# check docker compose v1 or v2
if [ $dcv1 -eq 0 ] ; then
  dc='docker-compose --ansi=never'
else
  dc='docker compose'
fi


# Clone FDO code
# *** Modify below variables ***
export USER=
if [ -z "${USER}" ]; then
	read -p "Please enter the local username: " USER
fi
export USER=$USER
export HOME=/home/$USER

cd $HOME
if [ -d /home/$USER/pri-fidoiot ]; then
	# Stop running containers
	cd /home/$USER/pri-fidoiot/component-samples/demo/db
	$dc down
	cd ../rv
	$dc down
	cd ../manufacturer
	$dc down
	cd ../owner
	$dc down
	# Delete old secrets and files
	sudo rm -rf /home/$USER/pri-fidoiot
fi

rm -rf /home/$USER/error_log_FDO
