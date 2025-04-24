# Edge Infrastructure Manager Tinker Actions

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Get Started](#get-started)
- [Contribute](#contribute)
- [Community and Support](#community-and-support)
- [License](#license)

## Overview

This repository is a suite of reusable Tinkerbell Actions that are used to compose Tinkerbell Workflows.
Tinkerbell Actions are reusable, containerized steps that are used to compose Tinkerbell Workflows.
Each action performs a specific task, such as provisioning an operating system, configuring network settings,
or running custom scripts. These actions are executed in sequence as part of a Tinkerbell Workflow to automate
the provisioning and management of bare metal servers.

Some of the tinker actions have been forked from the [github.com/tinkerbell/actions](https://github.com/tinkerbell/actions).
Summary of all the changes and contributions can be found [here](CHANGELOG.md)

Following is the list of tinker actions maintained:

| Action Name             | Description                                                               |
|-------------------------|---------------------------------------------------------------------------|
| cexec                   | chroot and execute binaries                                               |
| efibootset              | modify the boot order to prioritize the installed OS disk after a restart |
| erase_non_removable_disks | wipe data out in all the non-removable physical disks connected         |
| fde                     | setup and enable Full Disk Encryption                                     |
| image2disk              | write images to a block device                                            |
| kernelupgrd             | upgrade the kernel to the latest HWE version                              |
| qemu_nbd_image2disk     | write image to block device using qemu-nbd and dd                         |
| securebootflag          | check for secure boot                                                     |
| emt_partition           | create partition for Edge Microvisor Toolkit                              |
| writefile               | write a file to a file system on a block device                           |

## Features

- Designed to be modular and reusable.
- Each action is typically defined as a Docker container, which encapsulates the logic and dependencies
  required to perform the task.
- Automatic Destination Drive Detection: All the actions have logic to automatically detect the target disk,
  based on size, type of the disk.

## Get Started

Instructions on how to build tinker actions on your machine.

### Develop the Tinker Actions

There are several convenient `make` targets to support developer activities, you can use `help` to see a list of makefile
targets. The following is a list of makefile targets that support developer activities:

- `lint` to run a list of linting targets
- `docker-build` to build all the tinker action images

#### Run build, lint stages

```bash
make all
```

#### Build tinker action

```bash
make <ACTION NAME>
```

Example

```bash
make erase_non_removable_disks
```

#### Push tinker action with branch name as tag

```bash
make push-<ACTION NAME>
```

Example

```bash
make push-cexec
```

#### Push tinker action with VERSION as tag

```bash
make release-<ACTION NAME>
```

Example

```bash
make release-fde
```

#### Build all tinker actions

```bash
make docker-build
```

#### Push all tinker actions with BRANCH and VERSION tag

```bash
make docker-push
```

## Contribute

To learn how to contribute to the project, see the [contributor's guide][contributors-guide-url].

## Community and Support

To learn more about the project, its community, and governance, visit
the [Edge Orchestrator Community](https://community.intel.com/).

For support, start with [troubleshooting][troubleshooting-url].

## License

Edge Orchestrator is licensed under [Apache License
2.0](http://www.apache.org/licenses/LICENSE-2.0).

- For more information on how to onboard an edge node, refer to the [user guide on onboarding an edge node][user-guide-onboard-edge-node].
- To get started, check out the [user guide][user-guide-url].
- For the infrastructure manager development guide, visit the [infrastructure manager development guide][inframanager-dev-guide-url].
- If you are contributing, please read the [contributors guide][contributors-guide-url].
- For troubleshooting, see the [troubleshooting guide][troubleshooting-url].

[user-guide-onboard-edge-node]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/user_guide/set_up_edge_infra/index.html
[user-guide-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/user_guide/get_started_guide/index.html
[inframanager-dev-guide-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/infra_manager/index.html
[contributors-guide-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html
[troubleshooting-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/user_guide/troubleshooting/index.html

Last Updated Date: March 24, 2025
