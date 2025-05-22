#!/usr/bin/env bash

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

#set -x

# shellcheck source=/dev/null
source ./config
set -xuo pipefail

#######
GPG_KEY_DIR=$PWD/gpg_key
SB_KEYS_DIR=$PWD/../sb_keys
SERVER_CERT_DIR=$PWD/../server_certs
public_gpg_key=$PWD/boot.key
STORE_ALPINE_SECUREBOOT=$PWD/alpine_image_secureboot/
RSA_KEY_SIZE=4096
HASH_SIZE=512
en_http_proxy="${en_http_proxy:-}"
en_https_proxy="${en_https_proxy:-}"
en_no_proxy="${en_no_proxy:-}"

create_gpg_key() {
    if [ -f "$public_gpg_key" ];
    then
        echo "Reuse of existing key; Delete $public_gpg_key to Regenerate"
        return
    fi

    mkdir -p "$GPG_KEY_DIR"

    if ! echo "%no-protection
Key-Type:1
Key-Length:$RSA_KEY_SIZE
Subkey-Type:1
Subkey-Length:$RSA_KEY_SIZE
Name-Real: Boot verifier
Expire-Date:0
%commit"| gpg --batch  --gen-key --homedir "$GPG_KEY_DIR";
    then
	# Seems like gpg key creation failed with --homedir trying once with the default ~/.gnupg as the directory.
	# might happen with coder environment

	# gpg --export > "$public_gpg_key"
	if ! echo "%no-protection
Key-Type:1
Key-Length:$RSA_KEY_SIZE
Subkey-Type:1
Subkey-Length:$RSA_KEY_SIZE
Name-Real: Boot verifier
Expire-Date:0
%commit"| gpg --batch  --gen-key;
	then
	    echo "gpg agent is not install or is not working."
	    exit 1
	fi
	export GPG_KEY_DIR=~/.gnupg/
    fi

    gpg --homedir "$GPG_KEY_DIR" --export > "$public_gpg_key"
}

sign_all_components() {

    #temp untar to sign images only
    rm -rf "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp
    mkdir -p "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp

    tar -xvf "$STORE_ALPINE_SECUREBOOT"/hook_x86_64.tar.gz -C "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp

    mkdir -p "$GPG_KEY_DIR"

    pushd "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp || exit

    KEY_ID=$(gpg --homedir "$GPG_KEY_DIR" --list-secret-keys --keyid-format LONG | grep sec | awk '{print $2}' | cut -d '/' -f2)


    #vmlinuz

    #########
    # need to UEFI sb sign before gpg takes its signature.
    uefi_sign_vmlinuz
    #########

    rm -rf vmlinuz-x86_64.sig
    
    if ! gpg --batch --homedir "$GPG_KEY_DIR" --local-user "$KEY_ID" --detach-sign vmlinuz-x86_64;
    then
        echo "Failed to gpg sign vmlinuz"
        exit
    fi

    #initramfs
    rm -rf initramfs-x86_64.sig
    
    ######## repack initramfs to zstd format ##################
    mv initramfs-x86_64 initramfs-x86_64.gz
    gunzip -c initramfs-x86_64.gz | xz --check=crc32 -k -T 6 > initramfs-x86_64
    rm initramfs-x86_64.gz
    ###########################################################

    if ! gpg --batch  --homedir "$GPG_KEY_DIR" --local-user "$KEY_ID" --detach-sign initramfs-x86_64;
    then
        echo "Failed to gpg sign initramfs"
        exit
    fi

    popd || exit

    # copy public key used validation of HookOS to archive
    cp "$SB_KEYS_DIR"/db.der "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp/hookos_db.der
}

uefi_sign_vmlinuz() {
    
    if ! sbsign --key "$SB_KEYS_DIR"/db.key --cert "$SB_KEYS_DIR"/db.crt --output "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp/vmlinuz-x86_64 "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp/vmlinuz-x86_64;
    then
        echo "Failed to sign vmlinuz image"
        exit
    fi
}

