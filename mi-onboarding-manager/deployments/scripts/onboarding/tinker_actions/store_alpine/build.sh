#!/bin/bash

#####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2023 Intel Corporation                                              # 
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################


# This script needs access to public GitHub repository for downloading necessary
# artifacts. If your network is behind a proxy, please configure the proxy
# details in the environment variable.
# export http_proxy=http://proxy-ip:proxy-port
# export https_proxy=http://proxy-ip:proxy-port

#source ../../config

ver=latest


sed -i "s/tinkerbell_owner=[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+/tinkerbell_owner=$load_balancer_ip/g" store_alpine.sh
sed -i "s/pd_host_ip=[0-9]\+\.[0-9]\+\.[0-9]\+\.[0-9]\+/pd_host_ip=$pd_host_ip/g" store_alpine.sh

## # Build the container
docker build -f Dockerfile \
	--build-arg HTTP_PROXY=$http_proxy \
	--build-arg HTTPS_PROXY=$http_proxy \
	--build-arg NO_PROXY="$no_proxy" \
	--build-arg http_proxy=$http_proxy \
	--build-arg https_proxy=$http_proxy \
	--build-arg no_proxy="$no_proxy" \
	-t localhost:5015/store_alpine:$ver . --push

# Save the Docker image

docker tag localhost:5015/store_alpine:$ver store_alpine:$ver
printf "\rSaved the Docker image for store_alpine as store_alpine:$ver\n"
