# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

# Start from the getty as base
FROM linuxkit/getty:05eca453695984a69617f1f1f0bcdae7f7032967
# Install dmidecode
RUN apk update && apk upgrade openssl && apk add --no-cache dmidecode util-linux ca-certificates curl jq

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the Pre-built binary file from the local system to the Working Directory inside the container
COPY device-discovery . 
COPY client-auth.sh .

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

# Command to run the executable
CMD ["./device-discovery"]
