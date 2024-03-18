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

# This script is copied into the container docker container, and is executed within it.
# Argument 'TYPE' determines the type of the client binaries to run. Valid values are CLIENT-INTEL, CLIENT-SDK-TPM and CLIENT-SDK.
# Following is how DI is tried based on the value of 'TYPE', along with the name of status file that contains result of DI operation:
# 1. CLIENT-INTEL : Runs Client-Intel and result stored at CLIENT-INTEL_DI_STATUS. The value '**CLIENT-INTEL_DI_SUCCESSFUL**' indicates a successful DI, while
# value '**CLIENT-INTEL_DI_FAILED**' indicates DI failure.
# 2. CLIENT-SDK-TPM : Runs Client-SDK TPM and result is stored at CLIENT-SDK_TPM_DI_STATUS. The value '**CLIENT-SDK_TPM_DI_SUCCESSFUL**' indicates a successful DI, while
# value '**CLIENT-SDK_TPM_DI_FAILED**' indicates DI failure.
# 3. CLIENT-SDK : Runs Client-SDK and result is stored at CLIENT-SDK_DI_STATUS. The value '**CLIENT-SDK_DI_SUCCESSFUL**' indicates a successful DI, while
# value '**CLIENT-SDK_DI_FAILED**' indicates DI failure.
# 4. <Any_other_value> : Runs Client-Intel, then Client-SDK TPM and then Client-SDK, in the mentioned order.

#set -x

TLS=${FDO_TLS:-'https'}
IP_ADDRESS=${FDO_MFGIP:-'localhost'}
PORT=${FDO_MPORT:-8038}
MSTRING=${DEVICE_SERIAL:-'abcd12345'}
TYPE=${TYPE:-'CLIENT-SDK'}


#use discovered ip address on internal proxy route
if [ $IP_ADDRESS = "localhost" ]; then
  IP_ADDRESS=${PROXYADDR:-'localhost'}
fi

echo "TLS : $TLS"
echo "IP_ADDRESS : $IP_ADDRESS"
echo "PORT : $PORT"
echo "MSTRING : $MSTRING"
echo "TYPE : $TYPE"

#mkdir /target/
mkdir -p /target/boot/
sleep 2
# Retry 3 times by default incase the Platform does not support Client-Intel.
FDO_RETRIES=${FDO_RETRIES:-3}
if [ "$MSTRING" = "0" ]; then
  MSTRING="$(tr </dev/urandom -dc a-f0-9 | head -c10)"
fi
echo "MSTRING=$MSTRING" | tee /target/boot/SerialNo.txt

