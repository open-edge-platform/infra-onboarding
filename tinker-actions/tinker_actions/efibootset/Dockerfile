# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

FROM debian:12.7

RUN apt update && apt install -y parted grub-common udev grub-pc-bin grub-efi-amd64-bin dosfstools wget iproute2

COPY efibootset.sh /

RUN chmod +x efibootset.sh

COPY drive_detection.sh /
RUN chmod +x drive_detection.sh

COPY change_boot_order.sh /
RUN chmod +x change_boot_order.sh


# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:50054/healthz || exit 1

CMD ["/bin/bash", "/efibootset.sh"]
