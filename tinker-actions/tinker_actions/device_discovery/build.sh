#!/bin/bash -e

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

# build go binary
CGO_ENABLED=0
export CGO_ENABLED
export GOOS=linux
export GOARCH=amd64
export GOPRIVATE="github.com/intel-innersource/*"

go build -v -o app

# Build the container
docker build --no-cache -f Dockerfile \
	--build-arg HTTP_PROXY=$http_proxy \
	--build-arg HTTPS_PROXY=$http_proxy \
	--build-arg NO_PROXY="$no_proxy" \
	--build-arg http_proxy=$http_proxy \
	--build-arg https_proxy=$http_proxy \
	--build-arg no_proxy="$no_proxy" \
	-t device-discovery:$ver .

printf "\rSaved the Docker image for device-discovery as device-discovery:$ver\n"
