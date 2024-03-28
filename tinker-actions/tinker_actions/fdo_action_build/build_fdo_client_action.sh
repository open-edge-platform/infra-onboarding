#!/bin/bash -e

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

# This script is provided to build the FIDO Device Onboard (FDO) clients for the
# purpose of Tinkerbell Baremetal Onboarding.

# This script needs access to public GitHub repository for downloading necessary
# artifacts. If your network is behind a proxy, please configure the proxy
# details in the environment variable.
# export http_proxy=http://proxy-ip:proxy-port
# export https_proxy=http://proxy-ip:proxy-port

#source ../../config

# FDO Client Version
ver=latest


## # Build the FDO clients
docker build -f Dockerfile \
	--build-arg HTTP_PROXY=$HTTP_PROXY \
	--build-arg HTTPS_PROXY=$HTTPS_PROXY\
	--build-arg NO_PROXY="$NO_PROXY" \
	--build-arg http_proxy=$http_proxy \
	--build-arg https_proxy=$https_proxy \
	--build-arg no_proxy="$no_proxy" \
	-t fdoclient_action:$ver .

# Save the Docker image
docker tag fdoclient_action:$ver fdoclient_action:$ver
printf "\rSaved the Docker image for FDO Clients as fdoclient_action$ver.tar\n"
