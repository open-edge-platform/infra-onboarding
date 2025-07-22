#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#set -x

set -xueo pipefail
data_dir=$1
uos_file_name="emb_uos_x86_64.tar.gz"

# shellcheck disable=SC1091
source secure_uos.sh

pushd ../
# shellcheck disable=SC1091
source ./config
popd || exit
CPIO_OUTPUT=output
mkdir -p "$CPIO_OUTPUT"
cp "$data_dir"/$uos_file_name "$CPIO_OUTPUT"

LOCATION_OF_EXTRA_FILES=$PWD/etc
LOCATION_OF_ENV_CONFIG=$PWD/etc/emf/env_config
LOCATION_OF_UOS_ENV=$PWD/etc/emf/
IDP=$LOCATION_OF_EXTRA_FILES/idp
EXTRACTED_FILES_LOCATION=$CPIO_OUTPUT/emt_uos_x86_64_files

#######################################################################################################
create_env_config() {

    #Just to double confirm that the folder is available.
    mkdir -p "$LOCATION_OF_UOS_ENV"

    if [ -n "$keycloak_url" ];
    then
	echo -e "KEYCLOAK_URL=$keycloak_url" >> "$LOCATION_OF_ENV_CONFIG"
    fi
	release_svc="${release_svc:-}"
	oci_release_svc="${oci_release_svc:-}"
	tink_stack_svc="${tink_stack_svc:-}"
	tink_server_svc="${tink_server_svc:-}"
	onboarding_manager_svc="${onboarding_manager_svc:-}"
	onboarding_stream_svc="${onboarding_stream_svc:-}"
	en_http_proxy="${en_http_proxy:-}"
	en_no_proxy="${en_no_proxy:-}"
	
    if [ -n "$release_svc" ];
    then
	{
	echo -e "release_svc=$release_svc" 
	echo -e "oci_release_svc=$oci_release_svc" 
	echo -e "tink_stack_svc=$tink_stack_svc" 
	echo -e "tink_server_svc=$tink_server_svc"
	echo -e "onboarding_manager_svc=$onboarding_manager_svc"
	echo -e "onboarding_stream_svc=$onboarding_stream_svc"
	echo -e "OBM_PORT=443"
	} >> "$LOCATION_OF_ENV_CONFIG"
    fi

    if [ -n "$logging_svc" ];
    then
	echo -e "logging_svc=$logging_svc" >> "$LOCATION_OF_ENV_CONFIG"
    fi


    # Add proxy configs
    if [ -n "$en_https_proxy" ];
    then
	{
	echo -e "http_proxy=$en_http_proxy" 
	echo -e "https_proxy=$en_https_proxy" 
	echo -e "no_proxy=$en_no_proxy" 
	} >> "$LOCATION_OF_ENV_CONFIG"
    fi	
}

get_cert(){	

	if [ ! -f /etc/ssl/boots-ca-cert/ca.crt ]; then
		echo "======== file is not present ========"
		exit 1
	fi

	if [ ! -s /etc/ssl/boots-ca-cert/ca.crt ]; then
		echo "======== file size is zero ========"
		exit 1
	fi
	if [ ! -f /etc/ssl/orch-ca-cert/ca.crt ]; then
		echo "======== file is not present ========"
		exit 1
	fi

	if [ ! -s /etc/ssl/orch-ca-cert/ca.crt ]; then
		echo "======== file size is zero ========"
		exit 1
	fi

	# Get CA certificates
	cp /etc/ssl/orch-ca-cert/ca.crt "$IDP"/server_cert.pem
	cp /etc/ssl/orch-ca-cert/ca.crt "$IDP"/ca.pem

	#Boots certificates
	echo "" >> "$IDP"/ca.pem
	cat /etc/ssl/boots-ca-cert/ca.crt >> "$IDP"/ca.pem	
}
#######################################################################################################
#
#create a new image from exsisting initramfs image.
#
extract_tar_files() {
    mkdir -p "$EXTRACTED_FILES_LOCATION"
    if ! tar -xf "$CPIO_OUTPUT/$uos_file_name" -C "$EXTRACTED_FILES_LOCATION"; then
        echo "unable to uncompress tar $CPIO_OUTPUT/$uos_file_name"
        exit 1
    fi
    ls "$EXTRACTED_FILES_LOCATION"
}

