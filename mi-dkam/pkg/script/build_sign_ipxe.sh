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

IPXE_DIR=$PWD/ipxe
SB_KEYS_DIR=$PWD/sb_keys
SERVER_CERT_DIR=$PWD/server_certs
BIOS_CN=GA
O=INTEL
OU=NEX
C=IN

generate_bios_certs() {
	echo "====== Generating BIOS Certificate ======="
	#verify that pk kek db is already present.
	if [ -d $SB_KEYS_DIR ] || [ -f $SB_KEYS_DIR/db_ipxe.crt ]; then
		echo "======== Seems like Secure boot $SB_KEYS_DIR are already present. Reusing the same ========"
	else
		mkdir -p $SB_KEYS_DIR
		pushd $SB_KEYS_DIR

		openssl req -x509 -newkey rsa:4096 -keyout db_ipxe.key -out db_ipxe.crt -days 1000 -nodes -subj "/CN=4c4c4544-0035-3010-8030-c2c04f4a4633" -addext "subjectAltName = DNS:4c4c4544-0035-3010-8030-c2c04f4a4633"
		if [ ! -f $SB_KEYS_DIR/db_ipxe.key] || [ ! -f $SB_KEYS_DIR/db_ipxe.crt ] ; then
			echo "======== Seems like some issue with UEFI keys generation. Check again ========"
			popd
			exit 1
		fi
		popd
	fi
	echo "==========================================================================================="
}


generate_https_certs() {
	echo "====== Generating HTTPS Certificate ======="
	#verify that server certificates already present.
	if [ -d $SERVER_CERT_DIR ] && [ -f $SERVER_CERT_DIR/Full_server.crt ] && [ -f $SERVER_CERT_DIR/CA.crt ]; then
		echo "======== Seems like Full Server & CA Certificate already present. Reusing the same ========"
	else
		mkdir -p $SERVER_CERT_DIR
		pushd $SERVER_CERT_DIR
		openssl  s_client -showcerts -servername keycloak.demo2.maestro.intel.com -connect keycloak.demo2.maestro.intel.com:443 </dev/null | awk '/-----BEGIN CERTIFICATE-----/,/-----END CERTIFICATE-----/' > Full_server.crt
		openssl  s_client -showcerts -servername keycloak.demo2.maestro.intel.com -connect keycloak.demo2.maestro.intel.com:443 </dev/null | awk '/-----BEGIN CERTIFICATE-----/{flag=1; cert=""; } flag { cert = cert $0 RS } /-----END CERTIFICATE-----/{flag=0; lastCert = cert} END{printf "%s", lastCert}' > CA.crt
		popd
	fi
	echo "==========================================================================================="
}

build_ipxe_efi() {
	echo "======= Downloading the iPXE from GitHub repo ========"
	if [ -d $IPXE_DIR ]; then
		rm -rf $IPXE_DIR
	fi
	git clone https://github.com/ipxe/ipxe.git

	cp chain.ipxe $IPXE_DIR/src
	pushd $IPXE_DIR/src
	make bin-x86_64-efi/ipxe.efi

	sed -i 's|//#define\tCONSOLE_FRAMEBUFFER|#define\tCONSOLE_FRAMEBUFFER|g' $IPXE_DIR/src/config/console.h && \
	sed -Ei "s/^#undef([ \t]*DOWNLOAD_PROTO_(HTTPS|FTP|SLAM|NFS)[ \t]*)/#define\1/" $IPXE_DIR/src/config/general.h && \
	sed -Ei "s/^\/\/#undef([ \t]*SANBOOT_PROTO_(ISCSI|AOE|IB_SRP|FCP|HTTP)[ \t]*)/#define\1/" $IPXE_DIR/src/config/general.h && \
	sed -Ei "s/^\/\/(#define[ \t]*(NSLOOKUP|TIME|DIGEST|LOTEST|VLAN|REBOOT|POWEROFF|IMAGE_TRUST|PCI|PARAM|NEIGHBOUR|PING|CONSOLE|IPSTAT|PROFSTAT|NTP|CERT)_CMD)/\1/" $IPXE_DIR/src/config/general.h

	if [ ! -f $SERVER_CERT_DIR/Full_server.crt ] || [ ! -f $SERVER_CERT_DIR/CA.crt ] || [ ! -f chain.ipxe ]; then
		echo "======== Seems like the certificates and/or chain script are missing. Check again ========="
		popd
		exit 1
	fi

	echo "======== Embedding chain script while compiling iPXE ========"
	make bin-x86_64-efi/ipxe.efi CERT=$SERVER_CERT_DIR/Full_server.crt TRUST=$SERVER_CERT_DIR/CA.crt EMBED=chain.ipxe

	popd
	echo "==========================================================================================="
}

sign_ipxe_efi() {
	echo "======== Signing iPXE image ========= "
	mkdir -p out
	sbsign --key $SB_KEYS_DIR/db_ipxe.key --cert $SB_KEYS_DIR/db_ipxe.crt --output ./out/signed_ipxe.efi $IPXE_DIR/src/bin-x86_64-efi/ipxe.efi
	cp $SB_KEYS_DIR/db_ipxe.der $PWD/out/.
	echo "======== Save db_ipxe.der file to enroll inside UEFI BIOS Secure Boot Settings ========="
	echo "==========================================================================================="
}


echo "======= Main function to build & sign iPXE image ========"
echo "Discription of this script"
# echo "Arg #1 should be Self Signed Server Certificate"
# echo "Arg #2 should be Server Private Key"
# echo "Example : ./build_sign_ipxe.sh Server.crt Server.key"

	# if [ $# -eq 0 ]; then
	# 	echo "$0: Missing arguments"
	# 	exit 1
	# elif [ $# -lt 2 ]; then
	# 	echo "$0: Too few arguments"
	# 	exit 1
	# elif [ $# -gt 2 ]; then
	# 	echo "$0: Too many arguments"
	# 	exit 1
	# else
	# 	echo "==========================="
	# 	echo "Arg #1..............: $1"
	# 	echo "Arg #2..............: $2"
	# 	echo "==========================="
	# fi

	# if [[ $1 =~ .*\.(crt$) ]] ; then
	# 	echo "Arg #1 passed is a Certificate"
	# else
	# 	echo "Arg #1 passed is not a Certificate"
	# 	exit 1
	# fi

	# if [[ $2 =~ .*\.(key$) ]] ; then
	# 	echo "Arg #2 passed is a Key"
	# else
	# 	echo "Arg #2 passed is not a Key"
	# 	exit 1
	# fi

	# cat $1 > Server.crt
	# cat $2 > Server.key

	generate_bios_certs
	generate_https_certs
	build_ipxe_efi
	sign_ipxe_efi
