#!/bin/sh

# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

pipe='/tmp/pipetmp'
mkfifo "$pipe"

finished_read='false'
a=0
password_authenticated=0

readonly idp_folder=/dev/shm
readonly log_file="/var/log/onboot/client-auth.log"
readonly HTTP_OK=200
readonly HTTP_NO_CONTENT=204

# Ensure the log directory exists
mkdir -p /var/log/onboot
touch "$log_file"

# shellcheck disable=SC2016
enable_tty0() {
    echo 'False' > /tty0_status_user
    echo 'False' > /tty0_status_pass
    echo "Provide Username and password for the IDP" >> "$log_file"

	setsid /bin/sh -c "echo -e '\nProvide Username and password for the IDP' <> /dev/tty0 >&0 2>&1"
    setsid /bin/sh -c 'read -p "Username: " username <> /dev/tty0 >&0 2>&1 && [ ! -z "$username" ] && echo $username > /idp_username && echo "True" > /tty0_status_user'
    setsid /bin/sh -c 'read -s -p "Password: " password <> /dev/tty0 >&0 2>&1 && [ ! -z "$password" ] && echo $password > /idp_password  && echo "True" > /tty0_status_pass'
    setsid /bin/sh -c "echo -e '\nUsername, Password received: Processing' <> /dev/tty0 >&0 2>&1"

    userread=$(cat /tty0_status_user)
    passread=$(cat /tty0_status_pass)
    if [ "${userread}" = 'True' ] && [ "${passread}" = 'True' ];
    then
	finished_read='True'
	echo "$finished_read" > "$pipe"
	echo "tty0: Username and password received" >> "$log_file"
	echo 'False' > /tty0_status_user
	echo 'False' > /tty0_status_pass
    fi
}

# shellcheck disable=SC2016
enable_ttyS0() {
    echo 'False' > /ttys0_status_user
    echo 'False' > /ttys0_status_pass

    echo "Provide Username and password for the IDP" >> "$log_file"
	setsid /bin/sh -c "echo -e '\nProvide Username and password for the IDP' <> /dev/ttyS0 >&0 2>&1"
    setsid /bin/sh -c 'read -p "Username: " username <> /dev/ttyS0 >&0 2>&1 && [ ! -z "$username" ] && echo $username > /idp_username && echo "True" > /ttys0_status_user'
    setsid /bin/sh -c 'read -s -p "Password: " password <> /dev/ttyS0 >&0 2>&1 && [ ! -z "$password" ] && echo $password > /idp_password  && echo "True" > /ttys0_status_pass'
    setsid /bin/sh -c "echo -e '\nUsername, Password received: Processing' <> /dev/ttyS0 >&0 2>&1"

    userread=$(cat /ttys0_status_user)
    passread=$(cat /ttys0_status_pass)

    if [ "${userread}" = 'True' ] && [ "${passread}" = 'True' ];
    then
	finished_read='True'

	echo "$finished_read" > "$pipe"
	echo "ttyS0: Username and password received" >> "$log_file"
	echo 'False' > /ttys0_status_user
	echo 'False' > /ttys0_status_pass
    fi
    setsid /bin/sh -c "echo 'here-3' <> /dev/ttyS0 >&0 2>&1"
}

# shellcheck disable=SC2016
enable_ttyS1() {
    echo 'False' > /ttys1_status_user
    echo 'False' > /ttys1_status_pass
    echo "Provide Username and password for the IDP" >> "$log_file"
	setsid /bin/sh -c "echo -e '\nProvide Username and password for the IDP' <> /dev/ttyS1 >&0 2>&1"
    setsid /bin/sh -c 'read -p "Username: " username <> /dev/ttyS1 >&0 2>&1 && [ ! -z "$username" ] && echo $username > /idp_username && echo "True" > /ttys1_status_user'
    setsid /bin/sh -c 'read -s -p "Password: " password <> /dev/ttyS1 >&0 2>&1 && [ ! -z "$password" ] && echo $password > /idp_password  && echo "True" > /ttys1_status_pass'
    setsid /bin/sh -c "echo -e '\nUsername, Password received: Processing' <> /dev/ttyS1 >&0 2>&1"

    userread=$(cat /ttys1_status_user)
    passread=$(cat /ttys1_status_pass)
    if [ "${userread}" = 'True' ] && [ "${passread}" = 'True' ];
    then
	finished_read='True'
	echo "$finished_read" > "$pipe"
	echo "ttyS1: Username and password received" >> "$log_file"
	echo 'False' > /ttys1_status_user
	echo 'False' > /ttys1_status_pass
    fi
}


