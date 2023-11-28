#!/bin/bash
#########################################################################
#  Script to make ready the FDO pri services until helm chart ready
#  Need to be run on Provisioner
#  It runs Manufacturing service, Owner service, Rv service
#   Author : Nabendu Maiti <nabendu.bikash.maiti@intel.com>
#########################################################################################
# INTEL CONFIDENTIAL
# Copyright (2023) Intel Corporation
#
# The source code contained or described herein and all documents related to the source
# code("Material") are owned by Intel Corporation or its suppliers or licensors. Title
# to the Material remains with Intel Corporation or its suppliers and licensors. The
# Material contains trade secrets and proprietary and confidential information of Intel
# or its suppliers and licensors. The Material is protected by worldwide copyright and
# trade secret laws and treaty provisions. No part of the Material may be used, copied,
# reproduced, modified, published, uploaded, posted, transmitted, distributed, or
# disclosed in any way without Intel's prior express written permission.
#
# No license under any patent, copyright, trade secret or other intellectual property
# right is granted to or conferred upon you by disclosure or delivery of the Materials,
# either expressly, by implication, inducement, estoppel or otherwise. Any license under
# such intellectual property rights must be express and approved by Intel in writing.
#########################################################################################

#set -x

source ../config

dockeruponly=0
if [ "$#" -eq 1 ] && [ "$1" == "nobuild" ]; then
	dockeruponly=1
fi

#Please change/provide the private interface IP
#pri_interface_ip="192.168.1.30"
ip_regex="^([0-9]{1,3}\.){3}[0-9]{1,3}$"

if ! echo "$pd_host_ip" | grep -E -q "$ip_regex"; then

	if [ "$#" -eq 1 ]; then
		if echo "$1" | grep -E -q "$ip_regex"; then
			pd_host_ip=$1
		else
			echo "Populate Config file with pd_host_ip either in config or as argument"
			exit 1
		fi
	else
		echo "Populate Config file with pd_host_ip properly "
		exit 1
	fi
fi

# container name as argument
check_container_status_retry() {
    max_attempts=5
    retry_interval=6
    attempt=1
    container_name=$1

	cmd="docker inspect --format \"{{json .State.Status}}\" $container_name"
#    echo "CMD $cmd "
	while [[ $attempt -le $max_attempts ]]; do
        health=$(eval "$cmd")
		echo "docker health $health"
        if [[  $health == '"running"' ]]; then
			echo "FDO $container_name container is : $health"
			exit 0
        else
            sleep $retry_interval
            attempt=$((attempt + 1))
        fi
    done
	echo "FDO DB container is not running" >> /home/$USER/error_log_FDO
    exit 1
}


# check docker compose v1 or v2
if command -v docker compose >/dev/null 2>&1; then
	dc='docker compose'
else
	dc='docker-compose'
fi

# Clone FDO code
# *** Modify below variables ***
if [ -z "${USER}" ]; then
	read -p "Please enter the local username: " USER
fi
export USER=$USER
export HOME=/home/$USER


cp 0001-fdo-modification.patch $HOME
cp ../tinker_workflows/manifests/prod/template_prod_bkc.yaml $HOME
cp ../tinker_workflows/manifests/prod/workflow_bkc.yaml $HOME
cp ../tinker_workflows/manifests/prod/workflow.yaml $HOME
cp ../tinker_workflows/manifests/prod/template_prod.yaml $HOME
cd $HOME
if [ -d /home/$USER/pri-fidoiot ]; then
	# Stop running containers
	cd /home/$USER/pri-fidoiot/component-samples/demo/db
	$dc down
	cd ../rv
	$dc down
	cd ../manufacturer
	$dc down
	cd ../owner
	$dc down
	# Clean up old secrets and files
	cd ../../..
	sudo rm -rf component-samples/demo/db/app-data/
	git checkout .
	sudo rm -rf component-samples/demo/db/app-data/
	git clean -df
	cp $HOME/0001-fdo-modification.patch .
	cp $HOME/template_prod_bkc.yaml  $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/workflow_bkc.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/workflow.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/template_prod.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
#	git apply 0001-fdo-modification.patch

else
	git clone -b v1.1.4 https://github.com/secure-device-onboard/pri-fidoiot.git
	cp 0001-fdo-modification.patch pri-fidoiot/
	cd pri-fidoiot
#	git apply 0001-fdo-modification.patch
	cp $HOME/template_prod_bkc.yaml  $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/workflow_bkc.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/workflow.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/template_prod.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
fi


rm -rf /home/$USER/error_log_FDO
#Build FDO code
cd build
$dc build --no-cache && $dc up

