#!/bin/sh
#####################################################################################
# INTEL CONFIDENTIAL                                                                #
# Copyright (C) 2024 Intel Corporation                                              # 
# This software and the related documents are Intel copyrighted materials,          #
# and your use of them is governed by the express license under which they          #
# were provided to you ("License"). Unless the License provides otherwise,          #
# you may not use, modify, copy, publish, distribute, disclose or transmit          #
# this software or the related documents without Intel's prior written permission.  #
# This software and the related documents are provided as is, with no express       #
# or implied warranties, other than those that are expressly stated in the License. #
#####################################################################################


token_folder=/dev/shm
client_id_path=/dev/shm/boot/client_id
client_secret_path=/dev/shm/boot/client_secret

main() {

    source /etc/hook/env_config

	echo "Keycloak URL from env: $KEYCLOAK_URL"

	update-ca-certificates

	echo "Read client_id and client_secret from files"
	client_id=$(cat $client_id_path)
	if [ -z "$client_id" ]; then
		echo "Failed to read client_id from file"
		exit 1
	fi

	client_secret=$(cat $client_secret_path)
	if [ -z "$client_secret" ]; then
		echo "Failed to read client_secret from file"
		exit 1
	fi

	#update hosts if they were provided
	extra_hosts_needed=$(printf '%s\n' "$EXTRA_HOSTS" | sed "s|,|\n|g")

	echo -e "$extra_hosts_needed" >> /etc/hosts
	echo "adding extras host mappings completed"

	echo "Fetching JWT access token from keycloak"
	access_token=$(curl -X POST https://$KEYCLOAK_URL/realms/master/protocol/openid-connect/token \
				-u "$client_id:$client_secret" \
			    -d "grant_type=client_credentials" \
			    | jq -r '.access_token')

	if [ "$access_token" = 'null' ]; then
		echo "Failed to get JWT access token from keycloak"
		exit 1
	fi

	printf "%s" "$access_token" > "$token_folder/idp_access_token"

	release_server_url=$(echo $KEYCLOAK_URL | sed "s/keycloak/release/g" )
	release_token=$(curl -X GET https://$release_server_url/token -H "Authorization: Bearer $access_token")
	if [ "$release_token" = 'null' ]; then
		echo "Failed to get release service token from release service"
		exit 1
	fi

	printf "%s" "$release_token" > "$token_folder/release_token"

	echo "Successfully provisioned JWT tokens into uOS"
	exit 0
}

#main function
main
