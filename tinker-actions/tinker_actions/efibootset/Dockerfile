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
FROM debian:12

RUN apt update
RUN apt install -y parted grub-common udev grub-pc-bin grub-efi-amd64-bin dosfstools wget iproute2

COPY efibootset.sh /

RUN chmod +x efibootset.sh

COPY drive_detection.sh /
RUN chmod +x drive_detection.sh

COPY change_boot_order.sh /
RUN chmod +x change_boot_order.sh

CMD ["/bin/bash", "/efibootset.sh"]
