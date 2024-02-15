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

# Function to get the hexadecimal value of a given EFI boot entry name
get_hex(){
  # Retrieve the EFI boot entry information containing the specified name
  is_avail=$(efibootmgr | grep "$1" | awk '{print $1}')
  # Extract the hexadecimal value of the boot entry
  hex_val=$(echo "$is_avail" | sed s/Boot// | sed s/*//)
  echo "$hex_val"
}

# Function to delete an EFI boot entry by its hexadecimal value
delete_efi_entry(){
  efibootmgr -B -b $1
}

# Function to retrieve the current boot order from the EFI firmware
get_boot_order(){
  boot_order=$(efibootmgr | grep BootOrder | awk '{print $2}')
  echo "$boot_order"
}

# Function to setup partitions based on the drive type
setup_partitions() {
  DRIVE=$1
  if [[ $DRIVE == *nvme* ]]; then
    BOOT_PARTITION=${DRIVE}p1
    SWAP_PARTITION=${DRIVE}p2
    ROOT_PARTITION=${DRIVE}p3
  elif [[ $DRIVE == *sd[a-z]* ]]; then
    BOOT_PARTITION=${DRIVE}1
    SWAP_PARTITION=${DRIVE}2
    ROOT_PARTITION=${DRIVE}3
  elif [[ $DRIVE == *mmcblk* ]]; then
    BOOT_PARTITION=${DRIVE}p1
    SWAP_PARTITION=${DRIVE}p2
    ROOT_PARTITION=${DRIVE}p3
  else
    echo "No supported drives found!"
    exit 1
  fi
}

# Function to mount the boot partition
mount_boot_partition() {
  if [ ! -b $1 ]; then
    echo "Drive $DRIVE does not have the $1"
    partprobe
    if [ ! -b "$1" ]; then
      echo "Failed to find the $1 even after executing partprobe command"
      exit 1
    else
      echo "Successfully found the $1 using lsblk"
    fi
  else
    echo "Drive $DRIVE already has all the partitions"
  fi
  mkdir -p $2
  mount $1 $2
}

# Function to delete an old EFI boot entry
delete_old_efi_entry() {
  old_efi_boot_name=$1
  hex=$(get_hex $old_efi_boot_name)
  if [ ! -z $hex ]; then
    delete_efi_entry $hex
  fi
}

# Function to search for GRUB bootloader
search_grub_bootloader() {
  BOOTFS=$1
  possible_grub_paths=("\\EFI\\BOOT\\grubx64.efi" "\\EFI\\BOOT\\bootx64.efi" "\\EFI\\ubuntu\\grubx64.efi" "\\EFI\\centos\\grubx64.efi" "\\EFI\\redhat\\grubx64.efi")
  for grub_path in "${possible_grub_paths[@]}"; do
    file=${grub_path//\\/\/}
    if test -f "$BOOTFS$file"; then
      echo "Found GRUB bootloader at: $BOOTFS$file"
      GRUB_FILE_PATH=$file
      return 0
    fi
  done
  echo "None of the GRUB bootloader files were found."
  exit 1
}

# Function to create a new EFI boot entry
create_new_efi_entry() {
  new_efi_boot_name=$1
  efibootmgr -c -d $2 -p 1 -L $new_efi_boot_name -l $3
}

# Main configure_boot_order function
configure_boot_order() {
  DRIVE=$1
  BOOTFS=/target/boot
  NEW_EFI_BOOT_NAME="ubuntu_${DRIVE##*/}"
  GRUB_FILE_PATH=""
  echo "$(printf '#%.0s' $(seq 1 10)) setup_partitions $(printf '#%.0s' $(seq 1 10))"
  setup_partitions $DRIVE
  echo "$(printf '#%.0s' $(seq 1 10)) mount_boot_partition $(printf '#%.0s' $(seq 1 10))"
  mount_boot_partition $BOOT_PARTITION $BOOTFS
  echo "$(printf '#%.0s' $(seq 1 10)) delete_old_efi_entry $(printf '#%.0s' $(seq 1 10))"
  delete_old_efi_entry $NEW_EFI_BOOT_NAME
  echo "$(printf '#%.0s' $(seq 1 10)) search_grub_bootloader $(printf '#%.0s' $(seq 1 10))"
  search_grub_bootloader $BOOTFS
  echo "$(printf '#%.0s' $(seq 1 10)) create_new_efi_entry $(printf '#%.0s' $(seq 1 10))"
  create_new_efi_entry $NEW_EFI_BOOT_NAME $BOOT_PARTITION $GRUB_FILE_PATH
}
