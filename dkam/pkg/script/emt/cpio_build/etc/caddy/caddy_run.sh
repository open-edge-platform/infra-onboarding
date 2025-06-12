#!/bin/sh -e

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#set -x

TIMEOUT=600  # 10 minutes in seconds
INTERVAL=3  # Check every 3 seconds

start_time=$(date +%s)

while true; do
    # Check if the file exists and is non-empty
    if [ -s "/dev/shm/idp_access_token" ]; then
        echo "Access token file is present and non-empty."
        break
    fi

    current_time=$(date +%s)
    elapsed_time=$(( current_time - start_time ))
    if [ $elapsed_time -ge $TIMEOUT ]; then
        echo "Timed out waiting for the file to be non-empty."
        exit 1
    fi
    sleep $INTERVAL
done

if [ ! -s "/dev/shm/release_token" ]; then
    echo "Release service token file is empty, exiting.."
    exit 1
fi

access_token=$(cat /dev/shm/idp_access_token)
export access_token

release_token=$(cat /dev/shm/release_token)
export release_token

if [ ! -s "/dev/shm/project_id" ]; then
    echo "Project ID file is empty, exiting.."
    exit 1
fi

project_id=$(cat /dev/shm/project_id)
export project_id

# shellcheck source=/dev/null
.  /etc/emt/env_config

host_guid=$(cat /sys/class/dmi/id/product_uuid)
export host_guid
if [ -z "$host_guid" ]; then
    echo "Edge Node UUID is empty. exiting.."
    exit 1
fi


# # Update hosts if they were provided
# extra_hosts_needed=$(printf '%s\n' "$EXTRA_HOSTS" | sed "s|,|\n|g")

# printf '%s\n' "$extra_hosts_needed" >> /etc/hosts
# echo "Adding extra host mappings completed"

export http_proxy="$http_proxy"
export https_proxy="$https_proxy"
export no_proxy="$no_proxy"

export oci_release_svc="${oci_release_svc:-}"
export tink_stack_svc="${tink_stack_svc:-}"
export release_svc="${release_svc:-}"
export tink_server_svc="${tink_server_svc:-}"
export logging_svc="${logging_svc:-}"

cp /etc/idp/ca.pem /etc/pki/ca-trust/source/anchors/
# Update CA certificates
update-ca-trust
echo "Added CA certificates to trust pool"

exec /usr/bin/caddy run --config /etc/caddy/Caddyfile
