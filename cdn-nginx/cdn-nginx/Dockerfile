FROM nginx:1.25.3-alpine

# Set environment variables
ENV USER=nginx
ENV HOME=/home/$USER

# Change ownership of necessary directories to the new user
RUN chown -R $USER /var/cache/nginx /var/run /var/log/nginx /etc/nginx /usr/share/nginx/html
RUN chmod 777 -R /var/cache/nginx /var/run /var/log/nginx /etc/nginx /usr/share/nginx/html

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

# Set the working directory
WORKDIR $HOME

# Copy the init script
COPY init $HOME/

# Set permissions for the script
RUN chown -R $USER $HOME/init
RUN chmod 777 $HOME/init

COPY server/etc/nginx /etc/nginx
COPY server/etc/php /etc/php81
RUN chown -R $USER /etc/php81
RUN chmod 777 -R /etc/php81
COPY src /template
RUN chown -R $USER /template
RUN chmod 777 -R /template
RUN mkdir /var/run/php
RUN chown -R $USER /var/run/php
RUN chmod 777 -R /var/run/php
STOPSIGNAL SIGTERM

RUN php-fpm81
RUN chown $USER -R /var/log/php81
RUN chmod 777 -R /var/log/php81
RUN chmod 777 -R /usr/share/nginx/html
RUN chown -R $USER /usr/share/nginx/html

EXPOSE 80
EXPOSE 443

# Switch to the new user
USER $USER

ENTRYPOINT ["tini", "--"]
CMD ["./init"]
