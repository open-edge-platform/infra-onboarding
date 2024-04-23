#!/bin/bash
#####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2023 Intel Corporation                                              # 
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################


#####################################################################################
# #check if current boot was pxe if so then we are still in Device setup and device initialization
# out=$(grep "OS_MODE=provision" /proc/cmdline)
# if [ $? -ne 0 ];
# then
#     sleep 20
#     exit
# fi

####################################################################################
# drive detection
#source drive_detection.sh
#driveDetection
#if [ -z "$disk" ]; then
#    exit
#fi
#DRIVE=$disk


#####################################################################################
#move the nvme or the sda/sdb to the top of the boot orders
#source change_boot_order.sh
#configure_boot_order $DRIVE

####################################################################################
#delete the pile up HOOK OS partitions from bootMenu
while IFS= read -r boot_part_number; do
efibootmgr -b $boot_part_number -B
done < <(efibootmgr | grep -i hookos | awk '{print $1}'| cut -c 5-8 )

#####################################################################################
# ######## make PXE the last boot option possible.
pxe_boot_number=$(efibootmgr | grep -i "Bootcurrent" | awk '{print $2}')

boot_order=$(efibootmgr | grep -i "Bootorder" | awk '{print $2}')

# Convert boot_order to an array and remove , between the entries
IFS=',' read -ra boot_order_array <<< "$boot_order"

# Remove PXE boot entry from Array
final_boot_array=()
for element in "${boot_order_array[@]}"; do
    if [[ "$element" != "$pxe_boot_number" ]]; then
        final_boot_array+=("$element")
    fi
done

# Add the PXE  boot entry to the end of the boot order array
final_boot_array+=("$pxe_boot_number")

# Join the elements of boot_order_array into a comma-separated string
final_boot_order=$(IFS=,; echo "${final_boot_array[*]}")

#remove trail and leading , if preset
final_boot_order=$(echo "$final_boot_order" | sed -e  's/^,//;s/,$//' )

echo "final_boot order--->" $final_boot_order

# Update the boot order using efibootmgr
efibootmgr -o "$final_boot_order"

#Make UEFI boot as inactive 
efibootmgr -b $pxe_boot_number -A

echo "Made Disk as first boot and PXE boot order at end"
# #####################################################################################
