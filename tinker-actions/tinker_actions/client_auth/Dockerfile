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
FROM linuxkit/getty:5d86a2ce2d890c14ab66b13638dcadf74f29218b

RUN apk add --no-cache util-linux ca-certificates curl jq

COPY client_auth.sh /

RUN chmod +x client_auth.sh

CMD ["/bin/sh", "/client_auth.sh"]
