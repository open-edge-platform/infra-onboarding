# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM alpine:3.21.3 

RUN apk --no-cache add efibootmgr grub-efi

COPY efibootset.sh /

RUN chmod +x efibootset.sh

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

CMD ["sh", "/efibootset.sh"]
