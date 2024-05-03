#!/usr/bin/env bash
#####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2024 Intel Corporation                                              #
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################

echo "Read Edge Node UUID from system"
UUID="$(cat /sys/class/dmi/id/product_uuid)"
if [[ -z "$UUID" ]]; then
    echo "Edge Node UUID is empty. exiting.."
    exit 1
fi

echo "UUID: $UUID"
export EDGENODE_UUID=$UUID

echo "starting fluentbit container.."
/opt/fluent-bit/bin/fluent-bit -c /fluent-bit/etc/fluent-bit.conf