# Method to run DI using Client-SDK TPM.
runClientSdkTpmDi() {
  # Retry 3 times by default incase the Platform does not support CLIENT-INTEL.
  FDO_RETRIES=${FDO_RETRIES:-3}
  for run in $(seq 1 $FDO_RETRIES); do
    echo "Trying to DI the system: $run"
    sh /tpm-fdoout/utils/tpm_make_ready_ecdsa.sh -i -e 256 -p /tpm-fdoout/data
    echo -n ${MSTRING} >/tpm-fdoout/data/manufacturer_sn.bin
    echo -n ${MSTRING} >/tpm-fdoout/data/manufacturer_mod.bin
    echo -n ${TLS}://${IP_ADDRESS}:${PORT} >/tpm-fdoout/data/manufacturer_addr.bin
    mkdir /data
    cp -rf /tpm-fdoout/data/* /data/

    if [ ${TLS} == "https" ]; then
      # Currently '-ss' option is added to support self-signed certificates.
      # This can be removed later to ensure that only CA-signed certificates are
      # used while setting up TLS connections in Manufacturer service.
      /tpm-fdoout/linux-client -ss
    else
      /tpm-fdoout/linux-client
    fi

    if [ $? -eq 0 ]; then
      echo '**CLIENT_SDK_TPM_DI_SUCCESSFUL**' | tee /target/boot/CLIENT_SDK_TPM_DI_STATUS
      mkdir /target/boot/data/
      cp -rf /data/* /target/boot/data/
      export CLIENT_SDK_TPM_DI_STATUS=Success
      echo "*** Serial=$MSTRING CLIENT_SDK_TPM_DI_SUCCESSFUL***"
      break
    else
      echo '**CLIENT_SDK_TPM_DI_FAILED**' | tee /target/boot/CLIENT_SDK_TPM_DI_STATUS
      export CLIENT_SDK_TPM_DI_STATUS=Fail
      echo "*** Serial=$MSTRING CLIENT_SDK_TPM_DI_FAILED***"
    fi
  done
}

# Method to run DI using Client-SDK x86.
runClientSdkDi() {
  echo -n ${TLS}://${IP_ADDRESS}:${PORT} >/fdoout/data/manufacturer_addr.bin
  echo -n ${MSTRING} >/fdoout/data/manufacturer_sn.bin
  echo -n ${MSTRING} >/fdoout/data/manufacturer_mod.bin
  # Retry 3 times by default incase the Platform does not support CLIENT-INTEL.
  FDO_RETRIES=${FDO_RETRIES:-3}
  for run in $(seq 1 $FDO_RETRIES); do
    echo "Trying to DI the system: $run"
    rm -rf /data
    mkdir /data
    cp -rf /fdoout/data/* /data/

    if [ ${TLS} == "https" ]; then
      # Currently '-ss' option is added to support self-signed certificates.
      # This can be removed later to ensure that only CA-signed certificates are
      # used while setting up TLS connections in Manufacturer service.
      /fdoout/linux-client -ss
    else
      /fdoout/linux-client
    fi

    if [ $? -eq 0 ]; then
      echo '**CLIENT_SDK_DI_SUCCESSFUL**' | tee /target/boot/CLIENT_SDK_DI_STATUS
      mkdir /target/boot/data/
      cp -rf /data/* /target/boot/data/
      echo "*** Serial=$MSTRING CLIENT_SDK_DI_SUCCESSFUL***"
      break
    else
      echo '**CLIENT_SDK_DI_FAILED**' | tee /target/boot/CLIENT_SDK_DI_STATUS
      echo "*** Serial=$MSTRING CLIENT_SDK_DI_FAILED***"
    fi
  done
}

# Method to run DI using Client-SDK CSE.
runClientSdkCSEDi() {
  echo -n ${TLS}://${IP_ADDRESS}:${PORT} >/cse-fdoout/data/manufacturer_addr.bin
  echo -n ${MSTRING} >/cse-fdoout/data/manufacturer_sn.bin
  echo -n ${MSTRING} >/cse-fdoout/data/manufacturer_mod.bin
  # Retry 3 times by default incase the Platform does not support CLIENT-INTEL.
  FDO_RETRIES=${FDO_RETRIES:-3}
  for run in $(seq 1 $FDO_RETRIES); do
    echo "Trying to DI the system: $run"
    rm -rf /data
    mkdir /data
    cp -rf /cse-fdoout/data/* /data/

    if [ ${TLS} == "https" ]; then
      # Currently '-ss' option is added to support self-signed certificates.
      # This can be removed later to ensure that only CA-signed certificates are
      # used while setting up TLS connections in Manufacturer service.
      /cse-fdoout/linux-client -ss
    else
      /cse-fdoout/linux-client
    fi

    if [ $? -eq 0 ]; then
      echo '**CLIENT_SDK_CSE_DI_SUCCESSFUL**' | tee /target/boot/CLIENT_SDK_CSE_DI_STATUS
      mkdir /target/boot/data/
      cp -rf /data/* /target/boot/data/
      echo "*** Serial=$MSTRING CLIENT_SDK_CSE_DI_SUCCESSFUL***"
      break
    else
      echo '**CLIENT_SDK_CSE_DI_FAILED**' | tee /target/boot/CLIENT_SDK_CSE_DI_STATUS
      echo "*** Serial=$MSTRING CLIENT_SDK_CSE_DI_FAILED***"
    fi
  done
}

case "$TYPE" in
CLIENT-SDK-TPM)
  echo "Trying to DI the system using Client-SDK TPM"
  runClientSdkTpmDi
  ;;
CLIENT-SDK)
  echo "Trying to DI the system using Client-SDK"
  runClientSdkDi
  ;;
CLIENT-SDK-CSE)
  echo "Trying to DI the system using Client-SDK CSE"
  runClientSdkCSEDi
  ;;
*)
  echo "Trying to DI the system using CSME framework (CLIENT-INTEL)"
  runClientIntelDi
  DI_STATUS="**DI_FAILED**"
  if [ -e /target/boot/SerialNo.txt -a -e /target/boot/CLIENT_INTEL_DI_STATUS ]; then
    DI_STATUS=$(cat /target/boot/CLIENT_INTEL_DI_STATUS)
    if [ "$DI_STATUS" = "**CLIENT_INTEL_DI_FAILED**" ]; then
      echo "================================================================================================="
      echo "This System lacks DAL framework and CSE based FDO (CLIENT-SDK-CSE) init will be tried"
      echo "================================================================================================="
      runClientSdkCSEDi
      if [ -e /target/boot/SerialNo.txt -a -e /target/boot/CLIENT_SDK_CSE_DI_STATUS ]; then
        DI_STATUS=$(cat /target/boot/CLIENT_SDK_CSE_DI_STATUS)
        if [ "$DI_STATUS" = "**CLIENT_SDK_CSE_DI_FAILED**" ]; then
          echo "================================================================================================="
          echo "This System lacks DAL and CSE framework. TPM based FDO (CLIENT-SDK-TPM) init will be tried"
          echo "================================================================================================="
          runClientSdkTpmDi
          if [ -e /target/boot/SerialNo.txt -a -e /target/boot/CLIENT_SDK_TPM_DI_STATUS ]; then
            DI_STATUS=$(cat /target/boot/CLIENT_SDK_TPM_DI_STATUS)
            if [ "$DI_STATUS" = "**CLIENT_SDK_TPM_DI_FAILED**" ]; then
              echo "================================================================================================="
              echo "This System lacks DAL, CSE and TPM. Software Key based FDO (CLIENT-SDK) init will be accomplished"
              echo "================================================================================================="
              runClientSdkDi
              DI_STATUS=$(cat /target/boot/CLIENT_SDK_DI_STATUS)
            fi
          fi
        fi
      fi
    fi
  fi
  echo "$DI_STATUS"
  ;;
esac

if [ $(echo $DI_STATUS | grep -oP "DI_FAILED") ]; then
  #  echo "DI Failed"
  exit 1
fi

# mount the /CRED partition as /target folder
####  umount  /target