rename_kernel_files() {
    if initramfs_file=$(find "$EXTRACTED_FILES_LOCATION/" -type f -name 'initramfs*' | head -n 1); then
        mv "$initramfs_file" "$EXTRACTED_FILES_LOCATION/initramfs-x86_64"
        echo "Renamed $initramfs_file to initramfs-x86_64"
    else
        echo "Error: initramfs not exist"
        exit 1
    fi

    if vmlinuz_file=$(find "$EXTRACTED_FILES_LOCATION/" -name 'vmlinuz*' | head -n 1); then
        mv "$vmlinuz_file" "$EXTRACTED_FILES_LOCATION/vmlinuz-x86_64"
        echo "Renamed $vmlinuz_file to vmlinuz-x86_64"
    else
        echo "Error: vmlinuz not exist"
        exit 1
    fi
}

extract_and_prepare_rootfs() {
    mkdir -p "$EXTRACTED_FILES_LOCATION/extract_initramfs"
    fakeroot sh -c "zcat $EXTRACTED_FILES_LOCATION/initramfs-x86_64 | cpio -idmv -D $EXTRACTED_FILES_LOCATION/extract_initramfs"
    rm "$EXTRACTED_FILES_LOCATION/initramfs-x86_64"
    mkdir -p "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp"
    gzip -d "$EXTRACTED_FILES_LOCATION/extract_initramfs/rootfs.tar.gz"
    mv "$EXTRACTED_FILES_LOCATION/extract_initramfs/rootfs.tar" "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp"
}

copy_cert_and_env_files() {
    mkdir -p "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/etc/pki/ca-trust/source/anchors/"
    cp "$IDP/ca.pem" "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/etc/pki/ca-trust/source/anchors/"
    cp "$IDP/server_cert.pem" "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/etc/pki/ca-trust/source/anchors/"
    tar -uf "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/rootfs.tar" -C "$PWD" ./etc/emf/env_config
    mkdir -p "$PWD/etc/hook/"
    cp "$PWD/etc/emf/env_config" "$PWD/etc/hook/env_config"
    tar -uf "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/rootfs.tar" -C "$PWD" ./etc/hook/env_config
    tar -uf "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/rootfs.tar" -C "$PWD" ./etc/idp
}

copy_service_files() {
    chmod +x "$PWD/etc/fluent-bit/fluentbit_run.sh"
    tar -uf "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/rootfs.tar" -C "$PWD" ./etc/fluent-bit/
    chmod +x "$PWD/etc/caddy/caddy_run.sh"
    tar -uf "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/rootfs.tar" -C "$PWD" ./etc/caddy/
}

