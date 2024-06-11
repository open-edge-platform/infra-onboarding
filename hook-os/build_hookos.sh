#!/usr/bin/env bash

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

BASE_DIR=$PWD
STORE_ALPINE_SECUREBOOT=$PWD/alpine_image_secureboot/
STORE_ALPINE=$PWD/alpine_image/
CLIENT_AUTH_LOCATION=$PWD/client_auth/container
CLIENT_AUTH_SCRIPTS=$PWD/client_auth/scripts
CLIENT_AUTH_FILES=$PWD/client_auth/files
HOOKOS_IDP_FILES=$PWD/hook/files/idp/

FLUENTBIT_FILES=$PWD/fluent-bit/files
HOOKOS_FLUENTBIT_FILES=$PWD/hook/files/fluent-bit

CADDY_FILES=$PWD/caddy/files
HOOKOS_CADDY_FILES=$PWD/hook/files/caddy

# set this to `gsed` if on macos
SED_CMD=sed

# CI pipeline expects the below file. But we need to make the build independent of
# CI requirements. This if-else block creates a new file TINKER_ACTIONS_VERSION from
# versions and that is pulled when hook os is getting built.

VERSION_FILE=$PWD/tinker-actions/VERSION

if [ ! -f $VERSION_FILE ]; then

    if [ ! -f $PWD/TINKER_ACTIONS_VERSION ]; then
        cp $PWD/VERSION $PWD/TINKER_ACTIONS_VERSION
    fi

    VERSION_FILE=$PWD/TINKER_ACTIONS_VERSION
fi

copy_fluent_bit_files() {

    mkdir -p $HOOKOS_FLUENTBIT_FILES

    cp $FLUENTBIT_FILES/* $HOOKOS_FLUENTBIT_FILES

    if [ $? -ne 0 ]; then
        echo "Copy of the fluent-bit config file to the hook/files folder failed"
        exit 1
    fi
}

get_client_auth() {

    pushd $CLIENT_AUTH_SCRIPTS
    bash get_certs.sh
    popd

    mkdir -p $HOOKOS_IDP_FILES

    # if predefined files are needed place them in client_auth/files as ca.pem and server_cert.pem
    cp $CLIENT_AUTH_FILES/* $HOOKOS_IDP_FILES
    if [ $? -ne 0 ]; then
        echo "Copy of the certificates to the hook/files folder failed"
        exit 1
    fi
}

get_caddy_conf() {
    mkdir -p $HOOKOS_CADDY_FILES
    cp $CADDY_FILES/* $HOOKOS_CADDY_FILES
    if [ $? -ne 0 ]; then
        echo "Copy of the Caddyfile to the hook/files folder failed"
        exit 1
    fi

    # Update NGINX runtime configs in hook.yaml
    $SED_CMD -i "s|update_tink_stack_svc|$tink_stack_svc|g" hook.yaml
    $SED_CMD -i "s|update_tink_server_svc|$tink_server_svc|g" hook.yaml
    $SED_CMD -i "s|update_release_svc|$release_svc|g" hook.yaml
    $SED_CMD -i "s|update_oci_release_svc|$oci_release_svc|g" hook.yaml
    $SED_CMD -i "s|update_manufacturer_svc|$fdo_manufacturer_svc|g" hook.yaml
    $SED_CMD -i "s|update_owner_svc|$fdo_owner_svc|g" hook.yaml
    $SED_CMD -i "s|update_logging_svc|$logging_svc|g" hook.yaml
}

build_hook() {

    git reset HEAD -- hook

    pushd hook

    ver=$(cat $VERSION_FILE)
    # Iterate over the array and print each element
    arrayof_images=($(cat hook.yaml | grep -i ".*image:.*:.*$" | awk -F: '{print $2}'))
    for image in "${arrayof_images[@]}"; do
        temp=$(grep -i "/" <<< $image)
        if [ $? -eq 0 ];
        then
            # Non harbor Image
            continue
        fi
        $SED_CMD -i "s/$image:latest/$image:$ver/g" hook.yaml
    done

    # copy fluent-bit related files
    copy_fluent_bit_files

    # if kernel already built or pulled into docker images list then dont recompile
    if docker image inspect quay.io/tinkerbell/hook-kernel:5.10.85-ea30730ea52b3f903fad7ff11a82dd12dfbdbe6c-xz >/dev/null 2>&1; then
        echo "Rebuild of kernel not required, since its already present in docker images"
    else
        # i255 igc driver issue fix
        pushd kernel/
        mkdir patches-5.10.x
        pushd patches-5.10.x/
        # download the igc i255 driver patch file
        wget https://github.com/intel/linux-intel-lts/commit/170110adbecc1c603baa57246c15d38ef1faa0fa.patch
        popd

        make -j8 devbuild_5.10.x
        popd
    fi

    #update the hook.yaml file to point to new kernel
    $SED_CMD -i "s|quay.io/tinkerbell/hook-kernel:5.10.85-d1225df88208e5a732e820a182b75fb35c737bdd|quay.io/tinkerbell/hook-kernel:5.10.85-ea30730ea52b3f903fad7ff11a82dd12dfbdbe6c-xz|g" hook.yaml

    $SED_CMD -i "s|dl-cdn.alpinelinux.org/alpine/edge/testing|dl-cdn.alpinelinux.org/alpine/edge/community|g" hook-docker/Dockerfile
    $SED_CMD -i "s/hook_dind:latest/hook_dind:$ver/g" hook-docker/Dockerfile

    #update keycloak url
    $SED_CMD -i "s|update_idp_url|$keycloak_url|g" hook.yaml

    #update extra hosts needed?
    if [ -n "$extra_hosts" ]; then
        # needed for keycloak.kind.internal type of deployment
        $SED_CMD -i "s|update_extra_hosts|$extra_hosts|g" hook.yaml
    else
        #Remove the entire line for extra hosts if config doesnt have any value
        $SED_CMD -i "s|- EXTRA_HOSTS=update_extra_hosts||g" hook.yaml
    fi

    # get the client_auth files and container before running the hook os build.
    get_client_auth
    get_caddy_conf

    docker buildx build --load -t hook-nix-dev -f hack/Dockerfile .
    docker run --rm -e HTTP_PROXY=$http_proxy \
        -e HTTPS_PROXY=$https_proxy \
        -e NO_PROXY=$no_proxy \
        -e http_proxy=$http_proxy \
        -e https_proxy=${https_proxy} \
        -v "$PWD:$PWD" \
        -w "$PWD" \
        -v /var/run/docker.sock:/var/run/docker.sock hook-nix-dev nix-shell \
        --run "make dist"
    #    make dist

    popd # out of hook dir

    mkdir -p $STORE_ALPINE
    mkdir -p $STORE_ALPINE_SECUREBOOT

    cp $PWD/hook/out/sha-*/rel/hook_x86_64.tar.gz $STORE_ALPINE

    if [ $? -ne 0 ]; then
        echo "Build of HookOS failed!"
        exit 1
    fi

    cp $PWD/hook/out/sha-*/rel/hook_x86_64.tar.gz $STORE_ALPINE_SECUREBOOT

    #copy to the downloaded location of nginx
    if [ -d /opt/hook ]; then
        sudo cp $PWD/hook/out/sha-/rel/hook_x86_64.tar.gz /opt/hook/

        pushd /opt/hook/
        sudo tar -xzvf hook_x86_64.tar.gz 2 >/dev/null &>1
        sudo rm hook_x86_64.tar.gz
        popd
    fi

    echo "Build of HookOS succeeded!"
}

main() {

    sudo apt install -y build-essential bison flex

    build_hook

    secure_hookos
}

main
