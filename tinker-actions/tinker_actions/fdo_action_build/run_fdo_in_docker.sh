#!/bin/bash

#########################################################################################
# INTEL CONFIDENTIAL
# Copyright (2023) Intel Corporation
#
# The source code contained or described herein and all documents related to the source
# code("Material") are owned by Intel Corporation or its suppliers or licensors. Title
# to the Material remains with Intel Corporation or its suppliers and licensors. The
# Material contains trade secrets and proprietary and confidential information of Intel
# or its suppliers and licensors. The Material is protected by worldwide copyright and
# trade secret laws and treaty provisions. No part of the Material may be used, copied,
# reproduced, modified, published, uploaded, posted, transmitted, distributed, or
# disclosed in any way without Intel's prior express written permission.
#
# No license under any patent, copyright, trade secret or other intellectual property
# right is granted to or conferred upon you by disclosure or delivery of the Materials,
# either expressly, by implication, inducement, estoppel or otherwise. Any license under
# such intellectual property rights must be express and approved by Intel in writing.
#########################################################################################

# This script loads the docker image 'fdoclient115.tar' and creates a container, only if DI was successful as per the contents of one of the below files:
# CLIENT_INTEL_DI_STATUS
# CLIENT_SDK_TPM_DI_STATUS
# CLIENT_SDK_DI_STATUS
MAX_RETRIES_CADDY=3
retry_count=0

PROXYADDR=$(ip route get 1 | head -n 1 | grep -o 'src\s[.0-9a-z]\+' | awk '{print $2}')
# check if docker network
if echo "$default_ip" | grep -q '^172'; then
  PROXYADDR=$(ip route | grep default | grep -oE "\\b([0-9]{1,3}\\.){3}[0-9]{1,3}\\b")
fi
#add  host to the system
echo "$PROXYADDR localhost.internal1 localhost.internal2" | tee -a /etc/hosts

# Check if NGINX proxy service is up
until [ $(curl -w "%{http_code}" --output /dev/null -s -k https://localhost.internal1:8081/health) = 200 ]; do
  ((retry_count++))
  if [ $retry_count -lt $MAX_RETRIES_CADDY ]; then
    echo "Internal Proxy server still not up, wait for 10 sec"
    sleep 10
  else
    echo "Maximum retries reached. Proxy Not up. Exiting..."
    exit 1
  fi

done

echo "Internal Proxy server is up, resuming FDO operations.."

# mount the /CRED partition as /target folder
## sudo mount  -L ${DATA_PARTITION_LBL} /target
FDO_RUN_TYPE=${FDO_RUN_TYPE:-'to'}

PARTITION_LBL=${DATA_PARTITION_LBL:-'CREDS'}
# mount the /CRED partition as /target folder
mkdir -p /target
ret=$(mount -L ${PARTITION_LBL} /target)

if [ "$?" -ne 0 ]; then
  echo "No partion Found with CREDS Label"
  exit 1
fi

ret=0
if [[ $FDO_RUN_TYPE == 'di' ]]; then
  bash /usr/bin/run_fdo_di.sh
  ##TODO check retturn values and debug
  if [ $? -ne 0 ]; then
    echo "DI Failed"
    ret=1
    #      exit 1
  fi
else
  bash /usr/bin/run_fdo_to.sh

  if [ -e "/dev/shm" ] && [ -r "/dev/shm" ]; then
    cp -rf /target/boot /dev/shm/
    echo "fdo data copied at /dev/shm/boot."
  else
    echo "/dev/shm is not available or not readable."
  fi
fi

umount /target

exit $ret
