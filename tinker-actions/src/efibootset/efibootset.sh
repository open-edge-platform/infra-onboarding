#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

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
