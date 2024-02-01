# SPDX-FileCopyrightText: (C) 2023 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

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

#source ../config
#######
working_dir=$1
echo $working_dir
GPG_KEY_DIR=$working_dir/gpg_key
SB_KEYS_DIR=$working_dir/sb_keys
PROXY=1
public_gpg_key=$working_dir/boot.key
STORE_ALPINE=$working_dir/store_alpine
GRUB_CFG_LOC=${working_dir}/grub.cfg
GRUB_SRC=$working_dir/grub_source
echo "$GRUB_SRC"
BOOTX_LOC=$working_dir/BOOTX64.efi
mac_address_current_device="net_default_mac_user"

#tinkerbell_owner=${load_balancer_ip:-192.168.1.120}
#EXTRA_TINK_OPTIONS="tinkerbell=http://$tinkerbell_owner syslog_host=$tinkerbell_owner packet_action=workflow console=ttyS0,11520 tink_worker_image=quay.io/tinkerbell/tink-worker:v0.8.0 grpc_authority=$tinkerbell_owner:42113 packet_base_url=http://$tinkerbell_owner:8080/workflow tinkerbell_tls=false insecure_registries=$pd_host_ip:5015 instance_id=\${$mac_address_current_device} worker_id=\${$mac_address_current_device} packet_bootdev_mac=\${$mac_address_current_device} facility=sandbox"
#EXTRA_TINK_OPTIONS_PROXY="http_proxy=http://proxy.iind.intel.com:911 https_proxy=http://proxy.iind.intel.com:911 no_proxy=localhost,.intel.com,127.0.0.0/8,172.16.0.0/20,192.168.0.0/16,10.0.0.0/8 HTTP_PROXY=http://proxy.iind.intel.com:911 HTTPS_PROXY=http://proxy.iind.intel.com:911 NO_PROXY=localhost,.intel.com,127.0.0.0/8,172.16.0.0/20,192.168.0.0/16,10.0.0.0/8 "

create_grub_image() {
    echo "Inside create grub image"
    # make grub-image from the newly compiled grub.
    $GRUB_SRC/grub-mkimage -O x86_64-efi --disable-shim-lock --pubkey $public_gpg_key -p "/EFI/hook/" -o $BOOTX_LOC $MODULES
    echo "created BOOTX64.efi"
}


compile_grub() {
    echo "Inside compile grub"

    if [ -f $GRUB_SRC/grub-lib/bin/grub-mkimage ];
    then
        echo "Grub seems to be already built; delete grub$GRUB_SRC to recompile"
        return
    fi

    git clone https://git.savannah.gnu.org/git/grub.git $GRUB_SRC
    cd $GRUB_SRC
    ./bootstrap
    ./configure --with-platform=efi --libdir=$GRUB_SRC/grub-lib --prefix=$GRUB_SRC/grub-lib
    make -j
    make install
    cd $working_dir
}


create_gpg_key() {
    echo "Inside create gpg key"
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
    echo "Inside sign all components"

#     #temp untar to sign images only
    cd $SB_KEYS_DIR
    openssl x509 -in db.crt -out db.der -outform DER
    rm -rf $STORE_ALPINE
    mkdir -p $STORE_ALPINE

    if [ -e "/data" ]; then
        echo "Path /data exists."
        tar -xvf /data/hook_x86_64.tar.gz -C $STORE_ALPINE
        mkdir -p /data/unsigned
        mv /data/hook_x86_64.tar.gz /data/unsigned
    else
        echo "Path /data does not exist."
        if [ ! -f $working_dir/hook_x86_64.tar.gz ]; then
            rm -rf $GPG_KEY_DIR
            rm -rf $SB_KEYS_DIR
            rm -rf $working_dir/server_certs
            rm -rf $public_gpg_key
            rm -rf $STORE_ALPINE
            rm -rf $GRUB_SRC
            rm -rf $BOOTX_LOC
            exit 0
        fi
        tar -xvf $working_dir/hook_x86_64.tar.gz -C $STORE_ALPINE
        mkdir -p $working_dir/unsigned
        mv $working_dir/hook_x86_64.tar.gz $working_dir/unsigned
    fi     

    

    mkdir -p $GPG_KEY_DIR

    cd $STORE_ALPINE

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

#     #vmlinuz

#     #########
#     # need to UEFI sb sign before gpg takes its signature.
     uefi_sign_grub_vmlinuz
    #########

     rm -rf vmlinuz-x86_64.sig
     gpg --batch --homedir $GPG_KEY_DIR --local-user $KEY_ID --detach-sign vmlinuz-x86_64
     if [ $? != 0 ];
     then
         echo "Failed to gpg sign vmlinuz"
         exit
     fi

#     #initramfs
     rm -rf initramfs-x86_64.sig
     gpg --batch  --homedir $GPG_KEY_DIR --local-user $KEY_ID --detach-sign initramfs-x86_64
     if [ $? != 0 ];
     then
         echo "Failed to gpg sign initramfs"
         exit
     fi

     cd $working_dir

 }

uefi_sign_grub_vmlinuz() {
     
     sbsign --key $SB_KEYS_DIR/db.key --cert $SB_KEYS_DIR/db.crt --output $STORE_ALPINE/BOOTX64.efi $BOOTX_LOC
     if [ $? != 0 ];
     then
         echo "Failed to sign grub image"
         exit
     fi

     sbsign --key $SB_KEYS_DIR/db.key --cert $SB_KEYS_DIR/db.crt --output $STORE_ALPINE/vmlinuz-x86_64 $STORE_ALPINE/vmlinuz-x86_64
     if [ $? != 0 ];
     then
         echo "Failed to sign vmlinuz image"
         exit
     fi
 }

package_signed_hookOS(){

#     # make one tar which has all signatures and efi binaries
     cd $STORE_ALPINE

     sync

     #tar -czvf hook_x86_64.tar.gz .
    if [ -e "/data" ]; then
        echo "Path /data exists."
        cp $STORE_ALPINE/initramfs-x86_64 /data
        cp $STORE_ALPINE/vmlinuz-x86_64 /data
        #cp $SB_KEYS_DIR/db.crt /data/keys
    else
        echo "Path /data does not exist."
        tar -czvf hook_x86_64.tar.gz .
    fi 

     cd $working_dir

 }

create_grub_cfg() {
     cp $PWD/grub_template.cfg ${PWD}/grub.cfg

     if [ "$PROXY" = "1" ]; then
         echo "Ensure that the correct proxy is configured in the script: Else it will cause failure at the node"
         EXTRA_TINK_OPTIONS="$EXTRA_TINK_OPTIONS $EXTRA_TINK_OPTIONS_PROXY"
     fi

     sed -i "s+EXTRA_TINK_OPTIONS+$EXTRA_TINK_OPTIONS+g" ${PWD}/grub.cfg

 }

secure_hookos() {
#     #container/host setup
      apt install -y autoconf automake make gcc m4 git gettext autopoint pkg-config autoconf-archive python3 bison flex gawk efitools sbsigntool
#     #container/host setup done

     create_grub_cfg
     compile_grub
     create_gpg_key
     create_grub_image
     #generate_bios_certs
     sign_all_components
     package_signed_hookOS

     echo "Save db.cer file on a FAT volume to enroll inside UEFI bios"
 }

secure_hookos
rm -rf $GPG_KEY_DIR
rm -rf $SB_KEYS_DIR
rm -rf $working_dir/server_certs
rm -rf $public_gpg_key
rm -rf $STORE_ALPINE
rm -rf $GRUB_SRC
rm -rf $BOOTX_LOC

