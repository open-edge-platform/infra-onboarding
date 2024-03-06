#!/bin/bash

if [ ! -e /dev/shm/idp_access_token ];
then
    echo "Unable to locate orchestrator access token. exiting.."
fi

if [ ! -e /dev/shm/release_token ];
then
    echo "Unable to locate release service token. exiting.."
fi

source /etc/hook/env_config

export access_token=$(cat /dev/shm/idp_access_token)
export release_token=$(cat /dev/shm/release_token)

#update hosts if they were provided
extra_hosts_needed=$(sed "s|,|\n|g" <<< "$EXTRA_HOSTS")
echo -e "$extra_hosts_needed" >> /etc/hosts
echo "adding extras host mappings completed"

envsubst < /etc/nginx/templates/nginx.conf.template > /etc/nginx/nginx.conf
nginx -g 'daemon off;'
