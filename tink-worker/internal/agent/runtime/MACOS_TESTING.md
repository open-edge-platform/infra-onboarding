<!--
SPDX-FileCopyrightText: 2025 Intel Corporation

SPDX-License-Identifier: Apache-2.0
-->

# MacOS Testing

When developing on MacOS it may be necessary to create a symlink to `/var/run/docker.sock`. First, 
validate `/var/run/docker.sock` does not exist. If it does not exist, verify the socket exists at
`$HOME/.docker/run/docker.sock` and create a symlink.

```
sudo ln -s $HOME/.docker/run/docker.sock /var/run/docker.sock
```