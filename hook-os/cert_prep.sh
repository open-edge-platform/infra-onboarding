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

set -xu -o pipefail

source ./config

echo "getting intel and cluster certs"
rm -f client_auth/files/ca.pem

wget_no_proxy="--no-proxy"
if [ "$external_proxy" != '' ];
then
  wget_no_proxy=
fi

attempt=0
until [ $attempt -ge 5 ]
do
  wget --max-redirect=0 "https://vault.${deployment_dns_extension}/v1/pki_root/ca" $wget_no_proxy --no-check-certificate && break
  attempt=$((attempt+1))
  sleep 5
done

set -e

if [ $attempt -eq 5 ]; then
  echo "Failed to download maestro root cert"
  exit 1
fi

openssl x509 -in ca -inform der -outform pem > client_auth/files/ca.pem

wget "https://${deployment_dns_extension}/boots/ca.crt" $wget_no_proxy --no-check-certificate -O boots_ca.crt
cat boots_ca.crt >> client_auth/files/ca.pem

rm ca boots_ca.crt

# Add new line to ca.pem so that intel ca certificates can be inserted in new line
echo "" >> client_auth/files/ca.pem

for certfile in intel_5A.crt intel_5A_2.crt intel_5B.crt intel_5B_2.crt intel_root.crt
do
  curl https://ubit-artifactory-or.intel.com/artifactory/it-btrm-local/intel_cacerts/$certfile >> client_auth/files/ca.pem
done

# add new line to ca.pem so that public lets-encrypt certificates can be inserted in new line(CSA)
echo "" >> client_auth/files/ca.pem

# get letsencrypt certs
for certfile in isrgrootx1.pem lets-encrypt-r3.pem lets-encrypt-e1.pem trustid-x3-root.pem.txt
do
  curl https://letsencrypt.org/certs/$certfile >> client_auth/files/ca.pem
done

