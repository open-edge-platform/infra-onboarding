# Writefile

NOTICE: This file has been modified by Intel Corporation.
Original file can be found at [github.com/tinkerbell/actions](https://github.com/tinkerbell/actions).

slug: writefile
name: writefile
tags: disk
maintainers: Jason DeTiberus <jdetiberus@equinix.com>
description: "This action will mount a block device and write a file to a destination path on
it's filesystem."
version: main

The below example will write a file to the filesystem on the block device `/dev/sda3`.

```yaml
actions:
    - name: "expand ubuntu filesystem to root"
      image:  registry-rs.edgeorchestration.intel.com/edge-orch/infra/tinker-actions/writefile:main
      timeout: 90
      environment:
          DEST_DISK: /dev/sda3
          FS_TYPE: ext4
          DEST_PATH: /etc/myconfig/foo
          CONTENTS: hello-world
          UID: 0
          GID: 0
          MODE: 0600
          DIRMODE: 0700
```
