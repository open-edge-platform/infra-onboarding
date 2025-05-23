#!/bin/bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#set -x

set -xueo pipefail
data_dir=$1
pushd ../

# shellcheck source=/dev/null
source ./config
# shellcheck source=/dev/null
#source ./secure_hookos.sh
#STORE_ALPINE="$STORE_ALPINE_SECUREBOOT"/../alpine_image
#mkdir -p "$STORE_ALPINE_SECUREBOOT"
STORE_ALPINE=output
mkdir -p "$STORE_ALPINE"

#TEMPORARY
cp "$data_dir"/hook_x86_64.tar.gz "$data_dir"/emt_uos_x86_64.tar.gz


cp "$data_dir"/emt_uos_x86_64.tar.gz "$STORE_ALPINE"

popd || exit

LOCATION_OF_EXTRA_FILES=$PWD/etc
LOCATION_OF_ENV_CONFIG=$PWD/etc/emt/env_config
LOCATION_OF_HOOK_ENV=$PWD/etc/emt/
extras_cpio=$PWD/additional_files.cpio.gz
IDP=$LOCATION_OF_EXTRA_FILES/idp
# old_initramfs=$PWD/initramfs-x86_64
# new_initramfs=$PWD/initramfs-x86_64_new
#STORE_ALPINE="$STORE_ALPINE_SECUREBOOT"/../alpine_image
STORE_ALPINE=../output
LOCATION_OF_FLUENTBIT_YAML=$PWD/etc/emt/fluent-bit/fluent-bit.yaml
LOCATION_OF_CADDY_FILES=$PWD/etc/emt/caddy