main() {

    # shellcheck source=/dev/null
    . /etc/hook/env_config

    while [ $a -lt 3 ];
    do
	finished_read='False'
	a=$((a + 1))
	echo "Attempt $a to read username and password" >> "$log_file"

	enable_ttyS0 &
	enable_tty0 &
	enable_ttyS1 &

	check=0
	while [ "${finished_read}" != 'True' ]
	do
	    sleep 5
	    finished_read=$(cat "$pipe")
	    echo "${finished_read}"
		check=$((check + 1))
	    if [ "$check" -gt 10 ];
	    then
		break
	    fi
	done

	username=$(cat /idp_username)
	password=$(cat /idp_password)

	username=$(echo "$username" | tr -d " "  | tr -d "\n" | tr -d ";")
	password=$(echo "$password" | tr -d " "  | tr -d "\n" | tr -d ";")

	#username and password checks are done at keycloak this is just to ensure that there was some valid input received
	# shellcheck disable=SC3037,SC2086
	if [ "$(echo -n $username | wc -c)" -lt 3 ] || [ "$(echo -n $password | wc -c)" -lt 3 ]; then
		echo "Incorrect username password" >> "$log_file"
		continue
	fi

	if [ ! -e /usr/local/share/ca-certificates/IDP_keyclock.crt ];
	then
	    echo " IDP ca cert not found at the expected location: reboot" >> "$log_file"
	    sleep 3
	    reboot
	fi

	update-ca-certificates

	#update hosts if they were provided
	extra_hosts_needed=$(echo "$EXTRA_HOSTS" | sed "s|,|\n|g")
	echo "$extra_hosts_needed" >> /etc/hosts
	echo "adding extras completed" >> "$log_file"

	#login to IDP keycloak
	# proxy if not set then the code will not be able to invoke curl.

	access_token=$(curl --cacert /usr/local/share/ca-certificates/IDP_keyclock.crt -X POST https://"$KEYCLOAK_URL"/realms/master/protocol/openid-connect/token \
			    -d "username=$username" \
			    -d "password=$password" \
			    -d "grant_type=password" \
			    -d "client_id=system-client" \
			    -d "scope=openid" | jq -r '.access_token')

	if [ "$access_token" = 'null' ]; then
	    echo "Error login - retry" >> "$log_file"
	    continue
	else
	    password_authenticated=1
	    break
	fi


    done


    if [ $password_authenticated -ne 0 ]; then
	
		printf "%s" "$access_token" > "$idp_folder/idp_access_token"
		echo "Authentication successful, IDP Access token saved" >> "$log_file"

		release_server_url=$(echo "$KEYCLOAK_URL" | sed "s/keycloak/release/g" )

		response_body=$(mktemp)

		status_code=$(curl --cacert /usr/local/share/ca-certificates/IDP_keyclock.crt -X GET "https://$release_server_url/token" -H "Authorization: Bearer $access_token" -w "%{http_code}" -o "$response_body")
		release_token=$(cat "$response_body")

		rm "$response_body"

		# Check for status code 200
		if [ "$status_code" -eq $HTTP_OK ]; then
			echo "Successfully retrieved release token."
			printf "%s" "$release_token" > "$idp_folder/release_token"
		# Check for status code 204
		elif [ "$status_code" -eq $HTTP_NO_CONTENT ]; then
			echo "No release token exists. Creating an empty file."
			: > "$idp_folder/release_token"
		else
			echo "Failed to retrieve release token. HTTP status code: $status_code"
			exit 1
		fi
    else
		echo "Incorrect username and password provided." >> "$log_file"
		setsid /bin/sh -c "echo -e '\nIncorrect username and password provided.' <> /dev/ttyS1 >&0 2>&1"
		setsid /bin/sh -c "echo -e '\nIncorrect username and password provided.' <> /dev/ttyS0 >&0 2>&1"
		setsid /bin/sh -c "echo -e '\nIncorrect username and password provided.' <> /dev/tty0 >&0 2>&1"
		sleep 5
    fi
}

#main function
main
