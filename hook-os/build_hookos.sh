#!/usr/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -x

# shellcheck source=/dev/null
source ./config

export HOOK_KERNEL=${HOOK_KERNEL:-5.10}

if [ "$HOOK_KERNEL" == "5.10" ]; then
    #Current validated kernel_point_version is 228
    export KERNEL_POINT_RELEASE_CONFIG=228
fi

OUT_DIR=$PWD/out/

FLUENTBIT_FILES=$PWD/fluent-bit/files
HOOKOS_FLUENTBIT_FILES=$PWD/hook/files/fluent-bit

CADDY_FILES=$PWD/caddy/files
HOOKOS_CADDY_FILES=$PWD/hook/files/caddy

# set this to `gsed` if on macos
SED_CMD="sed"

# CI pipeline expects the below file. But we need to make the build independent of
# CI requirements. This if-else block creates a new file TINKER_ACTIONS_VERSION from
# versions and that is pulled when hook os is getting built.

copy_fluent_bit_files() {
    mkdir -p "$HOOKOS_FLUENTBIT_FILES"
    if ! cp "$FLUENTBIT_FILES"/* "$HOOKOS_FLUENTBIT_FILES"; then
        echo "Copy of the fluent-bit config file to the hook/files folder failed"
        exit 1
    fi
}

get_caddy_conf() {
    mkdir -p "$HOOKOS_CADDY_FILES"
    if ! cp "$CADDY_FILES"/* "$HOOKOS_CADDY_FILES"; then
        echo "Copy of the Caddyfile to the hook/files folder failed"
        exit 1
    fi
}

# shellcheck disable=SC2154
update_env_variables() {
    # Update runtime configs in hook.template.yaml
    $SED_CMD -i "s|update_idp_url|$keycloak_url|g" linuxkit-templates/hook.template.yaml
    $SED_CMD -i "s|update_tink_stack_svc|$tink_stack_svc|g" linuxkit-templates/hook.template.yaml
    $SED_CMD -i "s|update_tink_server_svc|$tink_server_svc|g" linuxkit-templates/hook.template.yaml
    $SED_CMD -i "s|update_onboarding_manager_svc|$onboarding_manager_svc|g" linuxkit-templates/hook.template.yaml
    $SED_CMD -i "s|update_onboarding_stream_svc|$onboarding_stream_svc|g" linuxkit-templates/hook.template.yaml
    $SED_CMD -i "s|update_release_svc|$release_svc|g" linuxkit-templates/hook.template.yaml
    $SED_CMD -i "s|update_oci_release_svc|$oci_release_svc|g" linuxkit-templates/hook.template.yaml
    $SED_CMD -i "s|update_logging_svc|$logging_svc|g" linuxkit-templates/hook.template.yaml

    #update extra hosts needed?
    if [ -n "$extra_hosts" ]; then
        # needed for keycloak.kind.internal type of deployment
        $SED_CMD -i "s|update_extra_hosts|$extra_hosts|g" linuxkit-templates/hook.template.yaml
    else
        #Remove the entire line for extra hosts if config doesnt have any value
        $SED_CMD -i "s|- EXTRA_HOSTS=update_extra_hosts||g" linuxkit-templates/hook.template.yaml
    fi
}
build_hook() {

    git reset HEAD -- hook

    cp -rf hook.yaml hook/linuxkit-templates/hook.template.yaml
    pushd hook || exit 1

    echo "starting to build kernel...................................................."

    if [ "$HOOK_KERNEL" == "6.6" ]; then
        if docker image inspect quay.io/tinkerbell/hook-kernel:6.6.52-2f1e89d8 >/dev/null 2>&1; then
            echo "Rebuild of kernel not required, since its already present in docker images"
        else
            pushd kernel/ || exit 1
            echo "Going to remove patches dir if any"
            rm -rf patches-6.6.y
            mkdir patches-6.6.y
                pushd patches-6.6.y || exit 1
            #download any patches
                popd || exit 1
            popd || exit 1

            #hook-default-amd64
            ./build.sh kernel hook-latest-lts-amd64
        fi
    else
        if docker image inspect quay.io/tinkerbell/hook-kernel:5.10.228-e0637f99 >/dev/null 2>&1; then
            echo "Rebuild of kernel not required, since its already present in docker images"
        else
            # i255 igc driver issue fix
            pushd kernel/ || exit 1
            echo "Going to remove patches DIR if any"
            rm -rf patches-5.10.y
            mkdir patches-5.10.y
                pushd patches-5.10.y || exit 1
                #download the igc i255 driver patch file
                wget https://github.com/intel/linux-intel-lts/commit/170110adbecc1c603baa57246c15d38ef1faa0fa.patch
                echo "Downloading kernel patches done"
                popd || exit 1
            popd || exit 1

            #    ./build.sh kernel default
            ./build.sh kernel
        fi
    fi

    update_env_variables

    # copy fluent-bit and caddy related files
    copy_fluent_bit_files
    get_caddy_conf

    if [ "$HOOK_KERNEL" == "6.6" ]; then
        ./build.sh build hook-latest-lts-amd64
    else
        ./build.sh
    fi

    popd || exit 1 # out of hook dir

    mkdir -p "$OUT_DIR"

    if [ "$HOOK_KERNEL" == "6.6" ]; then
        mv "$PWD"/hook/out/hook_latest-lts-x86_64.tar.gz "$PWD"/hook/out/hook_x86_64.tar.gz
    fi

    if ! cp "$PWD"/hook/out/hook_x86_64.tar.gz "$OUT_DIR"; then
        echo "Build of HookOS failed!"
        exit 1
    fi

    echo "Build of HookOS succeeded!"
}

main() {

    sudo apt install -y build-essential bison flex

    build_hook
}

main
