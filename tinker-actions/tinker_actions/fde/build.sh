#!/bin/bash -e

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0


# This script needs access to public GitHub repository for downloading necessary
# artifacts. If your network is behind a proxy, please configure the proxy
# details in the environment variable.
# export http_proxy=http://proxy-ip:proxy-port
# export https_proxy=http://proxy-ip:proxy-port


ver=latest


## # Build the container
docker build -f Dockerfile \
	--build-arg HTTP_PROXY=$http_proxy \
	--build-arg HTTPS_PROXY=$http_proxy \
	--build-arg NO_PROXY="$no_proxy" \
	--build-arg http_proxy=$http_proxy \
	--build-arg https_proxy=$http_proxy \
	--build-arg no_proxy="$no_proxy" \
	-t fde:$ver .

# Save the Docker image

docker tag fde:$ver fde:$ver
printf "\rSaved the Docker image for fde as fde:$ver\n"
