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

Following components have been added to the open source Hook OS for our specific purpose

1. Device Discovery: This service can read all the hardware data (serial number, UUID etc) and send out to the Edge Orchestrator
while provisioning.
2. Fluent Bit: This service helps us stream the logs of all important services to the Observability microservice
of Edge Orchestrator.
3. Caddy: This service is used as a proxy to communicate with the Edge Orchestrator securely.

## Features

- Lightweight OS: Size is less than 300MB.
- Easy to customise: Services can be embedded within the OS using using individual docker containers and can be configured
in a simple YAML file. Files can also be embedded similarly.
- Device Discovery: This service can read all the hardware data (serial number, UUID etc) and send out to the Edge Orchestrator
while provisioning.

## Get Started

Instructions on how to build HookOS on your machine.

### Develop the Hook OS

There are several convenient `make` targets to support developer activities, you can use `help` to see a list of makefile
targets. The following is a list of makefile targets that support developer activities:

- `lint` to run a list of linting targets
- `build` to build the compressed Hook OS image in tar.gz format

#### Builds Component

```bash
make <COMPONENT NAME>
```

Components can be device_discovery, fluent-bit, caddy, hook_dind

Example

```bash
make device_discovery
```

#### Builds all the docker image components to be embedded

```bash
make components
```

#### Pulls pre-built kernel container image

```bash
make pull-kernel
```

#### Builds hook OS binaries

```bash
make build
```

## Publish HookOS binaries as OCI artifacts

```bash
make artifact-publish
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

Last Updated Date: February 25, 2025
