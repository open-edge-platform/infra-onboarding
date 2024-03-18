# INTEL CONFIDENTIAL
# Copyright (2023) Intel Corporation
#
# The source code contained or described herein and all documents related to the source
# code("Material") are owned by Intel Corporation or its suppliers or licensors. Title
# to the Material remains with Intel Corporation or its suppliers and licensors. The
# Material contains trade secrets and proprietary and confidential information of Intel
# or its suppliers and licensors. The Material is protected by worldwide copyright and
# trade secret laws and treaty provisions. No part of the Material may be used, copied,
# reproduced, modified, published, uploaded, posted, transmitted, distributed, or
# disclosed in any way without Intel's prior express written permission.
#
# No license under any patent, copyright, trade secret or other intellectual property
# right is granted to or conferred upon you by disclosure or delivery of the Materials,
# either expressly, by implication, inducement, estoppel or otherwise. Any license under
# such intellectual property rights must be express and approved by Intel in writing.

################################################################################
FROM ubuntu:22.04 as fdo-client-sdk-builder

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update -y && apt-get upgrade -y && apt-get install -y \
  git \
  build-essential \
  cmake \
  libxml2-dev \
  uuid-dev \
  libcurl4-openssl-dev \
  openssl \
  libssl-dev \
  wget \
  bsdmainutils \
  unzip \
  libglib2.0-dev \
  xxd \
  nghttp2 \
  libnghttp2-dev

RUN mkdir build_dir

# Build & Install Curl version 8.4.0
WORKDIR /build_dir/
RUN wget https://curl.se/download/curl-8.4.0.tar.gz -O - | tar -xz && cd curl-8.4.0 && \
./configure --with-openssl --enable-versioned-symbols --with-nghttp2 && make && make install

# Setup safestring
WORKDIR /build_dir
RUN git clone -b v1.2.0 --progress --verbose https://github.com/intel/safestringlib.git safestring
ENV SAFESTRING_ROOT /build_dir/safestring
WORKDIR ${SAFESTRING_ROOT}
RUN mkdir obj && make

# Setup tinycbor
WORKDIR /build_dir
RUN git clone -b v0.6.0 --progress --verbose https://github.com/intel/tinycbor.git tinycbor
ENV TINYCBOR_ROOT /build_dir/tinycbor
WORKDIR ${TINYCBOR_ROOT}
RUN make

# Setup metee
WORKDIR /build_dir
RUN git clone --progress --verbose https://github.com/intel/metee.git metee
ENV METEE_ROOT /build_dir/metee
WORKDIR ${METEE_ROOT}
RUN cmake . && make && make install

ENV OPENSSL3_ROOT /usr
ENV CURL_ROOT /usr

# Setup client-sdk-cse.
WORKDIR /build_dir/
RUN mkdir /cse-fdoout
WORKDIR /build_dir/cse-fdoout/
RUN git clone -b v1.1.7 https://github.com/fido-device-onboard/client-sdk-fidoiot.git client-sdk
WORKDIR /build_dir/cse-fdoout/client-sdk/
RUN cmake -DDA=cse_ecdsa384 -DCSE_SHUTDOWN=false -DCSE_CLEAR=true .
RUN make -j$(nproc)

# Setup client-sdk and TPM libraries.
WORKDIR /build_dir/
RUN mkdir /tpm-fdoout
WORKDIR /build_dir/tpm-fdoout/
RUN git clone -b v1.1.7 https://github.com/fido-device-onboard/client-sdk-fidoiot.git client-sdk
WORKDIR /build_dir/tpm-fdoout/client-sdk/
RUN grep -rl '/opt/openssl' ./ | xargs sed -i 's/\/opt\/openssl/\/usr/g'
RUN grep -rl '/opt/curl' ./ | xargs sed -i 's/\/opt\/curl/\/usr/g'
RUN sed -i 's/lib64/lib\/x86_64-linux-gnu/' utils/install_tpm_libs.sh
RUN sed -i 's/==/-eq/' utils/tpm_make_ready_ecdsa.sh
RUN sh utils/install_tpm_libs.sh -i
RUN cmake -DDA=tpm20_ecdsa256 -DTPM2_TCTI_TYPE=tpmrm0 .
RUN make -j$(nproc)
RUN bash utils/keys_gen.sh .

