#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# This script needs access to public GitHub repository for downloading necessary
# artifacts. If your network is behind a proxy, please configure the proxy
# details in the environment variable.
# export http_proxy=http://proxy-ip:proxy-port
# export https_proxy=http://proxy-ip:proxy-port

#source ../../config

ver=latest


## # Build the container
docker build -f Dockerfile \
	--build-arg HTTP_PROXY=$http_proxy \
	--build-arg HTTPS_PROXY=$http_proxy \
	--build-arg NO_PROXY="$no_proxy" \
	--build-arg http_proxy=$http_proxy \
	--build-arg https_proxy=$http_proxy \
	--build-arg no_proxy="$no_proxy" \
	-t image2disk:$ver .

# Save the Docker image
docker tag image2disk:$ver image2disk:$ver
printf "\rSaved the Docker image for image2disk as image2disk:$ver\n"
