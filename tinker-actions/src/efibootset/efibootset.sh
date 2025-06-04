#!/bin/sh

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

boot_entry=""
EFI=""

# Delete the pile up Ubuntu/Emt partitions from BIOS bootMenu

for bootnumber in $(efibootmgr | grep -iE "emt|Ubuntu" | awk '{print $1}' | sed 's/Boot//;s/\*//'); do
  efibootmgr -b $bootnumber -B
done

# Check the provision flow for Ubuntu/EMT
# Get the rootfs and bios disk numbers 

is_fde_set=$(blkid | grep -c "crypto_LUKS" || true)

if [ "$is_fde_set" -ge 1 ]; then

    echo "FDE Enabled on Disk!!!"
    rootfs="/dev/mapper/rootfs_crypt"
else
    rootfs=$(blkid | grep -Ei 'TYPE="ext4"' | grep -Ei 'LABEL="rootfs"' | awk -F: '{print $1}')
fi

efiboot_part=$(blkid | grep -Ei 'TYPE="vfat"' | grep -Ei 'LABEL="esp|uefi"' | sed -E 's/.*[a-z]+[0-9]*p?([0-9]+):.*/\1/')
efiboot=$(blkid | grep -Ei 'TYPE="vfat"' | grep -Ei 'LABEL="esp|uefi"' |  awk -F: '{print $1}')

if echo "$efiboot" | grep -q "nvme"; then
    os_disk=$(echo "$rootfs_part" | grep -oE 'nvme[0-9]+n[0-9]+' | head -n 1)
elif echo "$efiboot" | grep -q "sd"; then
    os_disk=$(echo "$rootfs_part" | grep -oE 'sd[a-z]+' | head -n 1)
fi


# For Ubuntu efiboot partnumber 5
# For EMT efiboot partnumber 1

if [ "$efiboot_part" -eq 5 ]; then
    boot_entry="Ubuntu"
elif [ "$efiboot_part" -eq 1 ]; then
    boot_entry="Emt"
else
    echo  "Invalid OS type detected,Please check!!"
    echo "$efiboot_part"
    exit 1
fi

mount "${rootfs}" /mnt
mount $efiboot /mnt/boot/efi

if [ "$boot_entry" -eq "Ubuntu" ]; then
    EFI=$(chroot /mnt sh -c 'basename $(find /boot/efi/EFI/ubuntu -type f -iname "*.efi" | head -n1)') || exit 1
elif [ "$boot_entry" -eq "Emt" ]; then
    EFI=$(chroot /mnt sh -c 'basename $(find /boot/efi/EFI/Linux -type f -iname "*.efi" | head -n1)') || exit 1
fi

# Unmount the file systems
umount /mnt/boot/efi
umount /mnt

# Create the boot entry

new_boot_number=$(efibootmgr -c -d "/dev/${os_disk}" -p $efiboot_part -L "$boot_entry" -l "\\EFI\\Linux\\$EFI") || exit 1

echo "Successfully created the boot entry $efiboot_part"

# Update new boot number as first boot option.
pxe_boot_number=$(efibootmgr | grep -i "Bootcurrent" | awk '{print $2}')

boot_order_cur=$(efibootmgr | grep -i "Bootorder" | awk '{print $2}')

if [ -n "$boot_order_cur" ]; then
    final_boot_order="$new_boot_number,$boot_order_cur" || exit 1
else
    final_boot_order="$new_boot_number" || exit 1
fi

# Update the boot order using efibootmgr
efibootmgr -o "$final_boot_order" || exit 1

#Make UEFI boot as inactive 
efibootmgr -b $pxe_boot_number -A

echo "Made Disk as first boot option"
