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

# Clone FDO code
# *** Modify below variables ***
if [ -z "${USER}" ]; then
	read -p "Please enter the local username: " USER
fi
export USER=$USER
export HOME=/home/$USER
setup_script_path=$(pwd)


cp ../tinker_workflows/manifests/prod/template_prod_bkc.yaml $HOME
cp ../tinker_workflows/manifests/prod/workflow_bkc.yaml $HOME
cp ../tinker_workflows/manifests/prod/workflow.yaml $HOME
cp ../tinker_workflows/manifests/prod/template_prod.yaml $HOME
cd $HOME
if [ -d /home/$USER/pri-fidoiot ]; then
	cd /home/$USER/pri-fidoiot/
	sudo rm -rf component-samples/demo/db/app-data/
	git checkout .
	sudo rm -rf component-samples/demo/db/app-data/
	git clean -df
	cp $HOME/template_prod_bkc.yaml  $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/workflow_bkc.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/workflow.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/template_prod.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/

else
	git clone -b v1.1.4 https://github.com/secure-device-onboard/pri-fidoiot.git
	cd pri-fidoiot
	cp $HOME/template_prod_bkc.yaml  $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/workflow_bkc.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/workflow.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
	cp $HOME/template_prod.yaml $HOME/pri-fidoiot/component-samples/demo/owner/app-data/
fi


rm -rf /home/$USER/error_log_FDO

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

SECRETS_PATH=/home/$USER/.fdo-secrets
RV_CONFIGMAP=fdo-rv-service-env
MFG_CONFIGMAP=fdo-mfg-service-env
OWNER_CONFIGMAP=fdo-owner-service-env
helm_path=$setup_script_path/../../../helm/onboarding

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



if [ -d $SECRETS_PATH ]; then
    echo "removing existing directory"
	sudo rm -rf $SECRETS_PATH
fi
mkdir -p $SECRETS_PATH
cp -r $FDO_HOME/component-samples/demo/* $SECRETS_PATH/
cp -r secrets/ $SECRETS_PATH/db/
cp -r secrets/ service.env $SECRETS_PATH/rv/
cp -r secrets/ service.env $SECRETS_PATH/manufacturer/ 
cp -r secrets/ service.env $SECRETS_PATH/owner/

sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g $SECRETS_PATH/rv/service.yml
sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g $SECRETS_PATH/manufacturer/service.yml
sed -i s/"host.docker.internal"/"${IP_ADDRESS}"/g $SECRETS_PATH/owner/service.yml


#for checking running status of pods
pod_healthcheck() {
	POD_NAME=$1
	POD_STATUS=$(kubectl get pods  | grep -E $POD_NAME | awk '{print $3}')
	echo "$POD_NAME Status is $POD_STATUS"
	if [ "$POD_STATUS" == "Running" ]; then
    exit 0
	else
		{
		echo "please check $POD_NAME Pod"
		exit 1
		}
	fi
}

#Cleaning up old helm charts & configmaps
helm uninstall fdo-db
helm uninstall fdo-mfg
helm uninstall fdo-owner
helm uninstall fdo-rv
kubectl delete cm  $MFG_CONFIGMAP $OWNER_CONFIGMAP $RV_CONFIGMAP

#changes to values.yaml files for Helm deployment
cd $helm_path
sed -i "s/<USER>/$USER/g" fdo-db/values.yaml
sed -i "s/<USER>/$USER/g" fdo-rv/values.yaml
sed -i "s/<USER>/$USER/g" fdo-mfg/values.yaml
sed -i "s/<USER>/$USER/g" fdo-owner/values.yaml

sed -i "s/^\s*internalIP:.*/  internalIP: \"$pd_host_ip\"/" fdo-db/values.yaml fdo-rv/values.yaml fdo-mfg/values.yaml fdo-owner/values.yaml


###### Start DB service #######
echo "Deploying FDO DB helm chart"
helm install fdo-db fdo-db/
sleep 25             # extra delay to make mariadb internal database is up

svc_name=fdo-db
chk=$(pod_healthcheck "$svc_name")
if [ "$?" -ne 0 ]; then
	echo "FDO DB POD is not running" >> /home/$USER/error_log_FDO
	exit 1
