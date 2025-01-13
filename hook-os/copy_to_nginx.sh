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

set -xeu -o pipefail

tinkpod=$(kubectl --namespace maestro-iaas-system get  pods -l app=tink-stack --no-headers=true -o name)

kubectl --namespace maestro-iaas-system cp alpine_image_secureboot/hook_x86_64.tar.gz "${tinkpod#*/}:/usr/share/nginx/html/"

sleep 5
kubectl exec --namespace maestro-iaas-system "${tinkpod}" -- bash -c "cd /usr/share/nginx/html; tar -xzvf hook_x86_64.tar.gz"