#Generate new keys for FDO services
IP_ADDRESS=$pd_host_ip
FDO_HOME=$HOME/pri-fidoiot
RV_SVC_PATH=$FDO_HOME/component-samples/demo/rv
RV_HTTPS_PORT=8041
RV_HTTP_PORT=8040
RV_SVC_URI=https://$IP_ADDRESS:$RV_HTTPS_PORT
MFG_SVC_PATH=$FDO_HOME/component-samples/demo/manufacturer
MFG_HTTPS_PORT=8038
MFG_SVC_URI=https://$IP_ADDRESS:$MFG_HTTPS_PORT
OWNER_SVC_PATH=$FDO_HOME/component-samples/demo/owner
OWNER_HTTPS_PORT=8043
OWNER_SVC_URI=https://$IP_ADDRESS:$OWNER_HTTPS_PORT
DB_SVC_PATH=$FDO_HOME/component-samples/demo/db
SCRIPT_PATH=$FDO_HOME/component-samples/demo/scripts
cd $FDO_HOME/component-samples/demo/scripts
bash demo_ca.sh
sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g web-server.conf
sed -i s/'\#\[ req_ext \]/\[ req_ext \]'/g web-server.conf
sed -i s/'\#subjectAltName/subjectAltName/g' web-server.conf
sed -i s/'\#\[ alt_names \]/\[ alt_names \]/g' web-server.conf
sed -i s/"#DNS.1 =.*$"/"DNS.1 = ${IP_ADDRESS}"/g web-server.conf
sed -i 's/#IP.1 =.*$/IP.1 = 127.0.0.1/g' web-server.conf
sed -i s/"#IP.2 =.*$"/"IP.2 = ${IP_ADDRESS}"/g web-server.conf
bash web_csr_req.sh
bash user_csr_req.sh
bash keys_gen.sh
chmod 777 -R secrets/ service.env


#Copy files to services folder
cp -r secrets/ service.env $RV_SVC_PATH
cp -r secrets/ service.env $MFG_SVC_PATH
cp -r secrets/ service.env $OWNER_SVC_PATH
cp -r secrets/ $DB_SVC_PATH

#Start DB service
echo "Starting FDO DB service"
cd $DB_SVC_PATH
sed -i s/"mariadb"/"mariadb:10.6.14"/g docker-compose.yml
if [ "$dockeruponly"  == "1" ]; then
	$dc up -d
else
    $dc build --no-cache && $dc up -d
fi


container_name='db-fdo-db-1'
ret=$(check_container_status_retry "$container_name")
if [ "$?" -ne 0 ]; then
	echo "FDO DB container is not running" >> /home/$USER/error_log_FDO
	exit 1
fi

sleep 25 # extra delay to make mariadb internal database is up
#Start RV service
echo "Starting FDO RV service"
cd $RV_SVC_PATH
sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g docker-compose.yml
sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g service.yml
if [ "$dockeruponly"  == "1" ]; then
	$dc up -d
else
    $dc build --no-cache && $dc up -d
fi

container_name='pri-fdo-rv'
ret=$(check_container_status_retry "$container_name")
if [ "$?" -ne 0 ]; then
	echo "FDO RV container is not running" >> /home/$USER/error_log_FDO
	exit 1
fi

#Start Manufacturer service
echo "Starting FDO Manufacturer service"
cd $MFG_SVC_PATH
sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g docker-compose.yml
sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g service.yml
if [ "$dockeruponly"  == "1" ]; then
	$dc up -d
else
    $dc build --no-cache && $dc up -d
fi


container_name='pri-fdo-mfg'
ret=$(check_container_status_retry "$container_name")
if [ "$?" -ne 0 ]; then
	echo "FDO Manufacturer container is not running" >> /home/$USER/error_log_FDO
	exit 1
fi


#Start owner service
echo "Starting FDO Owner service"
cd $OWNER_SVC_PATH
sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g docker-compose.yml
sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g service.yml
cp ~/.kube/config app-data/
if [ "$dockeruponly"  == "1" ]; then
	$dc up -d
else
    $dc build --no-cache && $dc up -d
fi


container_name='pri-fdo-owner'
ret=$(check_container_status_retry "$container_name")
if [ "$?" -ne 0 ]; then
	echo "FDO Owner container is not running" >> /home/$USER/error_log_FDO
	exit 1
fi

#force sleep to be the services up internally
sleep 10

# Check services health
# Configure no_proxy with SIP_ADDRESS
if [[ "$no_proxy" != *"${IP_ADDRESS}"* ]]; then
	echo "Configuring no_proxy with current $IP_ADDRESS"
	export no_proxy="$no_proxy,${IP_ADDRESS}"
