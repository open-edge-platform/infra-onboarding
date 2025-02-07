# Edge Infrastructure Manager Provisioning Artifacts Server

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Get Started](#get-started)
- [Contribute](#contribute)
- [Community and Support](#community-and-support)
- [License](#license)

## Overview

Provisioning Artifacts Server (PA Server) is an NGINX based server which hosts files required for provisioning. These files are provided by DKAM (EFI file, HookOS binaries, iPXE files, GRUB files)


## Features

- Can handle 100+ parallel client requests

## Get Started

Instructions on how to install and set up the PA Server on your machine.

### Develop the PA Server

There are several convenience make targets to support developer activities, you can use `help` to see a list of makefile
targets. The following is a list of makefile targets that support developer activities:

- `lint` to run a list of linting targets
- `docker-build` to build the PA Server Docker container
- `test` to run the PA Server unit test

#### Build the Binary

Build the project as follows:

```bash
make  build
```

#### Runs build, lint, and test stages

```bash
make all
```

#### Builds the Docker image

```bash
make docker-build
```

#### Push the Docker image

```bash
make docker-push
```

#### Runs tests and generates output

```bash
make test
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

Last Updated Date: February 7, 2025