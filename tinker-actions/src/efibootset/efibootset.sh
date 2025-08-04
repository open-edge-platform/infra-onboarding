#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -x

efipart=""

# Delete the pile up Ubuntu/Emt partitions from BIOS bootMenu
echo "Deleting the boot entries from boot menu"
for bootnumber in $(efibootmgr | grep -iE "Linux Boot Manager|Ubuntu" | awk '{print $1}' | sed 's/Boot//;s/\*//'); do
    efibootmgr -b "$bootnumber" -B
done

# Get the pxe boot number
pxe_boot_number=$(efibootmgr | grep -i "Bootcurrent" | awk '{print $2}')

# Delete the duplicate boot entries from bootmenu
echo "Deleting the duplicate boot entries"
boot_order=$(efibootmgr -D)
echo "$boot_order"

# Get the rootfs && OS Type
rootfs=$(blkid | grep -Ei 'TYPE="ext4"' | grep -Ei 'LABEL="cloudimg-rootfs"' | awk -F: '{print $1}')
if [ -n "$rootfs" ]; then
    os="Ubuntu"
else
    rootfs=$(blkid | grep -Ei 'TYPE="ext4"' | grep -Ei 'LABEL="rootfs"' | awk -F: '{print $1}')
    os="Emt"
fi

efiboot=$(blkid | grep -Ei 'TYPE="vfat"' | grep -Ei 'LABEL="esp|uefi"' |  awk -F: '{print $1}')

if echo "$efiboot" | grep -q "nvme"; then
    osdisk=$(echo "$rootfs" | grep -oE 'nvme[0-9]+n[0-9]+' | head -n 1)
    efipart=$(echo "$rootfs" | awk 'NR==1 {print $1}' | awk -F'p' '{print $2}')
elif echo "$efiboot" | grep -q "sd"; then
    osdisk=$(echo "$rootfs" | grep -oE 'sd[a-z]+' | head -n 1)
    efipart=$(echo "$rootfs" | awk 'NR==1 {print $1}' | sed 's/[^0-9]*//g')
fi

if [ "$os" = "Emt" ]; then
    
    # Mount all required partitions to create bootctl install entry
    mount "${rootfs}" /mnt
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

elif [ "$os" = "Ubuntu" ]; then

    # Mount all required partitions to create boot entry
    mount "${rootfs}" /mnt
    if [ ! -d /mnt/boot/efi ]; then
        mkdir -p /mnt/boot/efi
    fi
    mount "${efiboot}" /mnt/boot/efi

    # Get the grubefi 
    EFI=$(chroot /mnt sh -c 'find /boot/efi/EFI/ubuntu -type f -iname "shimx64.efi" | head -n1 | xargs -r basename') || exit 1
    if [ -z "$EFI" ]; then
        EFI="grubx64.efi"
    fi
    
    umount /mnt/boot/efi
    umount /mnt

    new_boot_number=$(efibootmgr -c -d "/dev/${osdisk}" -p $efipart -L "Ubuntu" -l "\\EFI\\ubuntu\\$EFI") || { echo "Failed to create new boot entry"; exit 1; }
    echo "Successfully created the boot entry"

    #Make UEFI boot as inactive
    efibootmgr -b $pxe_boot_number -A

    echo "Made Disk as first boot option"
fi
