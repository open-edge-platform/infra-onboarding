#!/bin/bash -e

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#set -x

#relative paths of folders
read_sb_status=./tinker_actions/read_sb_status/
efibootset=./tinker_actions/efibootset
fde_setup=./tinker_actions/fde
creds_copy=./tinker_actions/creds_copy
caddy_proxy=./tinker_actions/caddy
fluentbit=./tinker_actions/fluentbit
image2disk=./tinker_actions/image2disk
cexec=./tinker_actions/cexec
writefile=./tinker_actions/writefile/v1
erase_non_removable_disks=./tinker_actions/erase_non_removable_disks
hook_dind=./tinker_actions/hook_dind
device_discovery=./tinker_actions/device_discovery
kernel_upgrade=./tinker_actions/kernel_upgrade
tiberos_partition=./tinker_actions/tiberos_partition

read_sb_status_setup() {
    pushd $read_sb_status

    bash -e build.sh

    popd
}

efibootset_setup() {

    pushd $efibootset

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

erase_non_removable_disks_setup() {

    pushd $erase_non_removable_disks

    bash build.sh

    popd
}

hook_dind_setup() {

    pushd $hook_dind

    bash build.sh

    popd
}

device_discovery_setup() {

    pushd $device_discovery

    bash -e build.sh

    popd
}

kernel_upgrade_setup() {

    pushd $kernel_upgrade

    bash -e build.sh

    popd
}

tiberos_partition_setup() {

    pushd $tiberos_partition

    bash -e build.sh

    popd
}

main() {

    apt install -y build-essential

    read_sb_status_setup
    efibootset_setup

    fde_setup_action
    build_credscopy

    caddyproxy_setup
    fluentbit_setup
    image2disk_setup
    cexec_setup
    writefile_setup
    erase_non_removable_disks_setup
    hook_dind_setup
    device_discovery_setup
    kernel_upgrade_setup
    tiberos_partition_setup
}

main
