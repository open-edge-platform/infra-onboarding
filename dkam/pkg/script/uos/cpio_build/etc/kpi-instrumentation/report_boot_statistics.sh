#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Usage: report_boot_statistics.sh [input_file]
# Default input file is /proc/cmdline

input_file="${1:-/proc/cmdline}"

bootKitStartTimestamp=$(date +%s)
echo "Reporting boot KPI s_bootkit_start $bootKitStartTimestamp"

parse_hex_to_uint32() {
    local hexstr="$1"
    hexstr="${hexstr//$'\n'/}"
    if [[ "$hexstr" == 0x* ]]; then
        hexstr="${hexstr:2}"
    fi
    # Convert hex to uint32
    printf "%d" "0x$hexstr"
}

if [[ ! -f "$input_file" ]]; then
    echo "Input file $input_file does not exist."
    exit 1
fi

for cmdline in $(tr ' ' '\n' < "$input_file"); do
    IFS='=' read -r bootTracepoint bootTracepointValue <<< "$cmdline"
    if [[ -z "$bootTracepoint" ]]; then
        continue
    fi
    if [[ "$bootTracepoint" == s_* ]]; then
        value=$(parse_hex_to_uint32 "$bootTracepointValue" 2>/dev/null)
        if [[ $? -ne 0 ]]; then
            echo "Failed to print boot statistics $bootTracepoint"
        else
            echo "Reporting boot KPI $bootTracepoint $value"
        fi
    fi
done
