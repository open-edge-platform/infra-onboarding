#!/bin/sh -e

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -eu

# Source the eject script (ensure this file exists and is correct)
source eject_all_removable_disks.sh

# Eject all removable devices
if ! eject_all_removable_devices; then
    echo "Error: Failed to eject all removable devices."
    exit 1
fi

# Format drives
# 1. Size should not be 0
# 2. Type should be disk and not partition or rom
# 3. Should be Non-removable

lsblk_output=$(lsblk --output NAME,SIZE,TYPE,RM -bldn)
echo "$lsblk_output" | awk '{if ($2 != 0 && $3 == "disk" && $4 != 1) print $1}' | while read -r disk; do
    dd if=/dev/zero of="/dev/$disk" bs=100MB count=20
done
partprobe
