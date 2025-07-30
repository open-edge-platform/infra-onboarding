#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Read /proc/cmdline and extract string after worker_id
worker_id=""
for i in $(cat "/proc/cmdline"); do
    if [[ "$i" == worker_id=* ]]; then
        worker_id="${i#worker_id=}"
        break
    fi
done
echo "Worker ID: ${worker_id}"

SLEEP_TIME=3
NUMBER_OF_RETRIES=10

# Check if IP address is assigned to interface matching MAC address with worker_id
for iface in $(ls /sys/class/net); do
    mac_address=$(cat "/sys/class/net/$iface/address")
    if [[ "$mac_address" == "$worker_id" ]]; then
        for ((attempt=1; attempt<=NUMBER_OF_RETRIES; attempt++)); do
            ip_address=$(ip addr show "$iface" | awk '/inet / {print $2}' | cut -d/ -f1)
            if [[ -n "$ip_address" ]]; then
                echo "IP Address $ip_address is assigned to interface $iface with MAC $mac_address"
                exit 0
            else
                echo "Attempt $attempt/$NUMBER_OF_RETRIES: No IP address assigned to interface $iface with MAC $mac_address yet"
                sleep "$SLEEP_TIME"
            fi
        done
    fi
done

echo "No interface found with MAC address matching worker_id: $worker_id"
exit 1
