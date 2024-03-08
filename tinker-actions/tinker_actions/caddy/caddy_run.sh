#!/bin/sh -e

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

if [ ! -s "/dev/shm/idp_access_token" ];
then
    echo "Orchestrator access token file is empty, exiting.."
    exit 1
fi

if [ ! -s "/dev/shm/release_token" ];
then
    echo "Release service token file is empty, exiting.."
    exit 1
fi

export access_token=$(cat /dev/shm/idp_access_token)
export release_token=$(cat /dev/shm/release_token)

source /etc/hook/env_config

#update hosts if they were provided
extra_hosts_needed=$(printf '%s\n' "$EXTRA_HOSTS" | sed "s|,|\n|g")

echo -e "$extra_hosts_needed" >> /etc/hosts
echo "adding extras host mappings completed"

echo -e "http_proxy=$http_proxy" >> /etc/environment
echo -e "https_proxy=$https_proxy" >> /etc/environment
echo -e "no_proxy=$no_proxy" >> /etc/environment

. /etc/environment

# Update ca certificates
update-ca-certificates
echo "Added ca certificates to trust pool"

/usr/bin/caddy run --environ --config /etc/caddy/Caddyfile
