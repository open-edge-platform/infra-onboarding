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
#######

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
