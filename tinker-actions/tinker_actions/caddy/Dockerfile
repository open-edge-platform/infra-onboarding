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

FROM caddy:2.8.4-alpine

# Copy self signed certificate and key
COPY "caddy-key.pem" "/etc/caddy/caddy-key.pem"
COPY "caddy-cert.pem" "/etc/caddy/caddy-cert.pem"

COPY caddy_run.sh /caddy_run.sh

RUN chmod +x /caddy_run.sh

CMD ["/bin/sh", "/caddy_run.sh"]