fi
echo "RV service health"
cd $RV_SVC_PATH
response=$(curl -w "\nhttp_code:%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request GET ${RV_SVC_URI}/health)

http_code=$(echo $response | sed 's/^.*http_code\://g')
if [[ ${http_code} != "200" ]]; then
	echo "Failed to get the RV service version"
	exit 1
fi
echo "RV service version $response"

echo "Manufacturer service health"
cd $MFG_SVC_PATH
response=$(curl -w "\nhttp_code:%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request GET ${MFG_SVC_URI}/health)

http_code=$(echo $response | sed 's/^.*http_code\://g')
if [[ ${http_code} != "200" ]]; then
	echo "Failed to get the RV service version"
	exit 1
fi
echo "MFG service version $response"

echo "Owner service health"
cd $OWNER_SVC_PATH
response=$(curl -w "\nhttp_code:%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request GET $OWNER_SVC_URI/health)

http_code=$(echo $response | sed 's/^.*http_code\://g')
if [[ ${http_code} != "200" ]]; then
	echo "Failed to get the RV service version"
	exit 1
fi
echo "Owner service version $response"

# Add rvinfo to manufacturer
cd $MFG_SVC_PATH
response=$(curl -w "%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request POST ${MFG_SVC_URI}/api/v1/rvinfo --header 'Content-Type: text/plain' --data-raw "[[[5,\"${IP_ADDRESS}\"],[3,${RV_HTTPS_PORT}],[12,2],[2,\"${IP_ADDRESS}\"],[4,${RV_HTTPS_PORT}]]]")

if [[ ${response} != "200" ]]; then
	echo "Manufacturer rvinfo API failed ${response}"
	exit 1
fi

echo "rvinfo API is success $http_code"

# Add redirection to owner
cd $OWNER_SVC_PATH
response=$(curl -w "%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request POST ${OWNER_SVC_URI}/api/v1/owner/redirect --header 'Content-Type: text/plain' --data-raw "[[\"${IP_ADDRESS}\",\"${IP_ADDRESS}\",${OWNER_HTTPS_PORT},5]]")

if [[ ${response} != "200" ]]; then
	echo "Owner redirection API failed ${response}"
	exit 1
fi

echo "Owner redirection API is success $http_code"

# Add Owner's certificate to RV via api/v1/rv/allow endpoint to accept TO0 requests from Owner.
response=$(curl -w "%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request GET ${OWNER_SVC_URI}/api/v1/certificate?alias=SECP256R1 --header 'Content-Type: text/plain' -o owner_cert256)

if [[ ${response} != "200" ]]; then
	echo "Owner redirection API failed ${response}"
	exit 1
fi

echo "Owner get certificate API is success $http_code"
OWNER_KEY_PEM=`cat owner_cert256`

cd $RV_SVC_PATH
response=$(curl -w "%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request POST ${RV_SVC_URI}/api/v1/rv/allow --header 'Content-Type: text/plain' --data-raw "${OWNER_KEY_PEM}")

if [[ ${response} != "200" ]]; then
	echo "Allow owners key API failed ${response}"
	exit 1
fi

unset http_proxy
unset https_proxy

cd $SCRIPT_PATH
echo $IP_ADDRESS

response=$(curl --location --request POST ${MFG_SVC_URI}/api/v1/rvinfo --header 'Content-Type: text/plain' --data-raw "[[[12,2],[3,8041],[5,\"${IP_ADDRESS}\"],[4,8041],[2,\"${IP_ADDRESS}\"]]]" --cacert ../scripts/secrets/ca-cert.pem --cert ../scripts/secrets/api-user.pem -v)

echo $(curl --location --request POST ${MFG_SVC_URI}/api/v1/rvinfo --header 'Content-Type: text/plain' --data-raw "[[[12,2],[3,8041],[5,\"${IP_ADDRESS}\"],[4,8041],[2,\"${IP_ADDRESS}\"]]]" --cacert ../scripts/secrets/ca-cert.pem --cert ../scripts/secrets/api-user.pem -v)

if [[ ${response} != "200" ]]; then
        echo "Allow rvinfo ${response}"    
fi
response=$(curl --location --request POST ${OWNER_SVC_URI}/api/v1/owner/redirect --header 'Content-Type: text/plain' --data-raw "[[\"${IP_ADDRESS}\",\"${IP_ADDRESS}\",${OWNER_HTTPS_PORT},5]]" --cacert ../scripts/secrets/ca-cert.pem --cert ../scripts/secrets/api-user.pem -v -x '')

if [[ ${response} != "200" ]]; then
        echo "Allow owners redirect ${response}"
fi
echo "Allow owners key API is success $http_code"
