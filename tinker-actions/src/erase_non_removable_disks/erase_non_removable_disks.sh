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

BLOCK_SIZE=1M
# Format drives
# 1. Size should not be 0
# 2. Type should be disk and not partition or rom
# 3. Should be Non-removable

lsblk_output=$(lsblk --output NAME,SIZE,TYPE,RM -bldn)
echo "$lsblk_output" | awk '{if ($2 != 0 && $3 == "disk" && $4 != 1) print $1}' | while read -r TARGET_DISK; do

    # Check if the target disk has a force_ro file and set its value to 0 if it exists
    FORCE_RO_FILE="/sys/block/$(basename "$TARGET_DISK")/force_ro"
    if [ -f "$FORCE_RO_FILE" ]; then
        echo 0 | tee "$FORCE_RO_FILE"
    fi

    # Get the total size of the target disk in bytes
    DISK_SIZE=$(blockdev --getsize64 "$TARGET_DISK")

    # Convert block size to bytes manually (e.g., 1M = 1048576 bytes)
    case "$BLOCK_SIZE" in
        *K) BLOCK_SIZE_BYTES=$((${BLOCK_SIZE%K} * 1024));;
        *M) BLOCK_SIZE_BYTES=$((${BLOCK_SIZE%M} * 1024 * 1024));;
        *G) BLOCK_SIZE_BYTES=$((${BLOCK_SIZE%G} * 1024 * 1024 * 1024));;
        *) echo "Unsupported block size format: $BLOCK_SIZE"; exit 1;;
    esac

    # Calculate the count (number of blocks to write)
    COUNT=$(($DISK_SIZE / $BLOCK_SIZE_BYTES))

    # Run the dd command with the calculated count
    echo "Writing to $TARGET_DISK with bs=$BLOCK_SIZE and count=$COUNT"
    dd if=/dev/zero of="$TARGET_DISK" bs="$BLOCK_SIZE" count="$COUNT"
done
partprobe
