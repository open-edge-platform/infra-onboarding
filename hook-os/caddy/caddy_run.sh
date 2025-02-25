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
        echo "File is present and non-empty."
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

if [ ! -s "/dev/shm/release_token" ];
then
    echo "Release service token file is empty, exiting.."
    exit 1
fi

export access_token=$(cat /dev/shm/idp_access_token)
export release_token=$(cat /dev/shm/release_token)

if [ ! -s "/dev/shm/project_id" ];
then
    echo "Project ID file is empty, exiting.."
    exit 1
fi

export project_id=$(cat /dev/shm/project_id)

export host_guid=$(cat /sys/class/dmi/id/product_uuid)
if [ -z "$host_guid" ]; then
    echo "Edge Node UUID is empty. exiting.."
    exit 1
fi

source /etc/hook/env_config

#update hosts if they were provided
extra_hosts_needed=$(printf '%s\n' "$EXTRA_HOSTS" | sed "s|,|\n|g")

echo -e "$extra_hosts_needed" >> /etc/hosts
echo "adding extras host mappings completed"

export http_proxy=$http_proxy
export https_proxy=$https_proxy
export no_proxy=$no_proxy

# Update ca certificates
update-ca-certificates
echo "Added ca certificates to trust pool"

# Define the log level based on the environment variable
IS_CADDY_DEBUG=$(cat /proc/cmdline | grep -o 'DEBUG=[^ ]*' | awk -F= '{print $2}')
if [ "$IS_CADDY_DEBUG" = "false" ]; then
    LOG_LEVEL="ERROR"
else
    LOG_LEVEL="DEBUG"
fi
cp /etc/caddy/Caddyfile /etc/caddy/Caddyfile2
# Replace the log level in the Caddyfile
sed -i "s/level .*/level $LOG_LEVEL/" /etc/caddy/Caddyfile2

/usr/bin/caddy run --config /etc/caddy/Caddyfile2
