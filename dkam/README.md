# Edge Infrastructure Manager DKAM service

## Table of Contents

- [Edge Infrastructure Manager DKAM service](#edge-infrastructure-manager-dkam-service)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Features](#features)
  - [Get Started](#get-started)
    - [Build the Binary](#build-the-binary)
  - [Contribute](#contribute)

## Overview

The DKAM stands for Dynamic Kit Adaptation Module is a component
in the Edge Infrastructure Manager,responsible to onboard edge node with
curated sets of bare metal agents and software that can support deployment
of other apps. It prepares artifacts like iPXE, HookOS for onboarding.

## Features

- Secure Boot support: Generate signing keys to enroll inside
  UEFI BIOS Secure Boot Settings
- iPXE build support: Build iPXE binary, inject orchestrator
  certificate and sign the binary for secure boot.
- HookOS Configurations: Download prebuilt HookOS, inject certificates
  and required configurations and sign the image.

## Get Started

Instructions on how to install and set up the DKAM on your machine.

This code requires the following tools to be installed on your development machine:

- [Go\* programming language](https://go.dev) - check [$GOVERSION_REQ](Makefile)
- [golangci-lint](https://github.com/golangci/golangci-lint) - check [$GOLINTVERSION_REQ](Makefile)

### Build the Binary

Build the project as follows:

```bash
# Build go binary
make build
```

The binary is installed in the [$OUT_DIR](../common.mk) folder.

## Contribute

To learn how to contribute to the project, see the [contributor's guide][contributors-guide-url]
The project will accept contributions through Pull-Requests (PRs).
PRs must be built successfully by the CI pipeline, pass linters
verifications and the unit tests.

There are several convenience make targets to support developer activities,
you can use `help` to see a list of makefile targets.
The following is a list of makefile targets that support developer activities:

- `generate` to generate the database schema, Go code, and the Python
  binding from the protobuf definition of the APIs
- `lint` to run a list of linting targets
- `mdlint` to run linting of this file.
- `test` to run the unit test
- `go-tidy` to update the Go dependencies and regenerate the `go.sum` file
- `build` to build the project and generate executable files
- `docker-build` to build the Inventory Docker container

- For more information on how to onboard an edge node,
  refer to the [user guide on onboarding an edge node][user-guide-onboard-edge-node].
- To get started, check out the [user guide][user-guide-url].
- For the infrastructure manager development guide, visit the
  [infrastructure manager development guide][inframanager-dev-guide-url].
- If you are contributing, please read the [contributors guide][contributors-guide-url].
- For troubleshooting, see the [troubleshooting guide][troubleshooting-url].

[user-guide-onboard-edge-node]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/set_up_edge_infra/edge_node_onboard.html
[user-guide-url]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/get_started_guide/gsg_content.html
[inframanager-dev-guide-url]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/get_started_guide/gsg_content.html
[contributors-guide-url]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/index.html
[troubleshooting-url]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/troubleshooting/troubleshooting.html

Last Updated Date: February 7, 2025
