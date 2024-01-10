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
source ./config

source ./secure_hookos.sh

STORE_ALPINE_SECUREBOOT=$PWD/alpine_image_secureboot/
STORE_ALPINE=$PWD/alpine_image/

build_hook() {

    local out=$(rm -rf hook)
    patch_file=$PWD/patch.p
    new_patch_file=$PWD/hook/new_patch.p

    git clone https://github.com/tinkerbell/hook hook
    pushd hook

    git checkout v0.8.1

    cp $patch_file $new_patch_file
    sed -i "s+FIX_H_TTP_PROXY+$http_proxy+g" $new_patch_file
    sed -i "s+FIX_H_TTPS_PROXY+$https_proxy+g" $new_patch_file

    git apply $new_patch_file
    
    #i255 igc driver issue fix
    pushd kernel/
    mkdir patches-5.10.x
    pushd patches-5.10.x/
    #download the igc i255 driver patch file
    wget https://github.com/intel/linux-intel-lts/commit/170110adbecc1c603baa57246c15d38ef1faa0fa.patch
    popd
    make devbuild_5.10.x
    popd

    #update the hook.yaml file to point to new kernel
    sed -i "s|quay.io/tinkerbell/hook-kernel:5.10.85-d1225df88208e5a732e820a182b75fb35c737bdd|quay.io/tinkerbell/hook-kernel:5.10.85-e546ea099917c006d1d08fe6b8398101de65cbc7|g" hook.yaml    

    sed -i "s|dl-cdn.alpinelinux.org/alpine/edge/testing|dl-cdn.alpinelinux.org/alpine/edge/community|g" hook-docker/Dockerfile

    docker run --rm -it -e HTTP_PROXY=$http_proxy -e HTTPS_PROXY=$https_proxy -e NO_PROXY=$no_proxy -e http_proxy=$http_proxy -e https_proxy=${https_proxy} -v "$PWD:$PWD" -w "$PWD" -v /var/run/docker.sock:/var/run/docker.sock nixos/nix nix-shell --run "make dist"
    #    make dist

    popd
    mkdir -p $STORE_ALPINE
    mkdir -p $STORE_ALPINE_SECUREBOOT
    sudo cp $PWD/hook/out/sha-/rel/hook_x86_64.tar.gz $STORE_ALPINE
    sudo cp $PWD/hook/out/sha-/rel/hook_x86_64.tar.gz $STORE_ALPINE_SECUREBOOT
    #copy to the downloaded location of nginx
    if [ -d /opt/hook ]; then
        sudo cp $PWD/hook/out/sha-/rel/hook_x86_64.tar.gz /opt/hook/
        pushd /opt/hook/
        sudo tar -xzvf hook_x86_64.tar.gz >/dev/null 2&>1
        sudo rm hook_x86_64.tar.gz
        popd
    fi

}


main() {

    sudo apt install -y build-essential

    build_hook

    secure_hookos
}

main
