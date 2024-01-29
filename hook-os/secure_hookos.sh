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
#######
GPG_KEY_DIR=$PWD/gpg_key
SB_KEYS_DIR=$PWD/sb_keys
PROXY=1
public_gpg_key=$PWD/boot.key
STORE_ALPINE_SECUREBOOT=$PWD/alpine_image_secureboot/
GRUB_CFG_LOC=${PWD}/grub.cfg
GRUB_SRC=$PWD/grub_source
BOOTX_LOC=$PWD/BOOTX64.efi
tinkerbell_owner=${load_balancer_ip:-192.168.1.120}
#mac_address_current_device=$(cat /proc/cmdline | grep -o "instance_id=..:..:..:..:..:.. " | awk ' {split($0,a,"="); print a[2]} ')
mac_address_current_device="net_default_mac_user"

MODULES="fat ext2 part_gpt normal
linux ls boot echo reboot search
search_label help signature_test pgp crypto gcry_dsa gcry_rsa gcry_sha1 gcry_sha512 gcry_sha256
configfile net loadenv"

EXTRA_TINK_OPTIONS="tinkerbell=http://$tinkerbell_owner syslog_host=$tinkerbell_owner packet_action=workflow console=ttyS0,11520 tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0 grpc_authority=$tinkerbell_owner:42113 packet_base_url=http://$tinkerbell_owner:8080/workflow tinkerbell_tls=false insecure_registries=$pd_host_ip:5015 instance_id=\${$mac_address_current_device} worker_id=\${$mac_address_current_device} packet_bootdev_mac=\${$mac_address_current_device} facility=sandbox"

EXTRA_TINK_OPTIONS_PROXY="http_proxy=http://proxy.iind.intel.com:911 https_proxy=http://proxy.iind.intel.com:911 no_proxy=localhost,.intel.com,127.0.0.0/8,172.16.0.0/20,192.168.0.0/16,10.0.0.0/8 HTTP_PROXY=http://proxy.iind.intel.com:911 HTTPS_PROXY=http://proxy.iind.intel.com:911 NO_PROXY=localhost,.intel.com,127.0.0.0/8,172.16.0.0/20,192.168.0.0/16,10.0.0.0/8 "
######

create_grub_image() {
    # make grub-image from the newly compiled grub.
    $GRUB_SRC/grub-mkimage -O x86_64-efi --disable-shim-lock --pubkey $public_gpg_key -p "/EFI/hook/" -o $BOOTX_LOC $MODULES
    echo "created BOOTX64.efi"
}


compile_grub() {

    if [ -f $GRUB_SRC/grub-lib/bin/grub-mkimage ];
    then
        echo "Grub seems to be already built; delete grub$GRUB_SRC to recompile"
        return
    fi

    git clone https://git.savannah.gnu.org/git/grub.git $GRUB_SRC
    pushd $GRUB_SRC
    ./bootstrap
    ./configure --with-platform=efi --libdir=$GRUB_SRC/grub-lib --prefix=$GRUB_SRC/grub-lib
    make -j
    make install
    popd
}


create_gpg_key() {
    if [ -f $public_gpg_key ];
    then
        echo "Reuse of existing key; Delete $public_gpg_key to Regenerate"
        return
    fi

    mkdir -p $GPG_KEY_DIR
    echo '%no-protection
Key-Type:1
Key-Length:2048
Subkey-Type:1
Subkey-Length:2048
Name-Real: Boot verifier
Expire-Date:0
%commit'| gpg --batch  --gen-key --homedir $GPG_KEY_DIR

    gpg --homedir $GPG_KEY_DIR --export > $public_gpg_key
}

sign_all_components() {

    #temp untar to sign images only
    rm -rf $STORE_ALPINE_SECUREBOOT/hook_sign_temp
    mkdir -p $STORE_ALPINE_SECUREBOOT/hook_sign_temp

    tar -xvf $STORE_ALPINE_SECUREBOOT/hook_x86_64.tar.gz -C $STORE_ALPINE_SECUREBOOT/hook_sign_temp

    mkdir -p $GPG_KEY_DIR

    pushd $STORE_ALPINE_SECUREBOOT/hook_sign_temp

    KEY_ID=$(gpg --homedir $GPG_KEY_DIR --list-secret-keys --keyid-format LONG | grep sec | awk '{print $2}' | cut -d '/' -f2)

    #grub.cfg
    cp $GRUB_CFG_LOC .
    rm -rf grub.cfg.sig
    gpg --batch  --homedir $GPG_KEY_DIR --local-user $KEY_ID --detach-sign grub.cfg
    if [ $? != 0 ];
    then
        echo "Failed to gpg sign grub.cfg"
        exit
    fi

    #vmlinuz

    #########
    # need to UEFI sb sign before gpg takes its signature.
    uefi_sign_grub_vmlinuz
    #########

    rm -rf vmlinuz-x86_64.sig
    gpg --batch --homedir $GPG_KEY_DIR --local-user $KEY_ID --detach-sign vmlinuz-x86_64
    if [ $? != 0 ];
    then
        echo "Failed to gpg sign vmlinuz"
        exit
    fi

    #initramfs
    rm -rf initramfs-x86_64.sig
    gpg --batch  --homedir $GPG_KEY_DIR --local-user $KEY_ID --detach-sign initramfs-x86_64
    if [ $? != 0 ];
    then
        echo "Failed to gpg sign initramfs"
        exit
    fi

    popd

}

