FROM debian:12.7

ENV DEBIAN_FRONTEND=noninteractive
RUN apt update
RUN apt install -y parted grub-common udev grub-pc-bin grub-efi-amd64-bin dosfstools wget iproute2

COPY partition.sh /

RUN chmod +x partition.sh

CMD ["/bin/bash", "/partition.sh"]