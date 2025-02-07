#!/bin/sh

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

echo "Read Edge Node UUID from system"
UUID="$(cat /sys/class/dmi/id/product_uuid)"
if [ -z "$UUID" ]; then
    echo "Edge Node UUID is empty. exiting.."
    exit 1
fi

echo "UUID: $UUID"
export EDGENODE_UUID=$UUID

echo "starting fluentbit container.."
/fluent-bit/bin/fluent-bit -c /fluent-bit/etc/fluent-bit.yaml