update_systemd_services() {
    pushd "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/" || exit

    tar -xvf rootfs.tar ./usr/lib/systemd/system/caddy.service
    sed -i 's|User=caddy|User=root|' ./usr/lib/systemd/system/caddy.service
    sed -i 's|Group=caddy|Group=root|' ./usr/lib/systemd/system/caddy.service
    sed -i 's|ExecStartPre=/usr/bin/caddy validate --config /etc/caddy/Caddyfile||' ./usr/lib/systemd/system/caddy.service
    sed -i 's|ExecReload=/usr/bin/caddy reload --config /etc/caddy/Caddyfile||' ./usr/lib/systemd/system/caddy.service
    sed -i 's|ExecStart=/usr/bin/caddy run --environ --config /etc/caddy/Caddyfile|ExecStart=/etc/caddy/caddy_run.sh|' ./usr/lib/systemd/system/caddy.service
    sed -i '/^ExecStart=.*caddy_run\.sh$/a ReadWritePaths=/etc/pki/ca-trust' ./usr/lib/systemd/system/caddy.service
    sed -i '/^\[Unit\]/,/^$/s/^After=network.target network-online.target/After=network.target network-online.target device-discovery.service/' ./usr/lib/systemd/system/caddy.service
    sed -i '/^\[Unit\]/,/^$/s/^Requires=network-online.target/Requires=network-online.target device-discovery.service/' ./usr/lib/systemd/system/caddy.service

    tar -xvf rootfs.tar ./usr/lib/systemd/system/fluent-bit.service
    sed -i 's|ExecStart=/usr/bin/fluent-bit -c /etc/fluent-bit/fluent-bit.conf|ExecStart=/etc/fluent-bit/fluentbit_run.sh|' ./usr/lib/systemd/system/fluent-bit.service
    sed -i '/^\[Unit\]/,/^$/s/^After=network.target/After=network.target caddy.service/' ./usr/lib/systemd/system/fluent-bit.service
    sed -i '/^\[Unit\]/,/^$/s/^Requires=network.target/Requires=network.target caddy.service/' ./usr/lib/systemd/system/fluent-bit.service

    tar -xvf rootfs.tar ./usr/lib/systemd/system/tink-worker.service
    sed -i '/^\[Unit\]/,/^$/s/^After=network.target/After=network.target caddy.service/' ./usr/lib/systemd/system/tink-worker.service
    sed -i '/^After=network.target caddy.service$/a Requires=caddy.service' ./usr/lib/systemd/system/tink-worker.service

    tar -uf rootfs.tar ./usr/lib/systemd/system/caddy.service
    tar -uf rootfs.tar ./usr/lib/systemd/system/fluent-bit.service
    tar -uf rootfs.tar ./usr/lib/systemd/system/tink-worker.service

    #Add crt for tink-worker
    tar -uf rootfs.tar ./etc/pki/ca-trust/source/anchors/server_cert.pem
    tar -uf rootfs.tar ./etc/pki/ca-trust/source/anchors/ca.pem

    popd || exit
}

setup_getty_autologin() {
    pushd "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/" || exit
    tar -xvf rootfs.tar ./usr/lib/systemd/system/getty@.service
    mkdir -p ./etc/systemd/system/
    cp ./usr/lib/systemd/system/getty@.service ./etc/systemd/system/getty@tty1.service
    sed -i 's|^ExecStart=.*agetty.*|ExecStart=-/usr/sbin/agetty --autologin root --noclear %I|' ./etc/systemd/system/getty@tty1.service
    sed -i '/^ConditionPathExists=/a Requires=device-discovery.service' ./etc/systemd/system/getty@tty1.service
    sed -i '/^ConditionPathExists=/a After=device-discovery.service' ./etc/systemd/system/getty@tty1.service
    sed -i '/^DefaultInstance=tty1/a Alias=getty@tty1.service' ./etc/systemd/system/getty@tty1.service
    tar --delete -f rootfs.tar ./etc/systemd/system/getty.target.wants/getty@tty1.service
    mkdir -p ./etc/systemd/system/getty.target.wants/
    ln -s /etc/systemd/system/getty@tty1.service ./etc/systemd/system/getty.target.wants/getty@tty1.service
    tar -uf rootfs.tar ./etc/systemd/system/getty@tty1.service
    tar -rf rootfs.tar ./etc/systemd/system/getty.target.wants/getty@tty1.service
    popd || exit
}

repack_and_cleanup() {
    pushd "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/" || exit
    gzip -c rootfs.tar > ../rootfs.tar.gz
    popd || exit
    rm -r "$EXTRACTED_FILES_LOCATION/extract_initramfs/roottmp/"
    pushd "$EXTRACTED_FILES_LOCATION/extract_initramfs/" || exit
    fakeroot sh -c "find . | cpio -o -H newc | gzip -9 > ../initramfs-x86_64"
    popd || exit
}

#######################################################################################################
main() {

    create_env_config
    get_cert

	extract_tar_files
    rename_kernel_files
    extract_and_prepare_rootfs
    copy_cert_and_env_files
    copy_service_files
    update_systemd_services
    setup_getty_autologin
    repack_and_cleanup

    pushd ../ || exit
    resign_uos
    popd || exit
}
#######################################################################################################
main
