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


pipe='/tmp/pipetmp'
mkfifo "$pipe"

finished_read='false'
a=0
password_authenticated=0
idp_folder=/dev/shm

enable_tty0() {

    echo "Provide Username and password for the IDP"
    setsid bash -c "echo 'Provide Username and password for the IDP' <> /dev/tty0 >&0 2>&1" 
    setsid bash -c 'read -p "Username: " username <> /dev/tty0 >&0 2>&1 && [[ ! -z "$username" ]] && echo $username > /idp_username'
    setsid bash -c 'read -s -p "Password: " password <> /dev/tty0 >&0 2>&1 && [[ ! -z "$password" ]] && echo $password > /idp_password'
    setsid bash -c "echo -e '\nUsername, Password received: Processing' <> /dev/tty0 >&0 2>&1"

    finished_read='True'
    echo "$finished_read" > "$pipe"
}

enable_ttyS0() {
    setsid -w /usr/sbin/getty -a root -L 115200 $tty vt100 &
    echo "Provide Username and password for the IDP"
    setsid bash -c "echo 'Provide Username and password for the IDP' <> /dev/ttyS0 >&0 2>&1" 
    setsid bash -c 'read -p "Username: " username <> /dev/ttyS0 >&0 2>&1 && [[ ! -z "$username" ]] && echo $username > /idp_username' 
    setsid bash -c 'read -s -p "Password: " password <> /dev/ttyS0 >&0 2>&1 && [[ ! -z "$password" ]] && echo $password > /idp_password' 
    
    setsid bash -c "echo -e '\nUsername, Password received: Processing' <> /dev/ttyS0 >&0 2>&1"

    finished_read='True'
    echo "$finished_read" > "$pipe"
}

enable_ttyS1() {
    echo "Provide Username and password for the IDP"
    setsid bash -c "echo 'Provide Username and password for the IDP' <> /dev/ttyS1 >&0 2>&1"
    setsid bash -c 'read -p "Username: " username <> /dev/ttyS1 >&0 2>&1 && [[ ! -z "$username" ]] && echo $username > /idp_username'
    setsid bash -c 'read -s -p "Password: " password <> /dev/ttyS1 >&0 2>&1 && [[ ! -z "$password" ]] && echo $password > /idp_password'

    setsid bash -c "echo -e '\nUsername, Password received: Processing' <> /dev/ttyS1 >&0 2>&1"

    finished_read='True'
    echo "$finished_read" > "$pipe"
}


main() {

    source /etc/hook/env_config

    while [ $a -lt 3 ];
    do
	finished_read='False'
	a=`expr $a + 1`

	enable_ttyS0 &
	enable_tty0 &
	enable_ttyS1 &

	check=0
	while [ ${finished_read} != 'True' ]
	do
	    sleep 5
	    finished_read=$(cat "$pipe")
	    echo "${finished_read}"
	    check=`expr $check + 1`
	    if [ $check -gt 10 ];
	    then
		break
	    fi
	done

	username=$(cat /idp_username)
	password=$(cat /idp_password)

	username=$(tr -d " " <<< $username | tr -d "\n" | tr -d ";")
	password=$(tr -d " " <<< $password | tr -d "\n" | tr -d ";")

	#read the single line IDP_certificate from the /proc/cmdline  awk 'NF {sub(/\r/, ""); printf "%s\\n",$0;}' test2

	#IDP_CERTIFICATE=$(grep -oi "\-\-\-\-\-BEGIN CERTIFICATE-----.*-----END CERTIFICATE-----" /proc/cmdline)
	# IDP_CERTIFICATE=$(grep -oi "\-\-\-\-\-BEGIN CERTIFICATE-----.*-----END CERTIFICATE-----" test)

	#unroll to a clean PEM file that can be used by IDP curl cmd.
	# echo -ne $IDP_CERTIFICATE > ca.pem

	# add to trust pool
	#cp $idp_folder/ca.pem /usr/local/share/ca-certificates/IDP_keyclock.crt
	if [ ! -e /usr/local/share/ca-certificates/IDP_keyclock.crt ];
	then
	    echo " IDP ca cert not found at the expected location: reboot"
	    sleep 3
	    reboot
	fi

	update-ca-certificates

	#update hosts if they were provided
	extra_hosts_needed=$(sed "s|,|\n|g" <<< "$EXTRA_HOSTS")
	echo -e "$extra_hosts_needed" >> /etc/hosts
	echo "adding extras completed"

	#login to IDP keycloak
	# proxy if not set then the code will not be able to invoke curl.

	access_token=$(curl -X POST https://$KEYCLOAK_URL/realms/master/protocol/openid-connect/token \
			    -d "username=$username" \
			    -d "password=$password" \
			    -d "grant_type=password" \
			    -d "client_id=ledge-park-system" \
			    -d "scope=openid" | jq -r '.access_token')

	if [[ $access_token == 'null' ]];
	then
	    echo "Error login - retry"
	    continue
	else
	    password_authenticated=1
	    break
	fi


    done


    if [ $password_authenticated -ne 0 ];
    then
	# mkdir -p $idp_folder
	printf "%s" "$access_token" > "$idp_folder/idp_access_token"

	release_server_url=$(sed "s/keycloak/release/g" <<< $KEYCLOAK_URL)
	release_token=$(curl -X GET https://$release_server_url/token -H "Authorization: Bearer $access_token")
	printf "%s" "$release_token" > "$idp_folder/release_token"

    else
	echo "Incorrect username and password provided: rebooting now"
	sleep 5
	reboot
    fi
}

#main function
main
