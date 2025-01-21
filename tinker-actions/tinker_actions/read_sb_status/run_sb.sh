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

function display_msg_to_tty_devices() {
    local msg="$1"
    local color="$2"
#    tty="ttyS0"
#    setsid  -w /sbin/getty -L 115200 "$tty" vt100 &
    # If color is 1 (red), use the escape sequence for red. If color is 2 (green), use the escape sequence for green.
    local color_code=$([ "$color" -eq 1 ] && echo -e '\033[31m' || echo -e '\033[32m')
    echo -e "\n $color_code $msg \033[0m \n" > /dev/tty0
    echo -e "\n $color_code $msg \033[0m \n" > /dev/ttyS0
    echo -e "\n $color_code $msg \033[0m \n" > /dev/ttyS1

}

main() {
    cat /proc/kmsg > /host/sblog.txt &
    result=$(./main)
    echo " output is $result "
    case "$result" in
        "") display_msg_to_tty_devices "Unable to read secure boot status" 1 &
	sleep 1
        exit 1
        ;;
        *Mismatch*) display_msg_to_tty_devices "Secure Boot Status MISMATCH" 1 &
	sleep 1
        exit 1
        ;;
        *) display_msg_to_tty_devices "Secure Boot Status MATCH" 2 ;;
    esac
    sleep 1
    exit 0
}
main
