#!/bin/bash -e

#####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2024 Intel Corporation                                              #
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################

ver=latest

# Generate self signed certificate and key
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout caddy-key.pem -out caddy-cert.pem -subj "/CN=localhost/CN=localhost.internal1/CN=localhost.internal2"

## # Build the container
docker build -f Dockerfile -t caddy_proxy:$ver .

# Save the Docker image
docker tag caddy_proxy:$ver caddy_proxy:$ver
printf "\rSaved the Docker image for caddy proxy as caddy_proxy:$ver\n"

rm -r caddy-key.pem caddy-cert.pem
