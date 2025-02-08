#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Function to list all removable disks and virtual media
list_removable_disks() {
    # List removable disks and virtual media (CD/DVD drives)
    local disks
    disks=$(lsblk -r -o NAME,RM,TYPE | awk '$2 == "1" && $3 == "disk" || $2 == "1" && $3 == "rom" {print $1}')

    if [ $? -ne 0 ]; then
        echo "Error: Failed to list removable disks."
        return 1
    fi

    # Return the found disks
    echo "$disks"
}

# Function to eject a given device
eject_device() {
    local device="$1"

    echo "Info: Ejecting device: $device"
    
    # Try to eject the device
    if echo 1 | tee /sys/block/$device/device/delete; then
        echo "Info: Successfully ejected $device"
        return 0
    else
        echo "Error: Ejection of device $device failed."
        return 1
    fi
}

# Function to eject all removable devices
eject_all_removable_devices() {

    # Get the list of removable disks and virtual media
    local devices=$(list_removable_disks)

    # Check if there are any devices to eject
    if [ -z "$devices" ]; then
        echo "Info: No removable disks or virtual media found."
        return 0
    fi

    # Eject each device and track failure
    for device in $devices; do
        # Check if ejection failed for any device
        if ! eject_device "$device"; then
            return 1
        else
            sleep 5
            if lsblk | grep -q "$device"; then
                echo "Error: Device $device is still present."
                return 1
            fi
        fi
    done

    return 0
}
