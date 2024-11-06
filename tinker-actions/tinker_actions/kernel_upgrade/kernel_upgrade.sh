#!/bin/sh
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
set -x

#upgrade the kernel version to latest HWE kernel
function update_kernel_image(){
#Mount the all required partitions for kernel upgrade
rootfs_part=$1
efiboot_part=$2
mount $rootfs_part /mnt
if echo "$rootfs_part" | grep -q "rootfs_crypt"; then
    boot_part=$3
    mount $boot_part /mnt/boot
fi
mount $efiboot_part /mnt/boot/efi
mount --bind /dev /mnt/dev
mount --bind /dev/pts /mnt/dev/pts
mount --bind /proc /mnt/proc
mount --bind /sys /mnt/sys

#resolve DNS in container
rm /mnt/etc/resolv.conf
touch /mnt/etc/resolv.conf
mount --bind /etc/resolv.conf /mnt/etc/resolv.conf

mv /mnt/etc/apt/apt.conf.d/99needrestart /mnt/etc/apt/apt.conf.d/99needrestart.bkp 
#Enter into Ubuntu OS for the HWE kernel instalation
chroot /mnt /bin/bash <<EOT

apt update
#install HWE kernel with all recommended packages
apt install -y --install-recommends linux-image-generic linux-headers-generic
if [ $? -eq 0 ]; then
    echo "Successfully Installed HWE kernel"
else
    echo "Something went wrong in HWE kernel installtion please check!!!"
    exit 1
fi
update-initramfs -u -k all

#update the latest kernel version and kernel command line parameters in grub config file
sed -i 's/GRUB_DEFAULT=.*/GRUB_DEFAULT=1/g' etc/default/grub
sed -i 's/GRUB_CMDLINE_LINUX=.*/GRUB_CMDLINE_LINUX="quiet splash plymouth.enable=0 fastboot intel_iommu=on iommu=pt pci=realloc console=tty1 console=ttyS0,115200"/' etc/default/grub

update-grub
if [ $? -eq 0 ]; then
    echo "Successfuly Updated Kernel grub!!"
else
    echo "Something went wrong in updating the grub please check!!!"
    exit 1
fi
EOT

mv /mnt/etc/apt/apt.conf.d/99needrestart.bkp /mnt/etc/apt/apt.conf.d/99needrestart

#unmount the partitions
for mount in $(mount | grep '/mnt' | awk '{print $3}' | sort -nr); do
  umount "$mount"
done

}
####@main#################

#check if FDE Enabled on the disk

is_fde_set=$(blkid | grep -c "crypto_LUKS")

if [ "$is_fde_set" -ge 1 ]; then

        echo "FDE Enabled on Disk!!!"

        rootfs_part="/dev/mapper/rootfs_crypt"
        efiboot_part=$(blkid | grep -i uefi | grep -i vfat |  awk -F: '{print $1}')
        boot_part=$(blkid | grep -i boot | grep -i ext4 |  awk -F: '{print $1}')

        update_kernel_image $rootfs_part $efiboot_part $boot_part
else
        echo "FDE Disabled on Disk!!!"
	#get the rootfs partition and efibootset partition from the disk
        rootfs_part=$(blkid | grep -i rootfs | grep -i ext4 |  awk -F: '{print $1}')
        efiboot_part=$(blkid | grep -i uefi | grep -i vfat |  awk -F: '{print $1}')

        echo "Partitions detected root:$rootfs_part efi:$efiboot_part"

        # Take biggest partition as rootfs if ext4 partion not detected
        if [ -z "$rootfs_part" ]; then
                if echo "$efiboot_part" | grep -q "nvme"; then
                        prefix_disk=$(echo "$efiboot_part" | grep -oE 'nvme[0-9]+n[0-9]+' | head -n 1)
                elif echo "$efiboot_part" | grep -q "sd"; then
                        prefix_disk=$(echo "$efiboot_part" | grep -oE 'sd[a-z]+' | head -n 1)
                fi
	        rootfs_part=$(lsblk -o NAME,SIZE -nr "/dev/$prefix_disk" | grep -v "^$prefix_disk " | sort -k 2 -h | tail -n 1 | awk '{print "/dev/" $1}')
        fi

        if echo "$rootfs_part" | grep -q "nvme"; then
                os_disk=$(echo "$rootfs_part" | grep -oE 'nvme[0-9]+n[0-9]+' | head -n 1)
                part_number=$(echo "$rootfs_part" | awk 'NR==1 {print $1}' | awk -F'p' '{print $2}')
        fi
        if echo "$rootfs_part" | grep -q "sd"; then
                os_disk=$(echo "$rootfs_part" | grep -oE 'sd[a-z]+' | head -n 1)
                part_number=$(echo "$rootfs_part" | awk 'NR==1 {print $1}' | sed 's/[^0-9]*//g')
        fi

	#get the number of devices attached to system ignoring USB/Virtual/Removable disks
        blk_devices=$(lsblk -o NAME,TYPE,SIZE,RM | grep -i disk | awk '$1 ~ /sd*|nvme*/ {if ($3 !="0B" && $4 ==0)  {print $1}}')
        set -- $blk_devices
        count=$#

        #If the Number of Disks detected=1 on the system split the disk with two partitons
        #one partition for OS, Other partition for LVM
        if [ "$count" -eq 1 ]; then
                echo "Single Disk"
                NEW_PARTITION_SIZE="100%"
                RESIZE_SIZE="100GB"
                sgdisk -e "/dev/${os_disk}"
                echo yes | parted ---pretend-input-tty "/dev/${os_disk}"  resizepart "${part_number}" "$RESIZE_SIZE"
                e2fsck -f "$rootfs_part"
                resize2fs "$rootfs_part"
                partprobe "/dev/${os_disk}"
                parted "/dev/${os_disk}" --script mkpart primary ext4 $RESIZE_SIZE $NEW_PARTITION_SIZE
                partprobe "/dev/${os_disk}"
        #if more than 1 disk detected expand the rootfs partition to MAX
        else
                echo "Multiple Disks"
                sgdisk -e "/dev/${os_disk}"
                e2fsck -f -y "$rootfs_part"
                growpart "/dev/${os_disk}" "${part_number}"
		partprobe "/dev/${os_disk}"
                sgdisk -e "/dev/${os_disk}"
                resize2fs "$rootfs_part"
		partprobe "/dev/${os_disk}"
        fi
	sync
        update_kernel_image $rootfs_part $efiboot_part

fi
