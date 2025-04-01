#!/usr/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

set -xe

# shellcheck source=/dev/null
source ./config

echo "KERNEL OCI SOURCE" "$HOOK_KERNEL_OCI_BASE"
echo "KERNEL POINT RELEASE" "$KERNEL_POINT_RELEASE"

# enable below to see debug build logs
# export DEBUG="yes"

OUT_DIR=$PWD/out/

FLUENTBIT_FILES=$PWD/fluent-bit/files
HOOKOS_FLUENTBIT_FILES=$PWD/hook/files/fluent-bit

CADDY_FILES=$PWD/caddy
HOOKOS_CADDY_FILES=$PWD/hook/files/caddy

# set this to `gsed` if on macos
SED_CMD="sed"

# CI pipeline expects the below file. But we need to make the build independent of
# CI requirements. This if-else block creates a new file TINKER_ACTIONS_VERSION from
# versions and that is pulled when hook os is getting built.

copy_fluent_bit_files() {
    mkdir -p "$HOOKOS_FLUENTBIT_FILES"
    if ! cp -r "$FLUENTBIT_FILES"/* "$HOOKOS_FLUENTBIT_FILES"; then
        echo "Copy of the fluent-bit config file to the hook/files folder failed"
        exit 1
    fi
}

get_caddy_conf() {
    mkdir -p "$HOOKOS_CADDY_FILES"
    if ! cp -r "$CADDY_FILES"/* "$HOOKOS_CADDY_FILES"; then
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
        update_env_variables
        copy_fluent_bit_files
        get_caddy_conf

        ./build.sh build hook-default-amd64
    popd || exit 1 # out of hook dir

    mkdir -p "$OUT_DIR"

    if ! cp "$PWD"/hook/out/hook_x86_64.tar.gz "$OUT_DIR"; then
        echo "Build of HookOS failed!"
        exit 1
    fi

    echo "Build of HookOS succeeded!"
}

build_hook
