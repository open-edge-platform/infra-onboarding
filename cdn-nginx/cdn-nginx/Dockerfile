# Copyright (C) 2023 Intel Corporation
# SPDX-License-Identifier: BSD-3-Clause

FROM nginx:1.23.1

RUN apt update
RUN apt-get update

RUN apt -y install \
      tini \
      vim \
      wget
    

COPY init /usr/local/bin/
RUN chmod +x /usr/local/bin/init

# RUN apt -y update && apt -y upgrade
RUN apt -y install bash
RUN apt -y install nginx

# RUN apt -y install software-properties-common
# RUN add-apt-repository ppa:ondrej/php
# RUN apt update

RUN apt update
RUN apt -y upgrade
RUN apt install -y lsb-release ca-certificates apt-transport-https software-properties-common gnupg2
RUN echo "deb https://packages.sury.org/php/ $(lsb_release -sc) main" | tee /etc/apt/sources.list.d/sury-php.list
RUN wget -qO - https://packages.sury.org/php/apt.gpg | apt-key add -
RUN apt update

RUN apt -y install php8.0
RUN apt -y install php8.0-fpm php8.0-opcache
RUN apt -y install php8.0-gd php8.0-curl
# RUN apt -y install php8-zlib 
COPY server/etc/nginx /etc/nginx
COPY server/etc/php /etc/php8
COPY src /usr/share/nginx/html
RUN mkdir /var/run/php
STOPSIGNAL SIGTERM

EXPOSE 80
ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/init"]
