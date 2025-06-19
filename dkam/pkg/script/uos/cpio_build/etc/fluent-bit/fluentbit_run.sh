#!/bin/sh

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

echo "Read Edge Node UUID from system"
EDGENODE_UUID="$(cat /sys/class/dmi/id/product_uuid)"
export EDGENODE_UUID
if [ -z "$EDGENODE_UUID" ]; then
	echo "Edge Node UUID is empty. exiting.."
	exit 1
fi
echo "EDGENODE_UUID: $EDGENODE_UUID"

echo "starting fluentbit container.."
/bin/fluent-bit -c /etc/fluent-bit/fluent-bit.yaml
