---
slug: image2disk
name: image2disk
tags: disk
maintainers: Dan Finneran <daniel.finneran@gmail.com>
description: "This action will stream a remote disk image (raw) to a block device, and
is mainly used to write cloud images to a disk. It is recommended to use the `qemu-img`
tool to convert disk images into raw, it is also possible to compress the raw images
with tar+gzip to prevent wasted disk space"
version: latest
---

This Action will stream a remote disk image (raw) to a block device, and
is mainly used to write cloud images to a disk. It is recommended to use the `qemu-img`
tool to convert disk images into raw, it is also possible to compress the raw images
with tar+gzip to prevent wasted disk space.

| env var | data type | default value | required | description |
|---------|-----------|---------------|----------|-------------|
| IMG_URL | string | "" | yes | URL of the image to be streamed |
| DEST_DISK | string | "" | yes | Block device to which to write the image |
| COMPRESSED | bool | false | no | Decompress the image before writing it to the disk |
| RETRY_ENABLED | bool | true | no | Retry the Action, using exponential backoff, for the duration specified in `RETRY_DURATION_MINUTES` before failing |
| RETRY_DURATION_MINUTES | int | 10 | no | Duration for which the Action will retry before failing |
| PROGRESS_INTERVAL_SECONDS | int | 3 | no | Interval at which the progress of the image transfer will be logged |
| TEXT_LOGGING | bool | false | no | Output from the Action will be logged in a more human friendly text format, JSON format is used by default |

The below example will stream a raw ubuntu cloud image (converted by qemu-img) and write
it to the block storage disk `/dev/sda`. The raw image is uncompressed in this example.

```yaml
actions:
    - name: "stream ubuntu"
      image: amr-registry.caas.intel.com/one-intel-edge/edgenode/image2disk:latest
      timeout: 90
      environment:
          IMG_URL: http://192.168.1.2/ubuntu.raw
          DEST_DISK: /dev/sda
          COMPRESSED: false
```

The below example will stream a compressed raw ubuntu cloud image (converted by qemu-img)
and then compressed with gzip to reduce local space. This also use retry.

```
qemu-img convert ubuntu.img ubuntu.raw
gzip ubuntu.raw
```

```yaml
actions:
    - name: "stream ubuntu"
      image: amr-registry.caas.intel.com/one-intel-edge/edgenode/image2disk:latest
      timeout: 90
      environment:
          IMG_URL: http://192.168.1.2/ubuntu.tar.gz
          DEST_DISK: /dev/sda
          COMPRESSED: true
          RETRY_ENABLED: true
```

## Compression format supported:

- bzip2 (`.bzip2`)
- gzip (`.gz`)
- xz (`.xz`)
- xs (`.xs`)