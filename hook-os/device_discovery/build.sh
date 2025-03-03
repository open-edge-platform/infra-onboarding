#!/bin/bash -e

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0


# This script needs access to public GitHub repository for downloading necessary
# artifacts. If your network is behind a proxy, please configure the proxy
# details in the environment variable.
# export http_proxy=http://proxy-ip:proxy-port
# export https_proxy=http://proxy-ip:proxy-port

set -e

ver=latest

# Ensure environment variables are assigned
http_proxy=${http_proxy:-}
https_proxy=${https_proxy:-}
no_proxy=${no_proxy:-}

# build go binary
CGO_ENABLED=0
export CGO_ENABLED
export GOOS=linux
export GOARCH=amd64
export GOPRIVATE="github.com/intel/*,github.com/intel-tiber/*"

go build -v -o app

# Build the container
docker build --no-cache -f Dockerfile \
	--build-arg HTTP_PROXY="$http_proxy" \
	--build-arg HTTPS_PROXY="$http_proxy" \
	--build-arg NO_PROXY="$no_proxy" \
	--build-arg http_proxy="$http_proxy" \
	--build-arg https_proxy="$http_proxy" \
	--build-arg no_proxy="$no_proxy" \
	-t device-discovery:$ver .

printf "\rSaved the Docker image for device-discovery as device-discovery:%s\n" "$ver"