generate_pk_kek_db() {

    #verfy that pk kek db is already present.
    # if [ -d "$SB_KEYS_DIR" ] || [ -f "$SB_KEYS_DIR"/db.crt ] ;
    # then
    #     echo "Seems like Secure boot "$SB_KEYS_DIR" are already present. Reusing the same"
    #     return
    # fi

    mkdir -p "$SB_KEYS_DIR"
    pushd "$SB_KEYS_DIR" || exit

    GUID=$(uuidgen)

    echo "$GUID"

    [ -f "$SB_KEYS_DIR"/pk.crt ]    || openssl req -newkey rsa:$RSA_KEY_SIZE -nodes -keyout pk.key -new -x509 -sha$HASH_SIZE -days 3650 -subj "/CN=Secure Boot PK/" -out pk.crt
    [ -f "$SB_KEYS_DIR"/pk.der ]    || openssl x509 -outform DER -in pk.crt -out pk.der
    [ -f "$SB_KEYS_DIR"/pk.esl ]    || cert-to-efi-sig-list -g "$GUID" pk.crt pk.esl
    [ -f "$SB_KEYS_DIR"/pk.auth ]   || sign-efi-sig-list -g "$GUID" -k pk.key -c pk.crt pk pk.esl pk.auth
    [ -f "$SB_KEYS_DIR"/nopk.auth ] || sign-efi-sig-list -g "$GUID" -c pk.crt -k pk.key pk /dev/null nopk.auth

    [ -f "$SB_KEYS_DIR"/kek.crt ] || openssl req -newkey rsa:$RSA_KEY_SIZE -nodes -keyout kek.key -new -x509 -sha$HASH_SIZE -days 3650 -subj "/CN=Secure Boot KEK/" -out kek.crt
    [ -f "$SB_KEYS_DIR"/kek.der ] || openssl x509 -outform DER -in kek.crt -out kek.der
    [ -f "$SB_KEYS_DIR"/kek.esl ] || cert-to-efi-sig-list -g "$GUID" kek.crt kek.esl
    [ -f "$SB_KEYS_DIR"/kek.auth ] || sign-efi-sig-list -g "$GUID" -k pk.key -c pk.crt kek kek.esl kek.auth

    [ -f "$SB_KEYS_DIR"/db.crt ] || openssl req -newkey rsa:$RSA_KEY_SIZE -nodes -keyout db.key -new -x509 -sha$HASH_SIZE -days 3650 -subj "/CN=Secure Boot DB/" -out db.crt
    [ -f "$SB_KEYS_DIR"/db.der ] || openssl x509 -outform DER -in db.crt -out db.der
    [ -f "$SB_KEYS_DIR"/db.esl ] || cert-to-efi-sig-list -g "$GUID" db.crt db.esl
    [ -f "$SB_KEYS_DIR"/db.auth ] || sign-efi-sig-list -g "$GUID" -k kek.key -c kek.crt db db.esl db.auth

    if [ ! -f "$SB_KEYS_DIR"/pk.crt ] || [ ! -f "$SB_KEYS_DIR"/kek.crt ] || [ ! -f "$SB_KEYS_DIR"/db.crt ] ;
    then
        echo "Seems like some issue with UEFI keys generation. Check again"
        popd || exit
        return
    fi
    popd || exit
}

package_signed_hookOS(){

    # make one tar which has all signatures and efi binaries
    pushd "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp || exit

    sync

    tar -czvf hook_x86_64.tar.gz .
    if [ -d "/data" ]; then
        echo "Path /data exists."
        cp "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp/initramfs-x86_64 /data
        cp "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp/vmlinuz-x86_64 /data
        cp "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp/hook_x86_64.tar.gz /data
    else
        echo "Path /data does not exist."
        mv -f "$STORE_ALPINE_SECUREBOOT"/hook_sign_temp/hook_x86_64.tar.gz "$STORE_ALPINE_SECUREBOOT"/hook_x86_64.tar.gz
    fi 

    popd || exit

}

secure_hookos() {

    echo "in secure_hookos()"

    #container/host setup
    sudo apt install -y autoconf automake make gcc m4 git gettext autopoint pkg-config autoconf-archive python3 bison flex gawk efitools
    #container/host setup done

    mkdir -p "$STORE_ALPINE_SECUREBOOT"
    create_gpg_key
    generate_pk_kek_db

    sign_all_components
    package_signed_hookOS

    echo "Save db.der file on a FAT volume to enroll inside UEFI bios"
}

resign_hookos() {

    if [ ! -f "$STORE_ALPINE_SECUREBOOT"/hook_x86_64.tar.gz ] ;
    then
	echo "Place the hook image at ""$STORE_ALPINE_SECUREBOOT""/hook_x86_64.tar.gz to proceed."
	exit 1
    fi


    create_gpg_key

    sign_all_components
    package_signed_hookOS
    rm -rf "$GPG_KEY_DIR"
    rm -rf "$SB_KEYS_DIR"
    rm -rf "$SERVER_CERT_DIR"
    rm -rf "$public_gpg_key"
    rm -rf "$STORE_ALPINE"

    echo "Save db.der file on a FAT volume to enroll inside UEFI bios"
}
