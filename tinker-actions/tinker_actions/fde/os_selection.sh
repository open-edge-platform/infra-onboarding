#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#####################################################################################
get_partition_suffix() {
    part_variable=''
    ret=$(grep -i "nvme" <<< "$1")
    if [ $? == 0 ]
    then
	part_variable="p"
    fi

    echo $part_variable
}

#####################################################################################
#top level script to check and run fde for ubuntu or tiber
run_enable_fde()
{
    disk_device=""

    list_block_devices=($(lsblk -o NAME,TYPE,SIZE,RM | grep -i disk | awk '$1 ~ /sd*|nvme*/ {if ($3 !="0B" && $4 ==0)  {print $1}}'))
    for block_dev in ${list_block_devices[@]};
    do
	#if there were any problems when the ubuntu was streamed.
	printf 'OK\n'  | parted ---pretend-input-tty -m  "/dev/$block_dev" p
	printf 'Fix\n' | parted ---pretend-input-tty -m  "/dev/$block_dev" p

	parted "/dev/$block_dev" p | grep -i boot
	if [ $? -ne 0 ];
	then
	   continue
	fi

	disk_device="/dev/$block_dev"
    done

    if [[ -z $disk_device ]];
    then
	echo "Failed to get the disk device: Most likely no OS was installed"
	exit 1
    fi

    DEST_DISK=$disk_device
    echo "DEST_DISK set as $DEST_DISK"
    suffix=$(get_partition_suffix "$DEST_DISK")

    # mkdir -p /mnt
    # mount $disk_device /mnt
    # grep -iq "ubuntu" /mnt/etc/os-release
    # ubuntu_found=$?
    # umount /mnt

    # assuming that partition 1 for ubuntu is always rootfs
    # assuming that part 1 for tiberOS (mariner ) will be efi part

    echo "checking ${DEST_DISK}${suffix}1"

    file -s "${DEST_DISK}${suffix}1" | grep -q ext4
    ubuntu_found=$?
    echo "Selecting correct FDE setups $ubuntu_found"
    if [ $ubuntu_found -eq 0 ];
    then
	# fde for ubuntu
	echo "Ubuntu detected. running enable_fde."
	bash enable_fde.sh
    else
	# fde for tiber
	echo "tiber detected. running enable_fde_tiber."
	bash enable_fde_tiber.sh
    fi
}

#####################################################################################
run_enable_fde
