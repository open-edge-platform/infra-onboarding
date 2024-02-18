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

####################
# efi_part="/dev/nvme0n1p1"
# hook_part="/dev/nvme0n1p2"
# swap_part="/dev/nvme0n1p3"
# creds_part="/dev/nvme0n1p4"
# remaining_part="/dev/nvme0n1p5"
efi_part="1"
hook_part="2"
swap_part="3"
creds_part="4"
remaining_part="5"

########################
# drive detection
source drive_detection.sh
driveDetection
if [ -z "$disk" ]; then
    exit
fi
BLOCK_DEVICE=$disk
TINKERBELL_OWNER=${TINKERBELL_OWNER:-localhost}
#BLOCK_DEVICE="/dev/nvme0n1"


####################
#tinkerbell owner information will be replaced during docker image build
mac_address_current_device=$(cat /proc/cmdline | grep -o "instance_id=..:..:..:..:..:.. " | awk ' {split($0,a,"="); print a[2]} ')

#TODO add ips from workflow env variables
####################
#fixed variables
hook_mnt="/var/hook"
efi_mnt="/var/efi"

####################
#Create Bootx64
MODULES="configfile ext2 fat part_gpt normal
linux ls boot echo reboot search search_fs_file search_fs_uuid
search_label help font efi_gop efi_uga gfxterm linuxefi regexp probe progress"

#Check if Tink options are not set
if [ -z "$TINKERBELL_OWNER" ]; then
	echo "$TINKERBELL_OWNER is not set"
	exit 1
fi
if [ -z "$mac_address_current_device" ]; then
	echo "MAC id of device is not set"
	exit 1
fi

###################
EXTRA_TINK_OPTIONS="tinkerbell=http://$TINKERBELL_OWNER syslog_host=$TINKERBELL_OWNER packet_action=workflow packet_state= osie_vendors_url= http_proxy=$http_proxy https_proxy=$https_proxy no_proxy=$no_proxy HTTP_PROXY=$HTTP_PROXY HTTPS_PROXY=$HTTPS_PROXY NO_PROXY=$NO_PROXY console=ttyS0,11520 tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0 grpc_authority=$TINKERBELL_OWNER:42113 packet_base_url=http://$TINKERBELL_OWNER:8080/workflow tinkerbell_tls=false instance_id=$mac_address_current_device worker_id=$mac_address_current_device packet_bootdev_mac=$mac_address_current_device facility=sandbox"




####################

fix_partition_suffix() {
    part_variable=''
    ret=$(grep -i "nvme" <<< "$BLOCK_DEVICE")
    if [ $? == 0 ]
    then
	part_variable="p"
    fi

    echo $part_variable
}

check_return_value() {
    if [ $1 -ne 0 ]
    then
	echo "$2"
	exit
    fi
}

echo "Selected Block Disk $BLOCK_DEVICE"
suffix=$(fix_partition_suffix)


#make grub partition
parted --script $BLOCK_DEVICE \
       mklabel gpt \
       mkpart ESP fat32 1MiB 1024MiB \
       set 1 esp on \
       mkpart primary ext4 1024MiB 2048MiB \
       mkpart primary linux-swap 2048MiB 3072MiB \
       mkpart primary ext4 3072MiB 4096MiB \
       mkpart primary 4096MiB 100%

check_return_value $? "Failed to create paritions"

sleep 5
partprobe 
#########
#hook_os partition
mkfs -t ext4 -L hook -F "${BLOCK_DEVICE}${suffix}${hook_part}"
check_return_value $? "Failed to mkfs hook"
echo "hook_os partition done"

e2label "${BLOCK_DEVICE}${suffix}${hook_part}" hook
check_return_value $? "Failed to create hook lable"
echo "hook_os partition label done"

mkdir -p $hook_mnt
mount "${BLOCK_DEVICE}${suffix}${hook_part}" ${hook_mnt}
check_return_value $? "Failed to mount hook"
echo "hook_os partition mount done"

