#!/bin/bash
# INTEL CONFIDENTIAL
# Copyright (C) 2023 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# This software and the related documents are Intel copyrighted materials, and
# your use of them is governed by the express license under which they were
# provided to you ("License"). Unless the License provides otherwise, you may
# not use, modify, copy, publish, distribute, disclose or transmit this
# software or the related documents without Intel's prior written permission.
#
# This software and the related documents are provided as is, with no express
# or implied warranties, other than those that are expressly stated in the
# License.
# shellcheck disable=all
ABS_SCRIPT_PATH="$(dirname "$(realpath "${BASH_SOURCE:-$0}")")"
#shellcheck source=logs.sh
source "${ABS_SCRIPT_PATH}"/log.sh

#Install Base Pakges Required to start the Iot_Edge_Installer
sudo apt install make -y

function usage() {
    echo "usage: $1 [-h] -e <env-file>"
}

env_file=""

while getopts ":h:e:" options ; do
    case "${options}" in
    e)
        env_file="${OPTARG}"
        ;;
    :)
        if [ "${OPTARG}" == "h" ] ; then
            usage "$0"
        else
            log_fatal "-${OPTARG} requires an argument value or is unknown"
        fi
        ;;
    *)
        log_fatal "Unknown argument: -${options} ${OPTARG}"
        ;;
    esac

done

if [ ! -f "${env_file}" ] ; then
    log_fatal "Environmental variable file does not exist"
fi

log_info "Sourcing ${env_file}"
# shellcheck source=/dev/null
source "${env_file}"

# Verify configuration values
if [ -z "${ID_SCOPE}" ] ; then
    log_fatal "ID_SCOPE is not defined"
elif [ "${ID_SCOPE}" == "" ] ; then
    log_fatal "ID_SCOPE is blank"
elif [ -z "${REGISTRATION_ID}" ] ; then
    log_fatal "REGISTRATION_ID is not defined"
elif [ "${REGISTRATION_ID}" == "" ] ; then
    log_fatal "REGISTRATION_ID is blank"
elif [ -z "${SYMMETRIC_KEY}" ] ; then
    log_fatal "SYMMETRIC_KEY is not defined"
elif [ "${SYMMETRIC_KEY}" == "" ] ; then
    log_fatal "SYMMETRIC_KEY is blank"
fi

log_info "Configuring apt repository to install edge-config-tool"
prod_list=$(curl -f -sSL https://packages.microsoft.com/config/ubuntu/20.04/prod.list)
check_error "Failed to retrieve prod.list"

echo "${prod_list}" | tee /etc/apt/sources.list.d/microsoft-prod.list
check_error "Failed to write microsoft-prod.list file"

msft_asc=$(curl -f -sSL https://packages.microsoft.com/keys/microsoft.asc)
check_error "Failed to retrieve microsoft.asc"

echo "${msft_asc}" | tee /etc/apt/trusted.gpg.d/microsoft.asc
check_error "Failed to write microsoft.asc"

log_info "Updating apt cache"
apt update
check_error "Failed to update apt cache"

log_info "Installing edge-config-tool"
apt install -y edge-config-tool
check_error "Failed to install edge-config-tool"

cd /usr/local/microsoft/edge-config-tool/ || exit
check_error "Failed to go to /usr/local/microsoft/edge-config-tool/ directory"

log_info "Installing Azure IoT Edge"
./azure-iot-edge-installer.sh \
    -s "${ID_SCOPE}" \
    -r "${REGISTRATION_ID}" \
    -k "${SYMMETRIC_KEY}"
check_error "Failed to install Azure IoT Edge"
touch "$(pwd)"/.azure_dps_setp_done
