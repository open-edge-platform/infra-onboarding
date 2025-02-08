#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

ver=latest

## # Build the container
docker build -f Dockerfile \
	--no-cache \
	--build-arg HTTP_PROXY=$http_proxy \
	--build-arg HTTPS_PROXY=$http_proxy \
	--build-arg NO_PROXY="$no_proxy" \
	--build-arg http_proxy=$http_proxy \
	--build-arg https_proxy=$http_proxy \
	--build-arg no_proxy="$no_proxy" \
	-t tiberos_partition:$ver .

# Save the Docker image
docker tag tiberos_partition:$ver tiberos_partition:$ver
printf "\rSaved the Docker image for tiberos_partition as tiberos_partition:$ver\n"