#######################################################################################################
create_env_config() {

    #Just to double confirm that the folder is available.
    mkdir -p "$LOCATION_OF_HOOK_ENV"

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

    # only for the extra hosts which is a list we need to add this change the env file needs to
    # the quotes else the source will fail.
    if [ -n "$extra_hosts" ];
    then
	echo -e 'EXTRA_HOSTS="'"$extra_hosts"'"' >> "$LOCATION_OF_ENV_CONFIG"
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
		exit 0
	fi

	if [ ! -s /etc/ssl/boots-ca-cert/ca.crt ]; then
		echo "======== file size is zero ========"
		exit 0
	fi
	if [ ! -f /etc/ssl/orch-ca-cert/ca.crt ]; then
		echo "======== file is not present ========"
		exit 0
	fi

	if [ ! -s /etc/ssl/orch-ca-cert/ca.crt ]; then
		echo "======== file size is zero ========"
		exit 0
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
#create a new image from exsisting initramfs image after adding the extra_cpio archive into it.
#
extract_emt_tar() {

    hookos_tar_files_to_update=("$STORE_ALPINE" )

    # run this for alpine_image folder and alpine_image_secureboot.
    # In current case we dont need to run the loop for secureboot.
    # But keeping the loop logic so that in future if needed and be enabled.
    for iter_folder in  "${hookos_tar_files_to_update[@]}";
    do
        echo "$iter_folder"
	mkdir -p "$iter_folder"/emt_uos_x86_64_files
	if ! tar -xf "$iter_folder"/emt_uos_x86_64.tar.gz -C "$iter_folder"/emt_uos_x86_64_files;
	then
	    echo "unable to uncompress tar $iter_folder/emt_uos_x86_64.tar.gz"
	    exit 1
	fi

	ls

	# Check for files matching initramfs* and vmlinuz* and rename then
	if initramfs_file=$(find "$iter_folder/emt_uos_x86_64_files/" -type f  -name 'initramfs*'| head -n 1); then
	    mv "$initramfs_file" "$iter_folder/emt_uos_x86_64_files/initramfs-x86_64"
            echo "Renamed $initramfs_file to initramfs-x86_64"
        else
	    echo "Error: initramfs not exist"
	    exit 1
	fi


	if vmlinuz_file=$(find "$iter_folder/emt_uos_x86_64_files/" -name 'vmlinuz*' | head -n 1); then
	    mv "$vmlinuz_file" "$iter_folder/emt_uos_x86_64_files/vmlinuz-x86_64"
	    echo "Renamed $vmlinuz_file to vmlinuz-x86_64"
	else 
	    echo "Error: vmlinuz not exist"
	    exit 1
	fi

	#Extract rootfs.tar.gz from initramfs and decompress
	mkdir -p $iter_folder/emt_uos_x86_64_files/extract_initramfs
	zcat $iter_folder/emt_uos_x86_64_files/initramfs-x86_64 | cpio -idmv -D $iter_folder/emt_uos_x86_64_files/extract_initramfs || true #> /dev/null 2>&1
	rm $iter_folder/emt_uos_x86_64_files/initramfs-x86_64
        mkdir -p $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp
	#tar -xvf $iter_folder/emt_uos_x86_64_files/extract_initramfs/rootfs.tar.gz -C $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp > /dev/null 2>&1
        gzip -d $iter_folder/emt_uos_x86_64_files/extract_initramfs/rootfs.tar.gz
	mv $iter_folder/emt_uos_x86_64_files/extract_initramfs/rootfs.tar $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp
	mkdir -p $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp/etc/pki/ca-trust/source/anchors/
        cp $IDP/Intel.crt $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp/etc/pki/ca-trust/source/anchors/

	#Copy env_config file and idp
	tar -uf $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp/rootfs.tar -C $PWD ./etc/emt/env_config
	#Workaround for device-discovery
	mkdir -p $PWD/etc/hook/
	cp $PWD/etc/emt/env_config $PWD/etc/hook/env_config
	tar -uf $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp/rootfs.tar -C $PWD ./etc/hook/env_config
	tar -uf $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp/rootfs.tar -C $PWD ./etc/idp

        #Copy fluent-bit file and update
	tar -uf $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp/rootfs.tar -C $PWD ./etc/fluent-bit/

	#Copy Caddy files and udpate
	tar -uf $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp/rootfs.tar -C $PWD ./etc/caddy/

        pushd "$iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp/" || exit
	tar -xvf rootfs.tar ./usr/lib/systemd/system/caddy.service
	sed -i 's|ExecStart=/usr/bin/caddy run --environ --config /etc/caddy/Caddyfile|ExecStart=/etc/caddy/caddy_run.sh|' ./usr/lib/systemd/system/caddy.service
	sed -i '/^\[Unit\]/,/^$/s/^After=network.target network-online.target/After=network.target network-online.target device-discovery.service/' ./usr/lib/systemd/system/caddy.service
	sed -i '/^\[Unit\]/,/^$/s/^Requires=network-online.target/Requires=network-online.target device-discovery.service/' ./usr/lib/systemd/system/caddy.service

	tar -xvf rootfs.tar ./usr/lib/systemd/system/fluent-bit.service
	sed -i 's|ExecStart=/usr/bin/fluent-bit -c /etc/fluent-bit/fluent-bit.conf|ExecStart=/etc/fluent-bit/fluentbit_run.sh|' ./usr/lib/systemd/system/fluent-bit.service
	sed -i '/^\[Unit\]/,/^$/s/^After=network.target/After=network.target caddy.service/' ./usr/lib/systemd/system/fluent-bit.service
	sed -i '/^\[Unit\]/,/^$/s/^Requires=network.target/Requires=network.target caddy.service/' ./usr/lib/systemd/system/fluent-bit.service
	
        tar -uf rootfs.tar ./usr/lib/systemd/system/caddy.service
	tar -uf rootfs.tar ./usr/lib/systemd/system/fluent-bit.service

	#Add crt for tink-worker
	tar -uf rootfs.tar ./etc/pki/ca-trust/source/anchors/Intel.crt

	gzip -c rootfs.tar > ../rootfs.tar.gz
	popd || exit
	rm -r $iter_folder/emt_uos_x86_64_files/extract_initramfs/roottmp/

	pushd "$iter_folder/emt_uos_x86_64_files/extract_initramfs/" || exit
	
        find . | cpio -o -H newc | gzip -9 > ../initramfs-x86_64

	popd || exit
    #     ls $iter_folder/
	# rm $iter_folder/emt_uos_x86_64.tar.gz
	# rm -rf $iter_folder/emt_uos_x86_64_files/extract_initramfs
	# tar -czvf $iter_folder/emt_uos_x86_64.tar.gz -C $iter_folder/emt_uos_x86_64_files . > /dev/null 2>&1
	# rm -rf $iter_folder/emt_uos_x86_64_files/
    done

}


#######################################################################################################
main() {

    create_env_config
    get_cert

    extract_emt_tar

    #pushd ../ || exit
    #resign_hookos
    #popd || exit
}
#######################################################################################################
main
