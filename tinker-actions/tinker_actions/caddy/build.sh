#!/bin/bash -e

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

ver=latest

# Generate self signed certificate and key
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout caddy-key.pem -out caddy-cert.pem -subj "/CN=localhost/CN=localhost.internal1/CN=localhost.internal2"

## # Build the container
docker build -f Dockerfile -t caddy_proxy:$ver .

# Save the Docker image
docker tag caddy_proxy:$ver caddy_proxy:$ver
printf "\rSaved the Docker image for caddy proxy as caddy_proxy:$ver\n"

rm -r caddy-key.pem caddy-cert.pem
