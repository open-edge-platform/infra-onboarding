# SPDX-FileCopyrightText: 2024 Intel Corporation
#
# SPDX-License-Identifier: LicenseRef-Intel

# Use an official Ubuntu base image
FROM debian:bullseye-slim AS fluent-bit-uos

# Set environment variables to non-interactive (this prevents some prompts)
ENV DEBIAN_FRONTEND=noninteractive

#Install requierments
RUN apt-get update && \
    apt-get install -y curl gnupg software-properties-common 

# Install Fluent Bit
RUN curl https://packages.fluentbit.io/fluentbit.key | gpg --dearmor > /usr/share/keyrings/fluentbit-keyring.gpg && \
    echo 'deb [signed-by=/usr/share/keyrings/fluentbit-keyring.gpg] https://packages.fluentbit.io/ubuntu/focal focal main' | tee  /etc/apt/sources.list.d/fluentbit.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
    fluent-bit=3.0.0 && \
    apt purge -y curl && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

COPY fluentbit_run.sh /fluentbit_run.sh

RUN chmod +x /fluentbit_run.sh

ENTRYPOINT ["bash", "-c", "/fluentbit_run.sh"]

