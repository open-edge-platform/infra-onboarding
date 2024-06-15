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

set -xuo pipefail
working_dir=$1
IPXE_DIR=$working_dir/ipxe
SB_KEYS_DIR=$working_dir/sb_keys
SERVER_CERT_DIR=$working_dir/server_certs
RSA_KEY_SIZE=4096
HASH_SIZE=512

generate_bios_certs() {
	echo "====== Generating BIOS Certificate ======="
	#verify that pk kek db is already present.
	cd "$working_dir" || exit
	if [ -d "$SB_KEYS_DIR" ] || [ -f "$SB_KEYS_DIR"/db.crt ]; then
		echo "======== Seems like Secure boot $SB_KEYS_DIR are already present. Reusing the same ========"
	else
		mkdir -p "$SB_KEYS_DIR"
		cd "$SB_KEYS_DIR" || exit

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


		echo "======== Save db.der file to enroll inside UEFI BIOS Secure Boot Settings ========="

		if [ ! -f "$SB_KEYS_DIR"/pk.crt ] || [ ! -f "$SB_KEYS_DIR"/kek.crt ] || [ ! -f "$SB_KEYS_DIR"/db.crt ] ; then
			echo "======== Seems like some issue with UEFI keys generation. Check again ========"
			cd "$working_dir" || exit
			exit 1
		fi
		cd "$working_dir" || exit
	fi
	echo "==========================================================================================="
}

generate_https_certs() {
	echo "====== Generating HTTPS Certificate ======="
	#verify that server certificates already present.
	if [ -d "$SERVER_CERT_DIR" ] && [ -f "$SERVER_CERT_DIR"/Full_server.crt ] && [ -f "$SERVER_CERT_DIR"/ca.crt ]; then
		echo "======== Seems like Full Server & CA Certificate already present. Reusing the same ========"
	else
		mkdir -p "$SERVER_CERT_DIR"
		cd "$SERVER_CERT_DIR" || exit
		if [ ! -f /etc/ssl/boots-ca-cert/ca.crt ]; then
			echo "======== file is not present ========"
			exit 0
		fi

		if [ ! -s /etc/ssl/boots-ca-cert/ca.crt ]; then
			echo "======== file size is zero ========"
			exit 0
		fi

		cp /etc/ssl/boots-ca-cert/ca.crt Full_server.crt
		cp /etc/ssl/boots-ca-cert/ca.crt ca.crt
		echo "certifictes copied from /etc/ssl/boots-ca-cert"

		# openssl  s_client -showcerts -servername $HTTPS_CN -connect $HTTPS_CN:443 </dev/null | awk '/-----BEGIN CERTIFICATE-----/,/-----END CERTIFICATE-----/' > Full_server.crt
		# openssl  s_client -showcerts -servername $HTTPS_CN -connect $HTTPS_CN:443 </dev/null | awk '/-----BEGIN CERTIFICATE-----/{flag=1; cert=""; } flag { cert = cert $0 RS } /-----END CERTIFICATE-----/{flag=0; lastCert = cert} END{printf "%s", lastCert}' > ca.crt
		#kubectl get secrets tls-maestro -n gateway-system -o yaml | grep ca.crt | sed 's/  ca.crt: //' | base64 -d > ca.crt
		#kubectl get secrets tls-maestro -n gateway-system -o yaml | grep tls.crt | sed 's/  tls.crt: //' | base64 -d > Full_server.crt
		#kubectl get secrets tls-maestro -n gateway-system -o yaml | grep tls.key | sed 's/  tls.key: //' | base64 -d > server.key
		cd "$working_dir" || exit
	fi
	echo "==========================================================================================="
}


verify_https_certs() {

	echo "======== Verifying the Signature of Full Server Certificate with CA ========="

	if [ ! -d "$SERVER_CERT_DIR" ]; then
		echo "======== 'server_certs' folder not present, Created empty folder ========"
		mkdir -p "$SERVER_CERT_DIR"
		echo "======== Please copy the Server Certificate & CA Certificate ========"
		exit 0
	fi

	if [ -d "$SERVER_CERT_DIR" ] && [ -f "$SERVER_CERT_DIR"/Full_server.crt ] && [ -f "$SERVER_CERT_DIR"/ca.crt ]; then
		echo "======== Seems like Full Server & CA Certificate already present. Reusing the same ========"
	elif [ -d "$SERVER_CERT_DIR" ] && [ ! -f "$SERVER_CERT_DIR"/Full_server.crt ] && [ ! -f "$SERVER_CERT_DIR"/ca.crt ]; then
		echo "======== Full Server & CA Certificate not present, Check again ========"
		exit 0
	fi

	#openssl verify -verbose -CAfile "$SERVER_CERT_DIR"/ca.crt "$SERVER_CERT_DIR"/Full_server.crt
	#if ! verify = $(openssl verify -verbose -CAfile "$SERVER_CERT_DIR"/ca.crt "$SERVER_CERT_DIR"/Full_server.crt); then
		#echo "====== Seems like Verification failed. Check again ======="
		#exit 1
	#fi
	echo "======== Use Full_server.crt file to enroll inside UEFI BIOS HTTPS Settings ========="

	echo "==========================================================================================="
}

