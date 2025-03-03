#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

ver=latest

set -e

# Ensure environment variables are assigned
http_proxy=${http_proxy:-}
https_proxy=${https_proxy:-}
no_proxy=${no_proxy:-}

# Generate self signed certificate and key
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout caddy-key.pem -out caddy-cert.pem -subj "/CN=localhost/CN=localhost.internal1/CN=localhost.internal2"

## # Build the container
docker build --no-cache -f Dockerfile \
    --build-arg HTTP_PROXY="$http_proxy" \
    --build-arg HTTPS_PROXY="$https_proxy" \
    --build-arg NO_PROXY="$no_proxy" \
    --build-arg http_proxy="$http_proxy" \
    --build-arg https_proxy="$https_proxy" \
    --build-arg no_proxy="$no_proxy" \
    -t caddy_proxy:$ver .

# Save the Docker image
docker tag caddy_proxy:$ver caddy_proxy:$ver
printf "\rSaved the Docker image for caddy proxy as caddy_proxy:%s\n" "$ver"

rm -r caddy-key.pem caddy-cert.pem