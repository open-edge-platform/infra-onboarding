# Hook microOS (Alpine Linuxkit)

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Get Started](#get-started)
- [Contribute](#contribute)
- [Community and Support](#community-and-support)
- [License](#license)

## Overview

Hook is the Tinkerbell Installation Environment for bare-metal. It runs in-memory, installs operating system,
and handles deprovisioning. Its based on [LinuxKit](https://github.com/linuxkit/linuxkit).

This repo is forked from the open source repo [github.com/tinkerbell/hook](https://github.com/tinkerbell/hook).

Following components have been added to the open source HookOS for our specific purpose

- Device Discovery: This service can read all the hardware data (serial number, UUID etc) and send out to the Edge Orchestrator
while provisioning.
- Fluent Bit: This service helps us stream the logs of all important services to the Observability microservice
of Edge Orchestrator.
- Caddy: This service is used as a proxy to communicate with the Edge Orchestrator securely.

## Features

- Lightweight OS: Size is less than 300MB.
- Easy to customise: Services can be embedded within the OS using individual container images and can be configured
in a simple YAML file. Files can be embedded similarly.
- Device Discovery: This service can read all the hardware data (serial number, UUID etc) and send out to the Edge Orchestrator
for provisioning.

## Get Started

Instructions on how to build HookOS on your machine.

### Develop the HookOS

There are several convenient `make` targets to support developer activities, you can use `help` to see a list of makefile
targets. The following is a list of makefile targets that support developer activities:

- `lint` to run a list of linting targets
- `build` to build the compressed HookOS image in `tar.gz` format

#### Builds Component

```bash
make <COMPONENT NAME>
```

Components can be `device_discovery`, `fluent-bit`, `caddy` or `hook_dind`

Example

```bash
make device_discovery
```

#### Builds container images of all the components to be embedded

```bash
make components
```

#### Configures Edge Node and Edge Orchestrator parameters

```bash
make configure
```

All the configurable parameter details can be found in [config.template](config.template)

#### Creates placeholder for Edge Orchestrator SSL Certificates

```bash
make certs
```

#### Builds HookOS kernel container image

**NOTE**:This target will build kernel container image even if another image with identical tag available locally
or in the Production Release service.

```bash
make kernel
```

The container image tag is determined by

1. Linux kernel version: For this project we currently support only `Linux 5.10`.
Its a Long-Term Support (LTS) kernel which is deemed to receive security updates and bug fixes until end of 2026.
2. Linux kernel point release: Extensive provisioning tests across various platforms have been
successfully conducted using Kernel Point Release `228`, which is the current default point release.
This can be modified by updating `HOOK_KERNEL_POINT_RELEASE` inside [Makefile](Makefile).
3. `SHA256` hash of combined contents of [Dockerfile](hook-os/hook/kernel/Dockerfile) and
[kernel parameters](hook-os/hook/kernel/configs/generic-5.10.y-x86_64):
Any change to these files will lead to a different kernel tag.

Example of a kernel tag: `5.10.228-95e4df98`

**NOTE: It has been observed that building the kernel with identical parameters**
**and environment variables results in a different container image SHA ID in every run.**

#### Builds HookOS binaries

**NOTE**:This target fails if the kernel container image is not available locally or in the Production Release service.
So either run `make kernel` before executing this target or run `make build` to combine both.

```bash
make binaries
```

The output can be found in the `out/` directory.

#### Builds the complete HookOS artifact

```bash
make build
```

The output can be found in the `out/` directory.

This process compiles all components, creates placeholders for certificates,
builds the kernel (if necessary), generates the binaries, and packages everything into a `.tar.gz` archive file.

The kernel image is built fresh locally only if the expected image tag is not available locally
or in the Production Release service.

#### Publish HookOS kernel as a container image to Production Release Service

**NOTE**:This target is intended exclusively for use within the CI/CD pipeline.

```bash
make push-kernel-ci
```

This pushes the HookOS kernel image to the Release Service only if the image tag
(not to be confused by image SHA ID) isn't already present in the Release Service.

#### Publish HookOS as a OCI artifact to Production Release Service

**NOTE**:This target is intended exclusively for use within the CI/CD pipeline.

```bash
make publish-binaries-ci
```

#### Publish both HookOS kernel container image and HookOS OCI artifact

**NOTE**:This target is intended exclusively for use within the CI/CD pipeline.

```bash
make artifact-publish
```

#### Lint for License, ShellCheck, and Markdown

```bash
make lint
```

## Contribute

To learn how to contribute to the project, see the [contributor's guide][contributors-guide-url].

## Community and Support

To learn more about the project, its community, and governance, visit
the [Edge Orchestrator Community](https://community.intel.com/).

For support, start with [troubleshooting][troubleshooting-url] or [contact us](mailto:adreanne.bertrand@intel.com).

## License

Edge Orchestrator is licensed under [Apache License
2.0](http://www.apache.org/licenses/LICENSE-2.0).

- For more information on how to onboard an edge node, refer to the [user guide on onboarding an edge node][user-guide-onboard-edge-node].
- To get started, check out the [user guide][user-guide-url].
- For the infrastructure manager development guide, visit the [infrastructure manager development guide][inframanager-dev-guide-url].
- If you are contributing, please read the [contributors guide][contributors-guide-url].
- For troubleshooting, see the [troubleshooting guide][troubleshooting-url].

[user-guide-onboard-edge-node]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/set_up_edge_infra/edge_node_onboard.html
[user-guide-url]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/get_started_guide/gsg_content.html
[inframanager-dev-guide-url]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/get_started_guide/gsg_content.html
[contributors-guide-url]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/index.html
[troubleshooting-url]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/troubleshooting/troubleshooting.html

Last Updated Date: March 24, 2025
