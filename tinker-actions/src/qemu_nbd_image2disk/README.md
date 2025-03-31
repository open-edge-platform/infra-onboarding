# QEMU NBD IMAGE2DISK

This Action will stream a remote disk image (img format) to a block device, and is mainly used to write cloud images
to a disk. This action currently supports only `.img` files as input.

The process begins by creating and executing an HTTP GET request to download the cloud image (img format)
from the input URL.
Once the image is downloaded, it is stored in a temporary file in the `qcow2` format. Following this,
the `qcow2` image is attached
as a network block device, allowing it to be accessed as if it were a physical disk.
An optional SHA256 checksum validation is performed before proceeding to the final step.
Finally, the operating system is installed onto the target disk using the `dd` command,
which performs a low-level copy of the image data to the block storage device.

The primary motivation behind using this action over [qemuimg2disk](https://github.com/tinkerbell/actions/tree/main/qemuimg2disk)
is the execution time. `qemu_nbd_image2disk` has been observed to be significantly faster than `qemuimg2disk`.
While `qemuimg2disk` streams the file directly onto the disk using `qemu-img convert`, `qemu_nbd_image2disk` adds an intermediate
step of downloading the image and then writing it onto the disk.

| env var | data type | default value | required | description |
|---------|-----------|---------------|----------|-------------|
| IMG_URL | string | "" | yes | URL of the image to be streamed |
| DEST_DISK | string | "" | no | Block device to write the image. If not provided its selected by pre-determined algo |
| RETRY_ENABLED | bool | true | no | Retry the Action, using exponential backoff based on `RETRY_DURATION_MINUTES` |
| RETRY_DURATION_MINUTES | int | 10 | no | Duration for which the Action will retry before failing |
| PROGRESS_INTERVAL_SECONDS | int | 3 | no | Interval at which the progress of the image transfer will be logged |
| TEXT_LOGGING | bool | false | no | Output will be logged in human friendly text format, JSON used by default |
| SHA256 | string | "" | no | SHA256 Checksum of `IMG_URL` for validation |

The below example will stream ubuntu cloud image (img format) and write it to the block storage disk `/dev/sda`.

```yaml
actions:
    - name: "stream ubuntu"
      image: registry-rs.edgeorchestration.intel.com/edge-orch/infra/tinker-actions/qemu_nbd_image2disk:main
      timeout: 90
      environment:
          IMG_URL: http://192.168.1.2/ubuntu.img
          DEST_DISK: /dev/sda
```
