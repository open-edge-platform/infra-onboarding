# Edge Infrastructure Manager Onboarding and Provisioing service

## Table of Contents

- [Edge Infrastructure Manager Onboarding and Provisioing service](#edge-infrastructure-manager-onboarding-and-provisioing-service)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [Features](#features)
  - [API](#api)
  - [Get Started](#get-started)
    - [Dependencies](#dependencies)
    - [Build the Binary](#build-the-binary)
  - [Contribute](#contribute)

## Overview

The Onboarding Manager (OM) is a critical component in the Edge Infrastructure Manager,
responsible for managing the onboarding process of edge nodes.
The Onboarding Manager handles.the entire edge node onboarding and provisioning process,
from pre-registration and system power-on up to iPXE boot,authentication,
and device discovery. It manages host resources,
facilitates user interaction for node verification, creates instance resources, and
generates Tinkerbell workflows for provisioning, which are executed by the Tink Agent.

## Features

- Automated Onboarding: Streamlines the onboarding process for edge nodes,
  from pre-registration to provisioning.
- Interactive and Non Interactive (passwordless) Onboarding
  support to onboard edge node.
- Host and Instance Resource Management: Interfaces with the
  Inventory Service to manage host and instance resources lifecycle.
- Workflow Automation: Generates and executes Tinkerbell workflows
  for provisioning edge nodes.
- Secure Boot and FDE Support: Supports Secure Boot and
  Full Disk Encryption (FDE) settings for enhanced security.
- Integration with Keycloak: Ensures secure authentication and
  token management for edge nodes.
- Status Reporting: Sends onboarding and provisioning status
  information to the User Interface via the Inventory Service.
- Scalability: Designed to scale with approximately 45 edge nodes
  and provisioning tasks,as validated to date.

## API

- Inventory Interaction: The onboarding manager uses the gRPC APIs
  exposed by the infra Inventory Service to manage host,instance,os resources.
- Device Discovery: The onboarding manager provides both unay
  and bi-directional stream gRPC-based APIs for device discovery.
- gRPC Stream Management for Non-Interactive Onboarding: The API establishes
  and manages gRPC stream connections with edge nodes to facilitate
  non-interactive onboarding.

The relevant API definitions can be found in
[this API directory](https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/tree/main/onboarding-manager/api/grpc/onboardingmgr)
and
[this API package](https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.eim-onboarding/tree/main/onboarding-manager/pkg/api).

## Get Started

Instructions on how to install and set up the onboarding-manager on your machine.

### Dependencies

Firstly, please verify that all dependencies have been installed.

```bash
# Return errors if any dependency is missing
make dependency
```

This code requires the following tools to be installed on your development machine:

- [Go\* programming language](https://go.dev)
  check [$GOVERSION_REQ](Makefile)
- [golangci-lint](https://github.com/golangci/golangci-lint)
  check [$GOLINTVERSION_REQ](Makefile)
- [buf](https://github.com/bufbuild/buf)
  check [$BUFVERSION_REQ](Makefile)
- [protoc-gen-doc](https://github.com/pseudomuto/protoc-gen-doc)
  check [$PROTOCGENDOCVERSION_REQ](Makefile)
- [protoc-gen-go-grpc](https://pkg.go.dev/google.golang.org/grpc/cmd/protoc-gen-go-grpc)
  check [$PROTOCGENGOGRPCVERSION_REQ](Makefile)
- [protoc-gen-go](https://pkg.go.dev/google.golang.org/protobuf/cmd/protoc-gen-go)
  check [$PROTOCGENGOVERSION_REQ](Makefile)
- [protoc-gen-validate](https://github.com/bufbuild/protoc-gen-validate)
  check[$PROTOCGENVALIDATEGOVERSION_REQ](Makefile)
- [GNU Compiler Collection](https://gcc.gnu.org/)

### Build the Binary

Build the project as follows:

```bash
# Build go binary
make build
```

The binary is installed in the [$OUT_DIR](../common.mk) folder.

## Contribute

To learn how to contribute to the project, see the [contributor's guide][contributors-guide-url].
The project will accept contributions through Pull-Requests (PRs).
PRs must be built successfully by the CI pipeline, pass linters
verifications and the unit tests.

There are several convenience make targets to support developer activities,
you can use `help` to see a list of makefile targets.
The following is a list of makefile targets that support developer activities:

- `generate` to generate the database schema, Go code, and the Python binding
  from the protobuf definition of the APIs
- `lint` to run a list of linting targets
- `mdlint` to run linting of this file.
- `test` to run the unit test
- `go-tidy` to update the Go dependencies and regenerate the `go.sum` file
- `build` to build the project and generate executable files
- `docker-build` to build the Inventory Docker container

- For more information on how to onboard an edge node, refer to the
  [user guide on onboarding an edge node][user-guide-onboard-edge-node].
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

Last Updated Date: February 17, 2025
