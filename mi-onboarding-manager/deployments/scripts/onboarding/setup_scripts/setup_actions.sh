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
source ../config

source ./secure_hookos.sh

ip_regex="^([0-9]{1,3}\.){3}[0-9]{1,3}$"

if ! echo "$pd_host_ip" | grep -E -q "$ip_regex"; then

    if [ "$#" -ge 1 ]; then
        if echo "$1" | grep -E -q "$ip_regex"; then
            pd_host_ip=$1
        else
            echo "Populate Config file with pd_host_ip either in config or as argument"
            exit 1
        fi
    else
        echo "Populate Config file with pd_host_ip properly "
        exit 1
    fi
fi

if ! echo "$load_balancer_ip" | grep -E -q "$ip_regex"; then
    if [ "$#" -eq 2 ]; then
        if echo "$2" | grep -E -q "$ip_regex"; then
            load_balancer_ip=$2
        else
            echo "Populate Config file with load_balancer_ip either in config or as argument"
            exit 1
        fi
    else
        echo "Populate Config file with load_balancer_ip properly "
        exit 1
    fi
fi

export pd_host_ip
export load_balancer_ip

#relative paths of folders
store_alpine=$PWD/../tinker_actions/store_alpine/
fdo_action_build=$PWD/../tinker_actions/fdo_action_build/
create_partition=$PWD/../tinker_actions/create_partition
efibootset=$PWD/../tinker_actions/efibootset
creds_copy=$PWD/../tinker_actions/creds_copy

daemon_json="""
{\n\"insecure-registries\" : [\"$pd_host_ip:5015\"]\n}
"""

allow_insecure_reg() {
    echo -e $daemon_json | sudo tee /etc/docker/daemon.json 2>&1 >/dev/null
    sudo systemctl daemon-reload
    sudo systemctl restart docker
    sleep 2
}

setup_docker_registry() {

    allow_insecure_reg
    # echo $PWD

    local out=$(docker container stop registry 2>&1)
    out=$(docker container prune -f)

    out=$(rm -rf docker_registry 2>&1)
    mkdir docker_registry
    pushd docker_registry

    openssl req -newkey rsa:4096 \
        -nodes -sha256 \
        -subj "/C=In/L=Bangalore/O=Intel/OU=Department/CN=example.com" \
        -keyout domain.key \
        -x509 -days 365 -out domain.crt

    popd

    docker run -d \
        --restart=always \
        --name registry \
        -v "$(pwd)"/docker_registry:/certs \
        -e REGISTRY_HTTP_ADDR=0.0.0.0:5015 \
        -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/domain.crt \
        -e REGISTRY_HTTP_TLS_KEY=/certs/domain.key \
        -p 5015:5015 \
        registry:2
}

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
    sed -i "s|dl-cdn.alpinelinux.org/alpine/edge/testing|dl-cdn.alpinelinux.org/alpine/edge/community|g" hook-docker/Dockerfile

    docker run --rm -it -e HTTP_PROXY=$http_proxy -e HTTPS_PROXY=$https_proxy -e NO_PROXY=$no_proxy -e http_proxy=$http_proxy -e https_proxy=${https_proxy} -v "$PWD:$PWD" -w "$PWD" -v /var/run/docker.sock:/var/run/docker.sock nixos/nix nix-shell --run "make dist"
    #    make dist

    popd
    sudo cp $PWD/hook/out/sha-/rel/hook_x86_64.tar.gz $store_alpine
    make_local_hook_over_pxe
}
build_credscopy() {
    pushd $creds_copy

    bash build.sh

    popd
}
verify_registry() {
    local docker_images=$(curl -X GET https://localhost:5015/v2/_catalog --insecure 2>&1)
    # ["create_partition","efibootset","fdoclient_action","store_alpine"]}
    out=$(grep -i "fdoclient_action" <<<$docker_images)
    if [ $? -ne 0 ]; then
        echo "fdoclient_action was missing in the registry. Check again"
        exit 1
    fi

    out=$(grep -i "create_partition" <<<$docker_images)
    if [ $? -ne 0 ]; then
        echo "create_partition was missing in the registry. Check again"
        exit 1
    fi

    out=$(grep -i "efibootset" <<<$docker_images)
    if [ $? -ne 0 ]; then
        echo "efibootset was missing in the registry. Check again"
        exit 1
    fi

    out=$(grep -i "store_alpine" <<<$docker_images)
    if [ $? -ne 0 ]; then
        echo "store_alpine was missing in the registry. Check again"
        exit 1
    fi

    out=$(grep -i "cred_copy" <<<$docker_images)
    if [ $? -ne 0 ]; then
        echo "cred_copy was missing in the registry. Check again"
        exit 1
    fi

    echo "All containers are present in the registry"
}

make_local_hook_over_pxe() {
    #copy to the downloaded location of nginx
    #    sudo cp $PWD/hook/out/sha-/rel/vmlinuz-x86_64 /opt/hook
    #    sudo cp $PWD/hook/out/sha-/rel/initramfs-x86_64 /opt/hook
    wget https://github.com/tinkerbell/hook/releases/download/v0.8.1/hook_x86_64.tar.gz
    cp /hook_x86_64.tar.gz /opt/hook
    tar -xvf hook_x86_64.tar.gz --no-same-owner
    rm -rf hook_x86_64.tar.gz
    sudo cp vmlinuz-x86_64 /opt/hook
    sudo cp initramfs-x86_64 /opt/hook

    rm -rf initramfs-x86_64  vmlinuz-x86_64
}

main() {

    sudo apt install -y build-essential

    setup_docker_registry
    fdo_docker_setup
    efibootset_setup

    build_hook

    secure_hookos

    store_alpine_setup
    create_partition_setup
    build_credscopy

    verify_registry

}

main
