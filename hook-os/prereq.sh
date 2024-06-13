#!/bin/bash

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

#set -x
source ./config
kernel_version="5.10.85-i225-igc-xz"

if [ ! -n "$harbor_url_tinker_actions" ]; then

    echo "Provide correct URL in the config file to proceed"
    exit 1
fi

VERSION_FILE=$PWD/tinker-actions/VERSION
if [ ! -f $VERSION_FILE ]; then
    if [ ! -f $PWD/TINKER_ACTIONS_VERSION ]; then
        cp $PWD/VERSION $PWD/TINKER_ACTIONS_VERSION
    fi
    VERSION_FILE=$PWD/TINKER_ACTIONS_VERSION
fi

if [ ! -f $VERSION_FILE ]; then
    echo "Fix version file, Either create it or check with the repo maintainer. $VERSION_FILE is expected"
    exit 1
fi

tag=$(cat $VERSION_FILE)

arrayof_images=($(cat hook.yaml | grep -i ".*image:.*:.*$" | awk -F: '{print $2}'))
for image in ${arrayof_images[@]}; do
    temp=$(grep -i "/" <<<$image)
    if [ $? -eq 0 ]; then
        echo "$image is excluded from harbor"
        continue
    fi
    echo "$image to be downloaded from harbor"

    # sed "s+$image+amr-registry.caas.intel.com/one-intel-edge/edgenode/tinker-actions/$image+g" patch.p
    docker pull $harbor_url_tinker_actions/$image:$tag
    if [ $? -ne 0 ]; then
        echo "unable to pull $harbor_url_tinker_action/$image:$tag"
        exit 1
    fi
    docker image tag $harbor_url_tinker_actions/$image:$tag $image:$tag
done

docker pull $harbor_url_tinker_actions/hook_dind:$tag
docker image tag $harbor_url_tinker_actions/hook_dind:$tag hook_dind:$tag

#download kernel image and tag it as expected
kernel_url=$(echo $harbor_url_tinker_actions | sed 's|/tinker-actions||')
docker pull $kernel_url/hook-kernel:$kernel_version
if [ $? -ne 0 ]; then
    echo "unable to pull $harbor_url_tinker_action/hook-kernel:$kernel_version"
    exit 1
fi

docker image tag $kernel_url/hook-kernel:$kernel_version quay.io/tinkerbell/hook-kernel:5.10.85-ea30730ea52b3f903fad7ff11a82dd12dfbdbe6c-xz
docker image rm $kernel_url/hook-kernel:$kernel_version
