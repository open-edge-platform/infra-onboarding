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

RUN apt-get update
RUN apt-get install -y parted cryptsetup-bin tpm2-tools xxd
RUN apt-get install -y lvm2 procps

RUN apt-get install -y file fdisk 

COPY enable_fde.sh /
COPY enable_fde_tiber.sh /
COPY os_selection.sh /

RUN chmod +x enable_fde.sh
RUN chmod +x enable_fde_tiber.sh
RUN chmod +x os_selection.sh

#CMD ["/bin/bash", "/enable_fde_tiber.sh"]
CMD ["/bin/bash", "/os_selection.sh"]
