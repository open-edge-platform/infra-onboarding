#!/bin/bash

#########################################################################
#  Script to apply Tinkerbell configuration and workflows
#  Need to be run on Provisioner
#  It run apply di, to and product deployment workflows
#   Author : Nabendu Maiti <nabendu.bikash.maiti@intel.com>
#########################################################################

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

RED='\033[0;31m'
BCyan='\033[1;36m'
NC='\033[0m' # No Color

### following are all default value
PROVISIONER_IP=192.168.1.20
LOADBALANCER_IP=192.168.1.30
MACHINE1_IP=192.168.1.50
MACHINE1_MAC=90:49:fa:09:bd:b4
### Above are all default value

export INTERNAL_CONTROL=true

INTERNAL_CTRL=${INTERNAL_CONTROL:-true}

cleanup() {
  echo "Ctrl+C detected. Cleaning up..."
  # Send SIGTERM to all child processes in the process group
  pkill -P $$
  exit
}
trap cleanup SIGINT

do_install_wflows() {
  local worker_ip=$1
  local worker_mac=$2
  local host_ip=$3
  local loadb_ip=$4
  local namespace=$5
  local manifests_dir=$6
  local run_stage=$7
  local disk_dev=$8
  local fdo_client=$9
  local devserial=""
  local os_prtition_no=1 #default
  local os_partition="1"
  local hook_partition_no="2"

  #npart=$(echo $disk_dev | grep '.*[0-9]$')

  if [ "$run_stage" = "prod_bkc" ]; then
    os_prtition_no=3
  else
    os_prtition_no=1
  fi

  if [ $(echo $disk_dev | grep '.*[0-9]$') ]; then
    os_partition="p$os_prtition_no"
    hook_partition="p$hook_partition_no"
  else
    os_partition="$os_prtition_no"
    hook_partition="$hook_partition_no"
  fi

  unique_id=$(echo $worker_mac | tr -d ':')

  if [ "$run_stage" = "di" ] || [ "$run_stage" = "wipe_di" ]; then
    truncatedid=$(echo "$unique_id" | cut -c 7-)
    rand=$(tr -dc 'a-z0-9' </dev/urandom | fold -w 5 | head -n 1)
    devserial="$truncatedid""$rand"
    echo -e "${RED}Di Start :${BCyan} MacID:$TINKERBELL_CLIENT_MAC Serial: $devserial ${NC} "
  fi

  case $run_stage in
  "device_setup")
    filenames=("device_setup/hardware.yaml" "device_setup/template_device_setup.yaml" "device_setup/workflow.yaml")
    ;;
  "di")
    filenames=("di/hardware.yaml" "di/template_di.yaml" "di/workflow.yaml")
    ;;
  "to")
    filenames=("to/template_to.yaml" "to/workflow.yaml")
    ;;
  "prod_jammy" | "prod_focal" | "prod_bkc" | "prod_focal-ms")
    if [ "$run_stage" == "prod_bkc" ]; then
      filenames=("prod/template_prod_bkc.yaml" "prod/workflow.yaml")
      os_template_name="bkc"
    elif [ "$run_stage" == "prod_jammy" ]; then
      filenames=("prod/template_prod_jammy.yaml" "prod/workflow.yaml")
      os_template_name="jammy"
    elif [ "$run_stage" == "prod_focal-ms" ]; then
      filenames=("prod/template_prod_ms.yaml" "prod/workflow.yaml")
      os_template_name="focal-ms"
    else
      filenames=("prod/template_prod.yaml" "prod/workflow.yaml")
      os_template_name="focal"
    fi
    ;;
  *)
    echo "do_install_wflows unknown input $run_stage"
    exit 1
    ;;
  esac

  nameserver_config=''
  for i in ${nameserver[@]}; do
    nameserver_config="$nameserver_config echo 'nameserver $i' >> /etc/resolv.conf;"
  done
  nameserver_config=$(sed 's/;;/;/g' <<<"$nameserver_config")
  # echo $nameserver_config
  (
    flock -x 200

    export DISK_DEVICE="$disk_dev"
    export ROOTFS_PART_NO="$os_prtition_no"
    export ROOTFS_PARTITION="$os_partition"
    export HOOKS_PARTITION="$hook_partition"
    export TINKERBELL_CLIENT_IP="$worker_ip"
    export TINKERBELL_CLIENT_MAC="$worker_mac"
    export TINKER_CLIENT_IMG_TYPE="$image_type"
    export TINKERBELL_HOST_IP="$loadb_ip"
    export PROVISIONER_HOST_IP="$host_ip"
    export TINKERBELL_CLIENT_GW="$host_ip"
    cli_gw=$(echo "$host_ip" | cut -d'.' -f1-3)
    export TINKERBELL_CLIENT_GW_IP="$cli_gw.1"
    export TINKERBELL_DEV_SERIAL="$devserial"
    export TINKERBELL_CLIENT_UID="$unique_id"
    export TINKERBELL_CLIENT_HOST="machine-$unique_id"
    # export UNIQUE_WFLOW_NAME="workflow-$unique_id"
    export OS_TEMPLATE_NAME="$os_template_name"
    export FDO_CLIENT_TYPE="$fdo_client"
    export NAMESERVER_CONFIG="$nameserver_config"

    if [ "$os_template_name" = "bkc" ]; then
      source ./.img_variable.txt
    fi

    PRODUCT_UUID_FILE="/sys/class/dmi/id/product_uuid"
    AGENT_HARDWARE_ID=$(sudo cat "$PRODUCT_UUID_FILE" 2>/dev/null)

    if [ $? -ne 0 ]; then
        echo "Error: $PRODUCT_UUID_FILE does not exist or cannot be read."
        AGENT_HARDWARE_ID="NODE-CANNOT-READ-HARDWARE-ID"
    fi

    if [ ! -f /opt/hook/agent_node_env.txt ]; then
      #export port numbers and host_ip for the aganets to start on edge node
      echo "export MGR_HOST=$host_ip" >>agent_node_env.txt
      echo "export NO_PROXY=$host_ip" >>agent_node_env.txt
      echo "export INVMGR_PORT=31846" >>agent_node_env.txt
      echo "export UPDATEMGR_PORT=31845" >>agent_node_env.txt
      echo "export UPDATEMGR_HOST=$host_ip" >>agent_node_env.txt
      echo "export AGENT_HARDWARE_ID=$AGENT_HARDWARE_ID" >>agent_node_env.txt
      sudo cp agent_node_env.txt /opt/hook/
      rm agent_node_env.txt
    fi

    for i in "${filenames[@]}"; do
      envsubst <"$manifests_dir"/"$i"
      echo -e '---'
    done >/tmp/manifests-$TINKERBELL_CLIENT_UID.yaml
    ##  ) 200>/var/lock/lockfile_manifestcreate
  ) 200>/var/lock/lockfile_manifestcreate

  ### to remove only successful host
  wss=$(kubectl get workflow -n tink-system | awk 'FNR>1 && /^workflow/ {print $1}')
  # delete old resources workflow
  for line in $wss; do
    wf=$(echo "$line" | grep $unique_id)
    if [ "$wf" ]; then
      : $(kubectl delete -n tink-system workflow/$wf 2>&1)
    fi
  done
  wss=$(kubectl get template -n tink-system)
  if [ ! "${run_stage%"prod_"*}" = "$run_stage" ]; then
    if [ "$(echo $wss | grep fdoto-$unique_id)" = 0 ]; then
      : $(kubectl delete -n tink-system template/fdoto-$unique_id 2>&1)
    fi
  elif [ "$run_stage" == "to" ]; then
    if [ "$(echo $wss | grep fdodi-$unique_id)" = 0 ]; then
      : $(kubectl delete -n tink-system template/fdodi-$unique_id 2>&1)
    fi
  elif [ "$run_stage" == "di" ] || [ "$run_stage" == "wipe_di" ] || [ "$run_stage" == "device_setup" ]; then
    for line in $wss; do
      if echo "$line" | grep -q $unique_id; then
        : $(kubectl delete -n tink-system template/$line 2>&1)
      fi
    done
  else
    echo "unknown input $run_stage"
    exit 1
  fi

  # Apply workflows

  : $(kubectl apply -n "$namespace" -f /tmp/manifests-$unique_id.yaml 2>&1)

  echo "Workflow Applied"
  #rm /tmp/manifests-$unique_id.yamls
}

