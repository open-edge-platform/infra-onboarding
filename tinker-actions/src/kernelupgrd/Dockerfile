# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM alpine:3.21.3

RUN apk update && apk --no-cache add lsblk sgdisk parted e2fsprogs-extra cloud-utils-growpart util-linux eudev lvm2 bc


COPY kernel_upgrade.sh /

RUN chmod +x kernel_upgrade.sh 

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

CMD ["sh", "/kernel_upgrade.sh"]