fi

###### Start RV service #######
kubectl create configmap $RV_CONFIGMAP --from-env-file=$SECRETS_PATH/rv/service.env
echo "Deploying FDO RV helm chart"
helm install fdo-rv fdo-rv/
sleep 15

svc_name=fdo-rv
chk=$(pod_healthcheck "$svc_name")
if [ "$?" -ne 0 ]; then
	echo "FDO RV POD is not running" >> /home/$USER/error_log_FDO
	exit 1
fi


###### Start MANUFACTURER service #######
kubectl create configmap $MFG_CONFIGMAP --from-env-file=$SECRETS_PATH/manufacturer/service.env
echo "Deploying FDO Manufacturer helm chart"
helm install fdo-mfg fdo-mfg/
sleep 15

svc_name=fdo-mfg
chk=$(pod_healthcheck "$svc_name")
if [ "$?" -ne 0 ]; then
	echo "FDO MFG POD is not running" >> /home/$USER/error_log_FDO
	exit 1
fi




#Start owner service
kubectl create configmap $OWNER_CONFIGMAP --from-env-file=$SECRETS_PATH/owner/service.env
echo "Deploying FDO Owner helm chart"
helm install fdo-owner fdo-owner/
sleep 15

svc_name=fdo-owner
chk=$(pod_healthcheck "$svc_name")
if [ "$?" -ne 0 ]; then
	echo "FDO OWNER POD is not running" >> /home/$USER/error_log_FDO
	exit 1
fi


# Check services health
# Configure no_proxy with SIP_ADDRESS
if [[ "$no_proxy" != *"${IP_ADDRESS}"* ]]; then
	echo "Configuring no_proxy with current $IP_ADDRESS"
	export no_proxy="$no_proxy,${IP_ADDRESS}"
fi
echo "RV service health"
cd $SECRETS_PATH/rv
response=$(curl -w "\nhttp_code:%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request GET ${RV_SVC_URI}/health)

http_code=$(echo $response | sed 's/^.*http_code\://g')
if [[ ${http_code} != "200" ]]; then
	echo "Failed to get the RV service version"
	exit 1
fi
echo "RV service version $response"

echo "Manufacturer service health"
cd $SECRETS_PATH/manufacturer
response=$(curl -w "\nhttp_code:%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request GET ${MFG_SVC_URI}/health)

http_code=$(echo $response | sed 's/^.*http_code\://g')
if [[ ${http_code} != "200" ]]; then
	echo "Failed to get the RV service version"
	exit 1
fi
echo "MFG service version $response"

echo "Owner service health"
cd $SECRETS_PATH/owner
response=$(curl -w "\nhttp_code:%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request GET $OWNER_SVC_URI/health)

http_code=$(echo $response | sed 's/^.*http_code\://g')
if [[ ${http_code} != "200" ]]; then
	echo "Failed to get the RV service version"
	exit 1
fi
echo "Owner service version $response"

# Add rvinfo to manufacturer
cd $SECRETS_PATH/manufacturer
response=$(curl -w "%{http_code}" --location  --cacert secrets/ca-cert.pem --cert secrets/api-user.pem -v --request POST ${MFG_SVC_URI}/api/v1/rvinfo --header 'Content-Type: text/plain' --data-raw "[[[5,\"${IP_ADDRESS}\"],[3,${RV_HTTPS_PORT}],[12,2],[2,\"${IP_ADDRESS}\"],[4,${RV_HTTPS_PORT}]]]")

if [[ ${response} != "200" ]]; then
	echo "Manufacturer rvinfo API failed ${response}"
	exit 1
fi

echo "rvinfo API is success $http_code"

# Add redirection to owner
cd $SECRETS_PATH/owner
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

cd $SECRETS_PATH/rv
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

cd $setup_script_path

# TODO Change root path of nodes agents
REPO_ROOT_PATH=$(echo "$(pwd)" | sed "s|/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service.*|/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service|")

helm install onb-mgr $REPO_ROOT_PATH/deployments/helm/onboarding-manager/ --set volumes.secret_path="$HOME/.fdo-secrets",volumes.kube_config="$HOME/.kube",env.repo_dir="$REPO_ROOT_PATH"
