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

function mismatch_msg_to_tty_device() {
    setsid -w /usr/sbin/getty -a root -L 115200 $tty vt100 &
    setsid bash -c "echo -e '\nSecure Boot Status MISMATCH\n' <> /dev/tty0 >&0 2>&1"
    setsid bash -c "echo -e '\nSecure Boot Status MISMATCH\n' <> /dev/ttyS0 >&0 2>&1"
    setsid bash -c "echo -e '\nSecure Boot Status MISMATCH\n' <> /dev/ttyS1 >&0 2>&1"
}

function match_msg_to_tty_device() {
    setsid -w /usr/sbin/getty -a root -L 115200 $tty vt100 &
    setsid bash -c "echo -e '\nSecure Boot Status MATCH\n' <> /dev/tty0 >&0 2>&1"
    setsid bash -c "echo -e '\nSecure Boot Status MATCH\n' <> /dev/ttyS0 >&0 2>&1"
    setsid bash -c "echo -e '\nSecure Boot Status MATCH\n' <> /dev/ttyS1 >&0 2>&1"
}


main() {
	result=$(./main)	
	echo " output is $result "
	if echo $result | grep -q "Mismatch"; then
		mismatch_msg_to_tty_device &
		sleep 1
		exit 1
	else
		match_msg_to_tty_device &
		sleep 1
	fi
	exit 0
}

main
