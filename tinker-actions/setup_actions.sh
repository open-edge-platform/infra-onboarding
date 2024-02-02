#!/bin/bash

#####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2023 Intel Corporation                                              #
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################
#set -x

#relative paths of folders
store_alpine=./tinker_actions/store_alpine/
fdo_action_build=./tinker_actions/fdo_action_build/
create_partition=./tinker_actions/create_partition
efibootset=./tinker_actions/efibootset
fde_setup=./tinker_actions/fde
creds_copy=./tinker_actions/creds_copy

fdo_docker_setup() {
    pushd $fdo_action_build

    bash build_fdo_client_action.sh

    popd
}

store_alpine_setup() {

    pushd $store_alpine

    bash build.sh

    popd
}

efibootset_setup() {

    pushd $efibootset

    bash build.sh

    popd
}

create_partition_setup() {

    pushd $create_partition

    bash build.sh

    popd
}

fde_setup_action() {

    pushd $fde_setup

    bash build.sh

    popd
}

build_credscopy() {
    pushd $creds_copy

    bash build.sh

    popd
}

main() {

    sudo apt install -y build-essential

    fdo_docker_setup
    efibootset_setup

    store_alpine_setup
    create_partition_setup
    fde_setup_action
    build_credscopy

}

main
