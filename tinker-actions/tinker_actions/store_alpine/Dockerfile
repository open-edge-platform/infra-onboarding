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
FROM debian:12.7

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    parted grub-common efibootmgr udev grub-pc-bin grub-efi-amd64-bin dosfstools wget iproute2 \
    && rm -rf /var/lib/apt/lists/*


#RUN wget https://github.com/tinkerbell/hook/releases/download/v0.8.0/hook_x86_64.tar.gz

COPY store_alpine.sh /

RUN chmod +x store_alpine.sh

COPY drive_detection.sh /
RUN chmod +x drive_detection.sh

CMD ["/bin/bash", "/store_alpine.sh"]
