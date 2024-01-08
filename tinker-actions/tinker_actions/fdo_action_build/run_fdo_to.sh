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

# This script runs on to stage. only if DI was successful as per the contents of one of the below files:
# CLIENT_INTEL_DI_STATUS
# CLIENT_SDK_TPM_DI_STATUS
# CLIENT_SDK_DI_STATUS




# mount the /CRED partition as /target folder


PORT=${FDO_MPORT:-8041}
BOOTFS=/target/boot/
TLS=${FDO_TLS:-https}

echo "Doing TO"
echo "TLS : $TLS"

SERIALNO=$(cat $BOOTFS/SerialNo.txt)

cd $BOOTFS
if [ -e $BOOTFS/SerialNo.txt -a -e $BOOTFS/CLIENT_SDK_CSE_DI_STATUS -a `cat $BOOTFS/CLIENT_SDK_CSE_DI_STATUS` = "**CLIENT_SDK_CSE_DI_SUCCESSFUL**" ]; then
    echo "Starting TO for Client-SDK CSE..."

    if [[ $TLS == 'https' ]] || [[ $TLS == 'HTTPS' ]]; then
       echo "Using HTTPS for TO operation."
       cd /target/boot  && /cse-fdoout/linux-client -ss
    else
       echo "Using HTTP for TO operation."
       cd /target/boot && /cse-fdoout/linux-client
    fi

    if [ $? -ne  0 ]; then
        echo "*** Serial=${SERIALNO#"MSTRING="} CLIENT_SDK_CSE_TO2_FAILED***"
    else
        echo "*** Serial=${SERIALNO#"MSTRING="} CLIENT_SDK_CSE_TO2_SUCCESSFUL***"
    fi

    # To provide proxy information to Client-sdk, pass the command as
    # "cd /target/boot && echo -n my-proxy.com:443 > data/rv_proxy.dat && echo -n my-proxy.com:443 > data/owner_proxy.dat && /tpm-fdoout/linux-client"
elif [ -e $BOOTFS/SerialNo.txt -a -e $BOOTFS/CLIENT_SDK_TPM_DI_STATUS -a `cat $BOOTFS/CLIENT_SDK_TPM_DI_STATUS` = "**CLIENT_SDK_TPM_DI_SUCCESSFUL**" ]; then
    echo "Starting TO for Client-SDK TPM..."

    if [[ $TLS == 'https' ]] || [[ $TLS == 'HTTPS' ]]; then
       echo "Using HTTPS for TO operation."
       cd /target/boot &&  /tpm-fdoout/linux-client -ss
   else
       echo "Using HTTP for TO operation."
       cd /target/boot && /tpm-fdoout/linux-client
    fi
    if [ $? -ne  0 ]; then
        echo "*** Serial=${SERIALNO#"MSTRING="} CLIENT_SDK_TPM_TO2_FAILED***"
    else
        echo "*** Serial=${SERIALNO#"MSTRING="} CLIENT_SDK_TPM_TO2_SUCCESSFUL***"
    fi
    # To provide proxy information to Client-sdk, pass the command as
    # "cd /target/boot && echo -n my-proxy.com:443 > data/rv_proxy.dat && echo -n my-proxy.com:443 > data/owner_proxy.dat && /tpm-fdoout/linux-client"
elif [ -e $BOOTFS/SerialNo.txt -a -e $BOOTFS/CLIENT_SDK_DI_STATUS -a `cat $BOOTFS/CLIENT_SDK_DI_STATUS` = "**CLIENT_SDK_DI_SUCCESSFUL**" ]; then
    echo "Starting TO for Client-SDK..."

    if [[ $TLS == 'https' ]] || [[ $TLS == 'HTTPS' ]]; then
       echo "Using HTTPS for TO operation."
       cd /target/boot && /fdoout/linux-client -ss
    else
       echo "Using HTTP for TO operation."
       cd /target/boot && /fdoout/linux-client
    fi
    if [ $? -ne  0 ]; then
        echo "*** Serial=${SERIALNO#"MSTRING="} CLIENT_SDK_TO2_FAILED***"
    else
        echo "*** Serial=${SERIALNO#"MSTRING="} CLIENT_SDK_TO2_SUCCESSFUL***"
    fi
    # To provide proxy information to Client-sdk, pass the command as
    # "cd /target/boot && echo -n my-proxy.com:443 > data/rv_proxy.dat && echo -n my-proxy.com:443 > data/owner_proxy.dat && /fdoout/linux-client"
fi

