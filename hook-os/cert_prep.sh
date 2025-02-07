#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -xu -o pipefail

source ./config

echo "Getting intel and cluster certs. Ensure KUBECONFIG file for the Orchestrator cluster is copied to $HOME/.kube/config path"
rm -f client_auth/files/ca.pem

# Copy maestro-ca.crt from kubernetes secret
kubectl get secret gateway-ca-cert -n maestro-iaas-system -o jsonpath='{.data.*}' | base64 -d > client_auth/files/ca.pem

# Add new line to ca.pem so that tinkerbell certificate can be inserted in new line
echo "" >> client_auth/files/ca.pem
wget "https://${deployment_dns_extension}/boots/ca.crt" --no-check-certificate -O boots_ca.crt
cat boots_ca.crt >> client_auth/files/ca.pem

rm boots_ca.crt

# Add new line to ca.pem so that intel ca certificates can be inserted in new line
echo "" >> client_auth/files/ca.pem

mkdir certs_tmp
curl -o certs.zip http://certificates.intel.com/repository/certificates/IntelSHA2RootChain-Base64.zip
unzip certs.zip -d certs_tmp
for file in certs_tmp/*; do
    if [ -f "$file" ]; then
	cat "$file" >> client_auth/files/ca.pem
    fi
done

rm certs.zip
rm -rf certs_tmp

# add new line to ca.pem so that public lets-encrypt certificates can be inserted in new line(CSA)
echo "" >> client_auth/files/ca.pem

# get letsencrypt certs
for certfile in isrgrootx1.pem lets-encrypt-r3.pem lets-encrypt-e1.pem trustid-x3-root.pem.txt
do
  curl https://letsencrypt.org/certs/$certfile >> client_auth/files/ca.pem
done

