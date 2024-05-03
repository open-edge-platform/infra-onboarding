#!/bin/bash -e

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
read_sb_status=./tinker_actions/read_sb_status/
store_alpine=./tinker_actions/store_alpine/
fdo_action_build=./tinker_actions/fdo_action_build/
create_partition=./tinker_actions/create_partition
efibootset=./tinker_actions/efibootset
fde_setup=./tinker_actions/fde
creds_copy=./tinker_actions/creds_copy
client_auth=./tinker_actions/client_auth
nginx_proxy=./tinker_actions/nginx_proxy
caddy_proxy=./tinker_actions/caddy
fluentbit=./tinker_actions/fluentbit
image2disk=./tinker_actions/image2disk/v1
cexec=./tinker_actions/cexec/v1
writefile=./tinker_actions/writefile/v1

read_sb_status_setup() {
    pushd $read_sb_status

    bash -e build.sh

    popd
}

fdo_docker_setup() {
    pushd $fdo_action_build

    bash -e build_fdo_client_action.sh

    popd
}

store_alpine_setup() {

    pushd $store_alpine

    bash -e build.sh

    popd
}

efibootset_setup() {

    pushd $efibootset

    bash -e build.sh

    popd
}

create_partition_setup() {

    pushd $create_partition

    bash -e build.sh

    popd
}

fde_setup_action() {

    pushd $fde_setup

    bash -e build.sh

    popd
}

build_credscopy() {
    pushd $creds_copy

    bash -e build.sh

    popd
}

client_auth_setup() {

    pushd $client_auth

    bash -e build.sh

    popd
}

nginxproxy_setup() {

    pushd $nginx_proxy

    bash -e build.sh

    popd
}

caddyproxy_setup() {

    pushd $caddy_proxy

    bash -e build.sh

    popd
}

fluentbit_setup() {

    pushd $fluentbit

    bash -e build.sh

    popd
}

image2disk_setup() {

    pushd $image2disk

    bash build.sh

    popd
}

cexec_setup() {

    pushd $cexec

    bash build.sh

    popd    
}

writefile_setup() {

    pushd $writefile

    bash build.sh

    popd
}

main() {

    apt install -y build-essential

    read_sb_status_setup
    fdo_docker_setup
    efibootset_setup

    store_alpine_setup
    create_partition_setup
    fde_setup_action
    build_credscopy

    client_auth_setup
    nginxproxy_setup
    caddyproxy_setup
    fluentbit_setup
    image2disk_setup
    cexec_setup
    writefile_setup
}

main