build_ipxe_efi() {
	echo "======= Downloading the iPXE from GitHub repo ========"
	if [ -d "$IPXE_DIR" ]; then
		rm -rf "$IPXE_DIR"
	fi
	git clone https://github.com/ipxe/ipxe.git

	cp chain.ipxe "$IPXE_DIR"/src
	cd "$IPXE_DIR"/src || exit
	make bin-x86_64-efi/ipxe.efi >> /dev/null

	sed -i 's|//#define\tCONSOLE_FRAMEBUFFER|#define\tCONSOLE_FRAMEBUFFER|g' "$IPXE_DIR"/src/config/console.h && \
	sed -Ei "s/^#undef([ \t]*DOWNLOAD_PROTO_(HTTPS|FTP|SLAM|NFS)[ \t]*)/#define\1/" "$IPXE_DIR"/src/config/general.h && \
	sed -Ei "s/^\/\/#undef([ \t]*SANBOOT_PROTO_(ISCSI|AOE|IB_SRP|FCP|HTTP)[ \t]*)/#define\1/" "$IPXE_DIR"/src/config/general.h && \
	sed -Ei "s/^\/\/(#define[ \t]*(NSLOOKUP|TIME|DIGEST|LOTEST|VLAN|REBOOT|POWEROFF|IMAGE_TRUST|PCI|PARAM|NEIGHBOUR|PING|CONSOLE|IPSTAT|PROFSTAT|NTP|CERT)_CMD)/\1/" "$IPXE_DIR"/src/config/general.h

	if [ ! -f "$SERVER_CERT_DIR"/Full_server.crt ] || [ ! -f "$SERVER_CERT_DIR"/ca.crt ] || [ ! -f chain.ipxe ]; then
		echo "======== Seems like the certificates and/or chain script are missing. Check again ========="
		cd "$working_dir" || exit
		exit 1
	fi

	echo "======== Embedding chain script while compiling iPXE ========"
	make bin-x86_64-efi/ipxe.efi CERT="$SERVER_CERT_DIR"/Full_server.crt TRUST="$SERVER_CERT_DIR"/ca.crt EMBED=chain.ipxe

	cd "$working_dir" || exit
	echo "==========================================================================================="
}

sign_ipxe_efi() {
	echo "======== Signing iPXE image ========= "
	mkdir -p out
	
	sbsign --key "$SB_KEYS_DIR"/db.key --cert "$SB_KEYS_DIR"/db.crt --output ./out/signed_ipxe.efi "$IPXE_DIR"/src/bin-x86_64-efi/ipxe.efi
	cp "$SB_KEYS_DIR"/db.der "$working_dir"/out
	
	if [ -d "/data" ]; then
		echo "Path /data exists."
		mkdir -p /data/keys
		cp "$SB_KEYS_DIR"/db.der /data/keys
		cp "$SERVER_CERT_DIR"/Full_server.crt /data/keys
		cp "$working_dir"/out/signed_ipxe.efi /data
    else
        echo "Path /data does not exist."
    fi
      
	echo "======== Save db.der file to enroll inside UEFI BIOS Secure Boot Settings ========="
	echo "==========================================================================================="
	
}

final_artifacts() {
	echo " /**************************************************************************************/"
	echo " /**************************************************************************************/"
	echo "Signed IPXE signed_ipxe.efi is in out/"
	echo "Certificate to enroll in UEFI BIOS Secure Boot Settings is in sb_keys/db.der"
	echo "Certificate to enroll in UEFI BIOS HTTPS Settings is in server_certs/Full_server.crt"
	echo " /**************************************************************************************/"
	echo " /**************************************************************************************/"
}



echo "======= Main function to build & sign iPXE image ========"
echo "Discription of this script"
#apt install -y autoconf automake make gcc m4 git gettext autopoint pkg-config autoconf-archive python3 bison flex gawk efitools sbsigntool
generate_bios_certs
generate_https_certs
verify_https_certs
build_ipxe_efi
sign_ipxe_efi
final_artifacts
rm -rf "$IPXE_DIR"
rm -rf "$working_dir"/out
rm -rf "$working_dir"/chain.ipxe
