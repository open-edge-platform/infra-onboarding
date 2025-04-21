#!/bin/sh

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

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
    display_msg_to_tty_devices "Secure Boot Status SKIPPED" 1
    exit 0
}
main
