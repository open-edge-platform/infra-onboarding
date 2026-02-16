# Hook microOS (Alpine Linuxkit)

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Get Started](#get-started)
- [Contribute](#contribute)
- [Community and Support](#community-and-support)
- [License](#license)

## Overview

Following components have been added to the open source HookOS for our specific purpose

- Device Discovery: This service can read all the hardware data (serial number, UUID etc) and send out to the Edge Orchestrator
while provisioning.
- Fluent Bit: This service helps us stream the logs of all important services to the Observability microservice
of the Edge Orchestrator.
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

There are several convenient `make` targets to support developer activities. You can use `help` to see a list of makefile
targets. The following is a list of makefile targets that support developer activities:

- `lint` to run a list of linting targets
- `build` to build the compressed HookOS image in `tar.gz` format

#### Build Component

```bash
make <COMPONENT NAME>
```

Components can be `device_discovery`, `fluent-bit`

Example

```bash
make device_discovery
```

#### Build container images of all the components to be embedded

```bash
make components
```

#### Configure Edge Node and Edge Orchestrator parameters

```bash
make configure
```

All the configurable parameter details can be found in [config.template](config.template)

#### Create placeholder for Edge Orchestrator SSL Certificates

```bash
make certs
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

Last Updated Date: March 31, 2025
