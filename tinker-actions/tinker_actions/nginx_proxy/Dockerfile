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

FROM nginx:1.25.3

# Copy self signed certificate and key
COPY "nginx-key.pem" "/etc/nginx/ssl/nginx-key.pem"
COPY "nginx-cert.pem" "/etc/nginx/ssl/nginx-cert.pem"

COPY nginx_run.sh /nginx_run.sh

RUN chmod +x /nginx_run.sh

#CMD ["bash", "-c", "export access_token=$(cat /dev/shm/idp_access_token) && export release_token=$(cat /dev/shm/release_token) && envsubst < /etc/nginx/templates/nginx.conf.template > /etc/nginx/nginx.conf && nginx -g 'daemon off;'"]
CMD ["bash", "-c", "/nginx_run.sh"]
