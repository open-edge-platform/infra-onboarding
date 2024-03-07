#!/bin/bash
# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel


source ../../config

download_certs() {
    files_location=$PWD/../files
    certs_folder=$PWD/certs

    mkdir -p $PWD/certs
    if [ $keycloak_url == '' ];
    then
      echo "keycloak URL not configured. Hence assuming that files will be present in hook/files/idp folder"
      return
    fi

    openssl s_client -showcerts -connect $keycloak_url:443 </dev/null |
    awk '/BEGIN CERTIFICATE/,/END CERTIFICATE/{ if(/BEGIN CERTIFICATE/){a++}; out="certs/cert"a".pem"; print >out}'

    if [ -f $files_location/server_cert.pem ];
    then
        echo "Using the server certificate which is already present in $files_location/server_cert.pem"
    else
        cp $PWD/certs/cert1.pem $files_location/server_cert.pem
        if [ $? -ne 0 ];
        then
            echo "Failed to copy the server cert"
        fi
    fi

    certfiles=($(ls -r $certs_folder))

    if [ -f $files_location/ca.pem ];
    then
        echo "Using the ca certificate which is already present in $files_location/ca.pem"
    else
        cp $certs_folder/${certfiles[0]} $files_location/ca.pem

        if [ $? -ne 0 ];
        then
            echo "Failed to copy the ca.pem cert"
        fi
    fi
}

echo "in get_certs.sh"
download_certs
echo "done get_certs.sh"
