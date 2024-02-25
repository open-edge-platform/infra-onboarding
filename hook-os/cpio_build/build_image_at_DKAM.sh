#!/bin/bash

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

#set -x
pushd ../

source ./config
source ./secure_hookos.sh

popd

LOCATION_OF_EXTRA_FILES=$PWD/etc
extras_cpio=$PWD/additional_files.cpio.gz
# old_initramfs=$PWD/initramfs-x86_64
# new_initramfs=$PWD/initramfs-x86_64_new
STORE_ALPINE=$STORE_ALPINE_SECUREBOOT/../alpine_image

#
#create a new image from exsisting initramfs image after adding the extra_cpio archive into it.
#
extract_alpine_tar() {

    hookos_tar_files_to_update=($STORE_ALPINE )

    # run this for alpine_image folder and alpine_image_secureboot.
    # In current case we dont need to run the loop for secureboot.
    # But keeping the loop logic so that in future if needed and be enabled.
    for iter_folder in  ${hookos_tar_files_to_update[@]};
    do

	mkdir -p $iter_folder/hook_x86_64_files
	tar -xf $iter_folder/hook_x86_64.tar.gz -C $iter_folder/hook_x86_64_files
	if [ $? -ne 0 ];
	then
	    echo "unable to uncompress tar $iter_folder/hook_x86_64.tar.gz"
	    exit 1
	fi

	# cat 2 gz images to create final one.
	cat $iter_folder/hook_x86_64_files/initramfs-x86_64 $extras_cpio > $iter_folder/hook_x86_64_files/initramfs-x86_64_new
	if [ $? -ne 0 ];
	then
	    echo "unable to create a new initramfs image"
	    exit 1
	fi

	mv $iter_folder/hook_x86_64_files/initramfs-x86_64_new $iter_folder/hook_x86_64_files/initramfs-x86_64
	if [ $? -ne 0 ];
	then
	    echo "unable to move files $iter_folder/hook_x86_64_files/initramfs-x86_64_new"
	    exit 1
	fi

	pushd $iter_folder/hook_x86_64_files

	tar -czvf hook_x86_64.tar.gz .
	if [ ! -f hook_x86_64.tar.gz ];
	then
	    echo "unable to compress files"
	    exit 1
	fi

	mv -f $iter_folder/hook_x86_64_files/hook_x86_64.tar.gz $iter_folder/hook_x86_64.tar.gz
	if [ $? -ne 0 ];
	then
	    echo "unable to move files $iter_folder/hook_x86_64_files/initramfs-x86_64_new"
	    exit 1
	fi

	popd

    done


    cp $STORE_ALPINE/hook_x86_64.tar.gz $STORE_ALPINE_SECUREBOOT/hook_x86_64.tar.gz
    if [ $? -ne 0 ];
    then
	echo "Copy of hook tar file from alpine_image to alpine_image_secureboot"
	exit 1
    fi
}


main() {

    # if pax is not installed then check and install
    if ! command -v pax &> /dev/null
    then
	sudo apt install pax -y
    fi

    pax -x sv4cpio -w etc | gzip -c > $extras_cpio

    # cat $old_initramfs $extras_cpio > $new_initramfs


    extract_alpine_tar

    pushd ../
    resign_hookos
    popd
}

main
