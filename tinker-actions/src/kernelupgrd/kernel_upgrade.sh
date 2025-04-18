#!/bin/sh

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -x

# Sync file system
function sync_file_system(){
rootfs_part=$1
# Check if the partition available 
count=0
while [ ! -b "$rootfs_part" ]; do
    sleep 1
    count=$((count+1))
    if [ "$count" -ge 15 ]; then
         echo "Partition table not synced,exiting the installation"
	 exit 1
    fi
done
}
#upgrade the kernel version to latest HWE kernel
function update_kernel_image(){
#Mount the all required partitions for kernel upgrade
rootfs_part=$1
efiboot_part=$2
# Wait until the partition is available
sync_file_system "$rootfs_part"
if [ "$?" -ne 0 ]; then
    echo "file sync for $rootfs_part failed, please check!!"
    exit 1
fi

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
mount --bind /run /mnt/run

#resolve DNS in container
rm /mnt/etc/resolv.conf
touch /mnt/etc/resolv.conf
mount --bind /etc/resolv.conf /mnt/etc/resolv.conf

mv /mnt/etc/apt/apt.conf.d/99needrestart /mnt/etc/apt/apt.conf.d/99needrestart.bkp 

#Get the Latest canonical 6.8 kerner version 
export kernel_version=$(chroot /mnt /bin/bash -c "apt-cache search linux-image | grep 'linux-image-6.8.*-generic' | tail -1 | awk '{print \$1}' | grep -oP '(?<=linux-image-)[0-9]+\.[0-9]+\.[0-9]+-[0-9]+'")

if [ -z "kernel_version" ]; then
    echo "Unable to get the kernel version,please check !!!!"
    exit 1
fi

#Enter into Ubuntu OS for the latest 6.x kernel instalation
chroot /mnt /bin/bash <<EOT

apt update

#install 6.x kernel with all recommended packages and kernel modules
apt install -y  linux-image-\${kernel_version}-generic linux-headers-\${kernel_version}-generic
apt install -y --install-recommends linux-modules-extra-\${kernel_version}-generic

if [ "$?" -eq 0 ]; then
    echo "Successfully Installed 6.x kernel"
else
    echo "Something went wrong in 6.x kernel installtion please check!!!"
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
                elif echo "$efiboot_part" | grep -q "mmcblk"; then
                        prefix_disk=$(echo "$efiboot_part" | grep -oE 'mmcblk[0-9]+' | head -n 1)
                elif echo "$efiboot_part" | grep -q "sd"; then
                        prefix_disk=$(echo "$efiboot_part" | grep -oE 'sd[a-z]+' | head -n 1)
                fi
	        rootfs_part=$(lsblk -o NAME,SIZE -nr "/dev/$prefix_disk" | grep -v "^$prefix_disk " | sort -k 2 -h | tail -n 1 | awk '{print "/dev/" $1}')
        fi

        if echo "$rootfs_part" | grep -q "nvme"; then
                os_disk=$(echo "$rootfs_part" | grep -oE 'nvme[0-9]+n[0-9]+' | head -n 1)
                part_number=$(echo "$rootfs_part" | awk 'NR==1 {print $1}' | awk -F'p' '{print $2}')
        fi
        if echo "$rootfs_part" | grep -q "mmcblk"; then
                os_disk=$(echo "$rootfs_part" | grep -oE 'mmcblk[0-9]+' | head -n 1)
                part_number=$(echo "$rootfs_part" | awk 'NR==1 {print $1}' | awk -F'p' '{print $2}')
        fi
        if echo "$rootfs_part" | grep -q "sd"; then
                os_disk=$(echo "$rootfs_part" | grep -oE 'sd[a-z]+' | head -n 1)
                part_number=$(echo "$rootfs_part" | awk 'NR==1 {print $1}' | sed 's/[^0-9]*//g')
        fi

	#get the number of devices attached to system ignoring USB/Virtual/Removable disks
        blk_devices=$(lsblk -o NAME,TYPE,SIZE,RM | grep -i disk | awk '$1 ~ /sd*|nvme*|mmcblk*/ && $1 !~ /boot/ {if ($3 != "0B" && $4 == 0) {print $1}}')
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
                partprobe "/dev/${os_disk}" 
		# Wait until the partition is available
                sync_file_system "$rootfs_part"
                e2fsck -f "$rootfs_part"
		# Before resize the partition 
		sync_file_system "$rootfs_part"
                resize2fs "$rootfs_part"
		if [ "$?" -ne 0 ]; then
		    echo "Resize of the $rootfs_part failed, please check!!"
		    exit 1
		fi
                partprobe "/dev/${os_disk}"
                parted "/dev/${os_disk}" --script mkpart primary ext4 $RESIZE_SIZE $NEW_PARTITION_SIZE
                partprobe "/dev/${os_disk}"
        #if more than 1 disk detected expand the rootfs partition to MAX
        else
                echo "Multiple Disks"
                sgdisk -e "/dev/${os_disk}"
                partprobe "/dev/${os_disk}"
                e2fsck -f -y "$rootfs_part"
                growpart "/dev/${os_disk}" "${part_number}"
		partprobe "/dev/${os_disk}"
                # Wait until the partition is available
                sync_file_system "$rootfs_part"
                sgdisk -e "/dev/${os_disk}"
		sync_file_system "$rootfs_part"
                resize2fs "$rootfs_part"
		if [ "$?" -ne 0 ]; then
                    echo "Resize of the $rootfs_part failed, please check!!"
                    exit 1
                fi
		partprobe "/dev/${os_disk}"
        fi
	sync
        update_kernel_image $rootfs_part $efiboot_part

fi