function usage() {
  echo "======USAGE=====:"
  echo -e "\n ./install_manifests.sh \n  OR  \n ./install_manifeststs.sh  <Host_ip> <load_balancer_ip> <Sut_ip> <Sut_mac> <run_stage(optional) di/to/prod> <disk(opt)> <fdo_clent_type>"
  exit 0
}

main() {

  # local loadbalancer_interface=$7
  local namespace="tink-system"
  local manifests_dir="./manifests"

  local arglen=$#
  if [ $arglen -eq 0 ]; then
    ./apply_manifests.sh "$MACHINE1_IP" "$MACHINE1_MAC" "$PROVISIONER_IP" "$LOADBALANCER_IP" "$namespace" "$manifests_dir" "device_setup" "/dev/nvme0n1" "CLIENT_SDK"
  elif [ $arglen -ge 4 ] && [ $arglen -le 7 ]; then
    local host_ip=$1
    local loadbalancer_ip=$2
    local sut_ip=$3
    local sut_mac=$4

    if [ $arglen -ge 5 ]; then
      run_stage="$5"
    else
      run_stage="di" ##di stage
    fi

    if [ $arglen -ge 6 ]; then
      disk="$6"
    else
      disk="/dev/nvme0n1" ##di stage
    fi

    if [ $arglen -ge 7 ]; then
      fdo_client="$7"
    else
      fdo_client="CLIENT_SDK" ##di stage
    fi
    echo $INTERNAL_CTRL

    if [ "$INTERNAL_CTRL" = true ] && [ ! "${run_stage%"prod_"*}" = "$run_stage" ]; then
      ./dl_images.sh "$run_stage" &
      wait
    fi

    do_install_wflows "$sut_ip" "$sut_mac" "$host_ip" "$loadbalancer_ip" "$namespace" "$manifests_dir" "$run_stage" "$disk" "$fdo_client"
    if [ $? = 1 ]; then
      exit 1
    fi
  else
    echo "Unknown parameters"
    usage
  fi
}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
#  set -x #euxo pipefail
  main "$@"
  echo "all done!"
fi
