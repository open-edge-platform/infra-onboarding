# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM alpine:3.21.3

RUN apk --no-cache add lsblk

COPY eject_all_removable_disks.sh /

RUN chmod +x eject_all_removable_disks.sh


COPY erase_non_removable_disks.sh /

RUN chmod +x erase_non_removable_disks.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

CMD ["sh", "/erase_non_removable_disks.sh"]
