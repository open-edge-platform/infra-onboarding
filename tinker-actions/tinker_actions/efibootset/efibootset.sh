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

remove_pxe=$(sed "s/$pxe_boot_number//g" <<< $boot_order)
remove_pxe=$(sed "s/,,/,/g" <<< $remove_pxe)

final_bootorder=$remove_pxe","$pxe_boot_number
final_bootorder=$(sed "s/,,/,/g" <<< $final_bootorder)
efibootmgr --bootorder $final_bootorder
echo "Made PXE the last in the boot order"
# #####################################################################################
