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

# Define a function to remove extra spaces from a string
removeExtraSpaces() {
    echo "$1" | tr -s ' '
}

# Define a function to filter drives based on size and type
filterDrives() {
    # Function parameters are stored in the array "$@"
    drives=("$@")
    local filteredDrives=()

    # Iterate over each drive in the array
    for drive in "${drives[@]}"; do
        # Split the drive info into name, type, and size
        IFS=' ' read -r name type size <<< "$drive"

        # Check conditions: size not zero and type is "disk"
        if [[ $size -ne 0 && $type == "disk" ]]; then
            filteredDrives+=("$drive")
        fi
    done

    # Print the filtered drives
    echo "${filteredDrives[@]}"
}

# Define a function to compare disk sizes
compare_disk_size() {
    local size1 size2
    # Split the input strings by space delimiter and extract the disk size
    size1=$(echo "$1" | awk '{print $3}')
    size2=$(echo "$2" | awk '{print $3}')
    # Compare disk sizes
    if (( size1 < size2 )); then
        echo -1
    elif (( size1 > size2 )); then
        echo 1
    else
        echo 0
    fi
}

# Define a function to compare disk types
compare_disk_type() {
    local type1 type2
    # Extract the disk type from the input strings
    type1=$(echo "$1" | awk '{print $2}')
    type2=$(echo "$2" | awk '{print $2}')
    # Compare disk types
    if [[ $type1 == *"nvme"* && $type2 != *"nvme"* ]]; then
        echo -1
    elif [[ $type1 != *"nvme"* && $type2 == *"nvme"* ]]; then
        echo 1
    else
        echo 0
    fi
}

# Define a function to get drives
getDrives() {
    # Command to list all connected storage devices with sizes using lsblk
    output=$(lsblk --output NAME,TYPE,SIZE -b 2>/dev/null)

    # Check if the command executed successfully
    if [[ $? -ne 0 ]]; then
        echo "Error running lsblk command."
        exit 1
    fi

    # Remove trailing whitespaces and split the output into lines
    outputLines=$(echo "$output" | sed 's/[[:space:]]*$//' | tail -n +2)

    # Initialize an array to store drive info
    declare -a -g drives

    # Iterate over each line in the output
    while IFS= read -r line; do
        # Remove extra spaces from the line
        line=$(removeExtraSpaces "$line")

        # Split the line into name, type, and size
        read -r name type size <<< "$line"

        # Append drive info to the drives array
        drives+=("$name $type $size")
    done <<< "$outputLines"

    # Print the list of all drives
    echo "List of all drives:"
    for element in "${drives[@]}"; do
        echo "$element"
    done

    # Filter out drives based on size and type
    filteredDrives=$(filterDrives "${drives[@]}")

    # Print the filtered drives
    echo "Filtered and Sorted Drives:"
    for drive in "${filteredDrives[@]}"; do
        echo "$drive"
    done

    # Set the variable "drives" to the filtered drives
    drives="${filteredDrives[@]}"
}

# Define a function to detect drives
driveDetection() {
    # Declare a global variable to store the detected disk
    declare -g disk

    # Get all the drives
    getDrives
    
    # Sort the array based on the following condition:
    #   1) If sizes are equal:
    #       - If either drive has the "nvme" prefix, prioritize it
    #       - If both drives have the "nvme" prefix, compare their names chronologically
    #       - If neither drive has the "nvme" prefix, compare their names chronologically
    #   2) If sizes are not equal, choose the drive with the smallest size

    sortedDrives=$(printf "%s\n" "$drives" | \
        awk '{print $3, $0}' | \
        sort -k1n,1 -k4 | \
        awk '{print $2 " " $3 " " $4}')

    # Output detected drive
    if [[ -n "$sortedDrives" ]]; then
        # Extract the name of the first drive from the sorted list
        disk="/dev/$(echo "$sortedDrives" | head -n 1 | cut -d ' ' -f 1)"
        echo "Detected drive: $disk"
    else
        echo "No valid drives found."
        exit 1
    fi
}
