#!/bin/bash

# SPDX-FileCopyrightText: (C) 2024 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

#set -x
source ./config


if [ ! -n "$harbor_url_tinker_actions" ];
then

    echo "Provide correct URL in the config file to proceed"
    exit 1
fi

VERSION_FILE=$PWD/tinker-actions/VERSION
if [ ! -f $VERSION_FILE ];
then
    if [ ! -f $PWD/TINKER_ACTIONS_VERSION ] ;
    then
	cp $PWD/VERSION $PWD/TINKER_ACTIONS_VERSION
    fi
    VERSION_FILE=$PWD/TINKER_ACTIONS_VERSION
fi

if [ ! -f $VERSION_FILE ];
then
    echo "Fix version file, Either create it or check with the repo maintainer. $VERSION_FILE is expected"
    exit 1
fi

tag=$(cat $VERSION_FILE)

arrayof_images=($(cat patch.p | grep -i "+.*image:.*:.*$" | awk -F: '{print $2}'))
for image in ${arrayof_images[@]};
do
    temp=$(grep -i "/" <<< $image)
    if [ $? -eq 0 ];
    then
        echo "$image is excluded from harbor"
        continue
    fi
    echo "$image to be downloaded from harbor"

    # sed "s+$image+amr-registry.caas.intel.com/one-intel-edge/edgenode/tinker-actions/$image+g" patch.p
    docker pull $harbor_url_tinker_actions/$image:$tag
    if [ $? -ne 0 ];
    then
        echo "unable to pull $harbor_url_tinker_action/$image:$tag"
        exit 1
    fi
    docker image tag $harbor_url_tinker_actions/$image:$tag $image:$tag
done
