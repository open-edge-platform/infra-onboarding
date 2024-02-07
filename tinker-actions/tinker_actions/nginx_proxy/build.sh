#!/bin/bash

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

#root_ca_cert="/usr/local/share/ca-certificates/ensp-orchestrator-ca.crt"
# Generate self signed certificate and key
openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout nginx-key.pem -out nginx-cert.pem -subj "/CN=nginx-proxy"

## # Build the container
docker pull nginx:latest
docker build -f Dockerfile \
        -t nginx_proxy_action:$ver .

# Save the Docker image

docker tag nginx_proxy_action:$ver nginx_proxy_action:$ver
printf "\rSaved the Docker image for nginx client proxy as nginx_proxy:$ver\n"

rm -r nginx-key.pem nginx-cert.pem
