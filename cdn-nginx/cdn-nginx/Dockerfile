

FROM nginx:1.25.3-alpine

RUN apk --no-cache add \
      ca-certificates \
      inotify-tools \
      openssl \
      tini \
      wget && \
    apk --no-cache -U upgrade && \
    update-ca-certificates

RUN apk add bash \
      nginx \
      php php-fpm php-opcache \
      php-gd php-zlib php-curl

COPY init /usr/local/bin/
RUN chmod +x /usr/local/bin/init

COPY server/etc/nginx /etc/nginx
COPY server/etc/php /etc/php81
COPY src /template
RUN mkdir /var/run/php
STOPSIGNAL SIGTERM

EXPOSE 80
EXPOSE 443
ENTRYPOINT ["tini", "--"]
CMD ["/usr/local/bin/init"]
