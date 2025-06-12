# Edge Infrastructure Manager PXE server

## Table of Contents

- [Edge Infrastructure Manager PXE server](#edge-infrastructure-manager-pxe-server)
    - [Table of Contents](#table-of-contents)
    - [Overview](#overview)
    - [Features](#features)
    - [Get Started](#get-started)
    - [Contribute](#contribute)

## Overview

TODO

## Features

- Exposes a minimal, read-only TFTP server to serve iPXE binaries.
- Runs in a ProxyDHCP mode to provide PXE boot information to PXE firmware clients.

## Get Started

Instructions on how to install and set up the onboarding-manager on your machine.

### Dependencies

This repository only contains scripts to build the Docker image, so you only need `docker` installed.

To build the PXE server image run:

```bash
make docker-build
```

## Contribute

To learn how to contribute to the project, see the [contributor's guide][contributors-guide-url].
The project will accept contributions through Pull-Requests (PRs).
PRs must be built successfully by the CI pipeline, pass linters
verifications and the unit tests.

There are several convenience make targets to support developer activities,
you can use `help` to see a list of makefile targets.
The following is a list of makefile targets that support developer activities:

- `lint` to run a list of linting targets.
- `mdlint` to run linting of this file.
- `hadolint` to run linter on Dockerfile.
- `docker-build` to build the Docker container.

- For more information on how to onboard an edge node, refer to the
  [user guide on onboarding an edge node][user-guide-onboard-edge-node].
- To get started, check out the [user guide][user-guide-url].
- For the infrastructure manager development guide, visit the
  [infrastructure manager development guide][inframanager-dev-guide-url].
- If you are contributing, please read the [contributors guide][contributors-guide-url].
- For troubleshooting, see the [troubleshooting guide][troubleshooting-url].

[user-guide-onboard-edge-node]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/user_guide/set_up_edge_infra/index.html
[user-guide-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/user_guide/get_started_guide/index.html
[inframanager-dev-guide-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/infra_manager/index.html
[contributors-guide-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html
[troubleshooting-url]: https://docs.openedgeplatform.intel.com/edge-manage-docs/main/user_guide/troubleshooting/index.html
