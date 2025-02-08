#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -xeu -o pipefail

tinkpod=$(kubectl --namespace maestro-iaas-system get  pods -l app=tink-stack --no-headers=true -o name)

kubectl --namespace maestro-iaas-system cp alpine_image_secureboot/hook_x86_64.tar.gz "${tinkpod#*/}:/usr/share/nginx/html/"

sleep 5
kubectl exec --namespace maestro-iaas-system "${tinkpod}" -- bash -c "cd /usr/share/nginx/html; tar -xzvf hook_x86_64.tar.gz"
