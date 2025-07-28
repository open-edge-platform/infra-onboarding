#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Delete the pile up Ubuntu/Emt partitions from BIOS bootMenu
for bootnumber in $(efibootmgr | grep -iE "Linux Boot Manager|Ubuntu" | awk '{print $1}' | sed 's/Boot//;s/\*//'); do
    efibootmgr -b "$bootnumber" -B
done

# Delete the duplicate boot entries from bootmenu
boot_order=$(efibootmgr -D)
echo "$boot_order"

# Get the rootfs
rootfs=$(blkid | grep -Ei 'TYPE="ext4"' | grep -Ei 'LABEL="rootfs"' | awk -F: '{print $1}')

efiboot=$(blkid | grep -Ei 'TYPE="vfat"' | grep -Ei 'LABEL="esp|uefi"' |  awk -F: '{print $1}')

if echo "$efiboot" | grep -q "nvme"; then
    osdisk=$(echo "$rootfs" | grep -oE 'nvme[0-9]+n[0-9]+' | head -n 1)
elif echo "$efiboot" | grep -q "sd"; then
     osdisk=$(echo "$rootfs" | grep -oE 'sd[a-z]+' | head -n 1)
fi
    
# Mount all required partitions to create bootctl install entry
# Check for secure boot endabled , if enabled change the rootfs path

rootfs_secure=$(blkid | grep "/dev/mapper/" | awk -F: '{print $1}')
if [ -n "$rootfs_secure" ]; then
    mount "${rootfs_secure}" /mnt
else
    mount "${rootfs}" /mnt
fi
mount $efiboot /mnt/boot/efi
mount --bind /dev /mnt/dev
mount --bind /dev/pts /mnt/dev/pts
mount --bind /proc /mnt/proc
mount --bind /sys /mnt/sys
mount --bind /sys/firmware/efi/efivars /mnt/sys/firmware/efi/efivars

chroot /mnt /bin/bash <<EOT
    set -e
    bootctl install
EOT

if [ "$?" -eq 0 ]; then
    echo "Made Disk as first boot option"
    #unmount the partitions
    for mount in $(mount | grep '/mnt' | awk '{print $3}' | sort -nr); do
        umount "$mount"
    done
else
    echo "Boot entry create failed,Please check!!"
    #unmount the partitions
    for mount in $(mount | grep '/mnt' | awk '{print $3}' | sort -nr); do
        umount "$mount"
    done
    exit 1 
fi