#########
sleep 5
partprobe
echo "mkfs -t vfat -n BOOT ${BLOCK_DEVICE}${suffix}${efi_part}   <-"
mkfs -t vfat -n BOOT "${BLOCK_DEVICE}${suffix}${efi_part}"
check_return_value $? "Failed to mkfs boot"
echo "Boot partition done"

mkdir -p $efi_mnt
mount "${BLOCK_DEVICE}${suffix}${efi_part}" ${efi_mnt}
check_return_value $? "Failed to mount efi part"
echo "efi partition mount done"
#########

mkswap "${BLOCK_DEVICE}${suffix}${swap_part}"
check_return_value $? "Failed to make swap space"
echo "swap part done"

#########

#the creds for FDO
mkfs -t ext4 -L CREDS -F "${BLOCK_DEVICE}${suffix}${creds_part}"
check_return_value $? "Failed to mkfs creds label"
echo "creds partition done"

#########
#the rest of the memory 
mkfs -t ext4 -F "${BLOCK_DEVICE}${suffix}${remaining_part}"
check_return_value $? "Failed to mkfs remaining partitions"
echo "free partition done"


# mkdir -p $remaining_mnt
# mount "${BLOCK_DEVICE}${suffix}${remaining_part}" ${remaining_mnt}
# check_return_value $? "Failed to mount partition 5"
echo "free partition mount done"

##########################################
# Making hook bootable
mkdir -p ${efi_mnt}/EFI/hook

#grubenv add
grub-editenv ${efi_mnt}/EFI/hook/mac_address set net_default_mac_user=$mac_address_current_device


#copy vmlinuz of hook
cd ${efi_mnt}/EFI/hook/
# wget https://github.com/tinkerbell/hook/releases/download/v0.8.0/hook_x86_64.tar.gz
cp /hook_x86_64.tar.gz ${efi_mnt}/EFI/hook/
tar -xvf hook_x86_64.tar.gz --no-same-owner
rm -rf hook_x86_64.tar.gz
echo "completed download and install of vmlinuz and initramfs-x86_64"

#######################################################
#move the nvme or the sda/sdb to the top of the boot orders
disk_bootnum=$(efibootmgr -v | grep -i "nvme")
if [ $? -ne 0 ];
then
    disk_bootnum=$(efibootmgr -v | grep -i "sata")
fi
disk_bootnum=$(awk '{ print substr($1, 5, 4)}' <<< $disk_bootnum)

echo "disk_bootnum $disk_bootnum"
boot_order=$(efibootmgr | grep -i "Bootorder" | awk '{print $2}')
remove_disk=$(sed "s/$disk_bootnum//g" <<< $boot_order)

final_bootorder=$disk_bootnum","$remove_disk
final_bootorder=$(sed "s/,,/,/g" <<< $final_bootorder)
final_bootorder=$(sed "s/,$//g" <<< $final_bootorder)
echo "bootorder -> $final_bootorder"

efibootmgr --bootorder $final_bootorder
echo "Made nvme/sata disk the first in the boot order"
########################################################


efibootmgr -c --remove-dups -d ${BLOCK_DEVICE} -p 1 -L "hookOS" -l '\EFI\hook\BOOTX64.efi'
echo "Configure EFI boot manager done"

######## make PXE the last boot option possible.
pxe_boot_number=$(efibootmgr | grep -i "Bootcurrent" | awk '{print $2}')

boot_order=$(efibootmgr | grep -i "Bootorder" | awk '{print $2}')

remove_pxe=$(sed "s/$pxe_boot_number//g" <<< $boot_order)
remove_pxe=$(sed "s/,,/,/g" <<< $remove_pxe)

final_bootorder=$remove_pxe","$pxe_boot_number
final_bootorder=$(sed "s/,,/,/g" <<< $final_bootorder)
efibootmgr --bootorder $final_bootorder
echo "Made PXE the last in the boot order"