# Setup client-sdk.
WORKDIR /build_dir/
RUN mkdir /fdoout
WORKDIR /build_dir/fdoout/
RUN git clone -b v1.1.7 https://github.com/fido-device-onboard/client-sdk-fidoiot.git client-sdk
WORKDIR /build_dir/fdoout/client-sdk/
RUN grep -rl '/opt/openssl' ./ | xargs sed -i 's/\/opt\/openssl/\/usr/g'
RUN grep -rl '/opt/curl' ./ | xargs sed -i 's/\/opt\/curl/\/usr/g'
RUN cmake -DDA=ecdsa384 .
RUN make -j$(nproc)
RUN bash utils/keys_gen.sh .

################################################################################
FROM ubuntu:22.04 as FDO-DI

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get upgrade -y && apt-get install -y \
  bash \
  libxml2 \
  uuid \
  libsystemd-dev \
  libproxy-dev \
  libcurl4-gnutls-dev \
  dmidecode \
  openssl \
  wget \
  libnghttp2-dev \
  iproute2 \
  bsdmainutils && apt-get clean

# Install curl built from fdo-client-sdk-builder
COPY --from=fdo-client-sdk-builder "/usr/local/lib/libcurl.so.4.8.0" "/usr/lib/"
COPY --from=fdo-client-sdk-builder "/usr/local/bin/curl" "/usr/bin/curl"
COPY --from=fdo-client-sdk-builder "/usr/local/include/curl" "/usr/include/curl"

# explicitly set symlinks
RUN ln -s /usr/lib/libcurl.so.4.8.0 /usr/lib/libcurl.so.4
RUN ln -s /usr/lib/libcurl.so.4 /usr/lib/libcurl.so

WORKDIR /

# Client-SDK CSE
RUN mkdir /cse-fdoout
RUN mkdir /cse-fdoout/build
RUN mkdir /cse-fdoout/data
COPY --from=fdo-client-sdk-builder /build_dir/cse-fdoout/client-sdk/build/linux-client /cse-fdoout/
COPY --from=fdo-client-sdk-builder /build_dir/cse-fdoout/client-sdk/build/cse-clear .
COPY --from=fdo-client-sdk-builder /build_dir/cse-fdoout/client-sdk/data/ /cse-fdoout/data/

# Client-SDK TPM
RUN mkdir /tpm-fdoout
RUN mkdir /tpm-fdoout/build
RUN mkdir /tpm-fdoout/data
RUN mkdir /tpm-fdoout/utils
COPY --from=fdo-client-sdk-builder /build_dir/tpm-fdoout/client-sdk/build/linux-client /tpm-fdoout/
COPY --from=fdo-client-sdk-builder /build_dir/tpm-fdoout/client-sdk/data/ /tpm-fdoout/data/
COPY --from=fdo-client-sdk-builder /build_dir/tpm-fdoout/client-sdk/utils/tpm_make_ready_ecdsa.sh /tpm-fdoout/utils/
COPY --from=fdo-client-sdk-builder /usr/local/lib /usr/local/lib
COPY --from=fdo-client-sdk-builder /usr/local/bin /usr/local/bin
COPY --from=fdo-client-sdk-builder /usr/lib/x86_64-linux-gnu/ossl-modules /usr/lib/x86_64-linux-gnu/ossl-modules
RUN ldconfig

# Client-SDK x86
RUN mkdir /fdoout
RUN mkdir /fdoout/build
RUN mkdir /fdoout/data
COPY --from=fdo-client-sdk-builder /build_dir/fdoout/client-sdk/build/linux-client /fdoout/
COPY --from=fdo-client-sdk-builder /build_dir/fdoout/client-sdk/data/ /fdoout/data/

# Finally copy the runnable script that executes DI.
COPY run_fdo_di.sh /usr/bin/
RUN chmod +x /usr/bin/run_fdo_di.sh

COPY run_fdo_to.sh /usr/bin/
RUN chmod +x /usr/bin/run_fdo_to.sh

COPY run_fdo_in_docker.sh /usr/bin/
RUN chmod +x /usr/bin/run_fdo_in_docker.sh
WORKDIR /

CMD ["/usr/bin/run_fdo_in_docker.sh"]
