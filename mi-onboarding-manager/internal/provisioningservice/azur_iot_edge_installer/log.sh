#!/bin/bash
# INTEL CONFIDENTIAL
# Copyright (C) 2023 Intel Corporation
#
# This software and the related documents are Intel copyrighted materials, and
# your use of them is governed by the express license under which they were
# provided to you ("License"). Unless the License provides otherwise, you may
# not use, modify, copy, publish, distribute, disclose or transmit this
# software or the related documents without Intel's prior written permission.
#
# This software and the related documents are provided as is, with no express
# or implied warranties, other than those that are expressly stated in the
# License.

##
## Logging utilities for bash
##

NO_COLOR=0
RED='\033[0;31m'
YELLOW="\033[1;33m"
GREEN="\033[0;32m"
NC='\033[0m' # No Color

function log_warn() {
    if [ "$NO_COLOR" -eq "1" ] ; then
        echo -e "[$(date)] - WARN: $1"
    else
        echo -e "${YELLOW}[$(date)] WARN: $1${NC}"
    fi
}

function log_info() {
    if [ "$NO_COLOR" -eq "1" ] ; then
        echo -e "[$(date)] INFO: $1"
    else
        echo -e "${GREEN}[$(date)] INFO: $1${NC}"
    fi
}

function log_error() {
    if [ "$NO_COLOR" -eq "1" ] ; then
        echo -e "[$(date)] ERROR: $1"
    else
        echo -e "${RED}[$(date)] ERROR: $1${NC}"
    fi
}

function log_fatal() {
    if [ "$NO_COLOR" -eq "1" ] ; then
        echo -e "[$(date)] FATAL: $1"
    else
        echo -e "${RED}[$(date)] FATAL: $1${NC}"
    fi

    export EXITCODE=-1
    exit ${EXITCODE}
}

function check_error() {
    if [ $? -ne 0 ] ; then
        log_fatal "$1"
    fi
}

function check_error_cb() {
    if [ $? -ne 0 ] ; then
        $1
        log_fatal $2
    fi
}