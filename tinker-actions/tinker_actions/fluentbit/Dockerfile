# SPDX-FileCopyrightText: 2024 Intel Corporation
#
# SPDX-License-Identifier: LicenseRef-Intel

# Use an official Ubuntu base image
FROM fluent/fluent-bit:3.1.10 AS fluent-bit-uos

COPY --from=busybox:stable-uclibc /bin/sh /bin/sh
COPY --from=busybox:stable-uclibc /bin/cat /bin/cat

COPY fluentbit_run.sh /fluentbit_run.sh

ENTRYPOINT ["/bin/sh", "-c", "/fluentbit_run.sh"]

