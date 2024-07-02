#!/bin/bash

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

#set -x

set -xuo pipefail
mode=$1
data_dir=$2
pushd ../

# shellcheck source=/dev/null
source ./config
# shellcheck source=/dev/null
source ./secure_hookos.sh
STORE_ALPINE="$STORE_ALPINE_SECUREBOOT"/../alpine_image
mkdir -p "$STORE_ALPINE_SECUREBOOT"
mkdir -p "$STORE_ALPINE"
tar -xf "$data_dir"/grub_source.tar.gz
rm -rf "$data_dir"/grub_source.tar.gz
mv "$data_dir"/hook_x86_64.tar.gz "$STORE_ALPINE"

popd || exit

LOCATION_OF_EXTRA_FILES=$PWD/etc
LOCATION_OF_ENV_CONFIG=$PWD/etc/hook/env_config
LOCATION_OF_HOOK_ENV=$PWD/etc/hook/
extras_cpio=$PWD/additional_files.cpio.gz
IDP=$LOCATION_OF_EXTRA_FILES/idp
# old_initramfs=$PWD/initramfs-x86_64
# new_initramfs=$PWD/initramfs-x86_64_new
STORE_ALPINE="$STORE_ALPINE_SECUREBOOT"/../alpine_image

#######################################################################################################
create_env_config() {

    #Just to double confirm that the folder is available.
    mkdir -p "$LOCATION_OF_HOOK_ENV"

    if [ -n "$keycloak_url" ];
    then
	echo -e "KEYCLOAK_URL=$keycloak_url" >> "$LOCATION_OF_ENV_CONFIG"
    fi
	fdo_owner_svc="${fdo_owner_svc:-}"
	release_svc="${release_svc:-}"
	oci_release_svc="${oci_release_svc:-}"
	tink_stack_svc="${tink_stack_svc:-}"
	tink_server_svc="${tink_server_svc:-}"
	onboarding_manager_svc="${onboarding_manager_svc:-}"
	http_proxy="${http_proxy:-}"
	no_proxy="${no_proxy:-}"
    if [ -n "$fdo_manufacturer_svc" ];
    then
	{
	echo -e "fdo_manufacturer_svc=$fdo_manufacturer_svc"
	echo -e "fdo_owner_svc=$fdo_owner_svc" 
	echo -e "release_svc=$release_svc" 
	echo -e "oci_release_svc=$oci_release_svc" 
	echo -e "tink_stack_svc=$tink_stack_svc" 
	echo -e "tink_server_svc=$tink_server_svc"
	echo -e "onboarding_manager_svc=$onboarding_manager_svc"
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
    if [ -n "$https_proxy" ];
    then
	{
	echo -e "http_proxy=$http_proxy" 
	echo -e "https_proxy=$https_proxy" 
	echo -e "no_proxy=$no_proxy" 
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

	#Intel CA
	echo "$mode"
	if [ "$mode" == "dev" ]; then
		echo "mode:"
		echo "" >> "$IDP"/ca.pem
		mkdir temp
		curl -o tmp.zip http://certificates.intel.com/repository/certificates/IntelSHA2RootChain-Base64.zip
		unzip tmp.zip -d temp
		for file in temp/*; do
			if [ -f "$file" ]; then
				cat "$file" >> "$IDP"/ca.pem
			fi
		done
		rm tmp.zip
		rm -rf temp
	fi
}
#######################################################################################################
#
#create a new image from exsisting initramfs image after adding the extra_cpio archive into it.
#
extract_alpine_tar() {

    hookos_tar_files_to_update=("$STORE_ALPINE" )

    # run this for alpine_image folder and alpine_image_secureboot.
    # In current case we dont need to run the loop for secureboot.
    # But keeping the loop logic so that in future if needed and be enabled.
    for iter_folder in  "${hookos_tar_files_to_update[@]}";
    do

	mkdir -p "$iter_folder"/hook_x86_64_files
	if ! tar -xf "$iter_folder"/hook_x86_64.tar.gz -C "$iter_folder"/hook_x86_64_files;
	then
	    echo "unable to uncompress tar $iter_folder/hook_x86_64.tar.gz"
	    exit 1
	fi

	# cat 2 gz images to create final one.
	if ! cat "$iter_folder"/hook_x86_64_files/initramfs-x86_64 "$extras_cpio" > "$iter_folder"/hook_x86_64_files/initramfs-x86_64_new;
	then
	    echo "unable to create a new initramfs image"
	    exit 1
	fi

	
	if ! mv "$iter_folder"/hook_x86_64_files/initramfs-x86_64_new "$iter_folder"/hook_x86_64_files/initramfs-x86_64;
	then
	    echo "unable to move files $iter_folder/hook_x86_64_files/initramfs-x86_64_new"
	    exit 1
	fi

	pushd "$iter_folder"/hook_x86_64_files || exit

	tar -czvf hook_x86_64.tar.gz .
	if [ ! -f hook_x86_64.tar.gz ];
	then
	    echo "unable to compress files"
	    exit 1
	fi

	
	if ! mv -f "$iter_folder"/hook_x86_64_files/hook_x86_64.tar.gz "$iter_folder"/hook_x86_64.tar.gz;
	then
	    echo "unable to move files $iter_folder/hook_x86_64_files/initramfs-x86_64_new"
	    exit 1
	fi

	popd || exit

    done


    if ! cp "$STORE_ALPINE"/hook_x86_64.tar.gz "$STORE_ALPINE_SECUREBOOT"/hook_x86_64.tar.gz;
    then
	echo "Copy of hook tar file from alpine_image to alpine_image_secureboot"
	exit 1
    fi
}


#######################################################################################################
main() {

    # if pax is not installed then check and install
    if ! command -v pax &> /dev/null
    then
	sudo apt install pax -y
    fi

    create_env_config
	get_cert

    pax -x sv4cpio -w etc | gzip -c > "$extras_cpio"

    # cat $old_initramfs $extras_cpio > $new_initramfs


    extract_alpine_tar

    pushd ../ || exit
    resign_hookos
    popd || exit
}
#######################################################################################################
main
