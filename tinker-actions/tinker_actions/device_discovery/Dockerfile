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

# Start from the getty:v1.0.0 as base
FROM linuxkit/getty:v1.0.0

# Install dmidecode
RUN apk update && apk add --no-cache dmidecode util-linux ca-certificates curl jq

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the local system to the Working Directory inside the container
COPY app . 
COPY client-auth.sh .

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

# Command to run the executable
CMD ["./app"]