uefi_sign_grub_vmlinuz() {

    sbsign --key $SB_KEYS_DIR/db.key --cert $SB_KEYS_DIR/db.crt --output $STORE_ALPINE_SECUREBOOT/hook_sign_temp/BOOTX64.efi $BOOTX_LOC
    if [ $? != 0 ];
    then
        echo "Failed to sign grub image"
        exit
    fi

    sbsign --key $SB_KEYS_DIR/db.key --cert $SB_KEYS_DIR/db.crt --output $STORE_ALPINE_SECUREBOOT/hook_sign_temp/vmlinuz-x86_64 $STORE_ALPINE_SECUREBOOT/hook_sign_temp/vmlinuz-x86_64
    if [ $? != 0 ];
    then
        echo "Failed to sign vmlinuz image"
        exit
    fi
}

generate_pk_kek_db() {

    #verfy that pk kek db is already present.
    # if [ -d $SB_KEYS_DIR ] || [ -f $SB_KEYS_DIR/db.crt ] ;
    # then
    #     echo "Seems like Secure boot $SB_KEYS_DIR are already present. Reusing the same"
    #     return
    # fi

    mkdir -p $SB_KEYS_DIR
    pushd $SB_KEYS_DIR

    GUID=$(uuidgen)

    echo $GUID

    [ -f $SB_KEYS_DIR/pk.crt ]    || openssl req -newkey rsa:2048 -nodes -keyout pk.key -new -x509 -sha256 -days 3650 -subj "/CN=Secure Boot PK/" -out pk.crt
    [ -f $SB_KEYS_DIR/pk.der ]    || openssl x509 -outform DER -in pk.crt -out pk.der
    [ -f $SB_KEYS_DIR/pk.esl ]    || cert-to-efi-sig-list -g $GUID pk.crt pk.esl
    [ -f $SB_KEYS_DIR/pk.auth ]   || sign-efi-sig-list -g $GUID -k pk.key -c pk.crt pk pk.esl pk.auth
    [ -f $SB_KEYS_DIR/nopk.auth ] || sign-efi-sig-list -g $GUID -c pk.crt -k pk.key pk /dev/null nopk.auth

    [ -f $SB_KEYS_DIR/kek.crt ] || openssl req -newkey rsa:2048 -nodes -keyout kek.key -new -x509 -sha256 -days 3650 -subj "/CN=Secure Boot KEK/" -out kek.crt
    [ -f $SB_KEYS_DIR/kek.der ] || openssl x509 -outform DER -in kek.crt -out kek.der
    [ -f $SB_KEYS_DIR/kek.esl ] || cert-to-efi-sig-list -g $GUID kek.crt kek.esl
    [ -f $SB_KEYS_DIR/kek.auth ] || sign-efi-sig-list -g $GUID -k pk.key -c pk.crt kek kek.esl kek.auth

    [ -f $SB_KEYS_DIR/db.crt ] || openssl req -newkey rsa:2048 -nodes -keyout db.key -new -x509 -sha256 -days 3650 -subj "/CN=Secure Boot DB/" -out db.crt
    [ -f $SB_KEYS_DIR/db.der ] || openssl x509 -outform DER -in db.crt -out db.der
    [ -f $SB_KEYS_DIR/db.esl ] || cert-to-efi-sig-list -g $GUID db.crt db.esl
    [ -f $SB_KEYS_DIR/db.auth ] || sign-efi-sig-list -g $GUID -k kek.key -c kek.crt db db.esl db.auth

    if [ ! -f $SB_KEYS_DIR/pk.crt ] || [ ! -f $SB_KEYS_DIR/kek.crt ] || [ ! -f $SB_KEYS_DIR/db.crt ] ;
    then
        echo "Seems like some issue with UEFI keys generation. Check again"
        popd
        return
    fi
    popd
}

package_signed_hookOS(){

    # make one tar which has all signatures and efi binaries
    pushd $STORE_ALPINE_SECUREBOOT/hook_sign_temp

    sync

    tar -czvf hook_x86_64.tar.gz .

    mv $STORE_ALPINE_SECUREBOOT/hook_sign_temp/hook_x86_64.tar.gz $STORE_ALPINE_SECUREBOOT/hook_x86_64.tar.gz

    popd

}

create_grub_cfg() {
    cp $PWD/grub_template.cfg ${PWD}/grub.cfg

    if [ $PROXY == 1 ]; then
        echo "Ensure that the correct proxy is configured in the script: Else it will cause failure at the node"
        EXTRA_TINK_OPTIONS="$EXTRA_TINK_OPTIONS $EXTRA_TINK_OPTIONS_PROXY"
    fi

    sed -i "s+EXTRA_TINK_OPTIONS+$EXTRA_TINK_OPTIONS+g" ${PWD}/grub.cfg

}

secure_hookos() {
    #container/host setup
    sudo apt install -y autoconf automake make gcc m4 git gettext autopoint pkg-config autoconf-archive python3 bison flex gawk efitools
    #container/host setup done

    mkdir -p $STORE_ALPINE_SECUREBOOT
    create_grub_cfg
    compile_grub
    create_gpg_key
    create_grub_image
    generate_pk_kek_db

    sign_all_components
    package_signed_hookOS

    echo "Save db.der file on a FAT volume to enroll inside UEFI bios"
}

#secure_hookos
