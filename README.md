# Edge Infrastructure Manager Onboarding and OS provisioning

## Overview

The repository includes the onboarding and os provisioning micro-services of the Edge Infrastructure Manager of the
Edge Manageability Framework.

## Get Started

The repository comprises the following components and services:

- [**Onboarding-Manager**](onboarding-manager/): implements a resource manager to onboard and provision edge nodes.
- [**DKAM**](dkam/): manages OS profiles, builds iPXE binaries with digital signatures, and facilitates MicroOS provisioning
  based on the manifest file in the orchestrator environment.
- [**HookOS**](hook-os/): contains the Tinkerbell installation environment for bare-metal. It runs in-memory, installs
  operating system, and handles deprovisioning.
- [**Tinker Actions**](tinker-actions/): contains custom Tinkerbell Actions that are used to compose Tinkerbell Workflows.

Read more about Edge Orchestrator in the TODO [User Guide][user-guide-url].

## Develop

To develop one of the services, please follow its guide in README.md located in its respective folder.

## Contribute

To learn how to contribute to the project, see the \[Contributor's
Guide\](<https://website-name.com>).

## Community and Support

To learn more about the project, its community, and governance, visit
the \[Edge Orchestrator Community\](<https://website-name.com>).

For support, start with \[Troubleshooting\](<https://website-name.com>) or
\[contact us\](<https://website-name.com>).

## License

Each component of the Edge Infrastructure core is licensed under
[Apache 2.0][apache-license].

Last Updated Date: April 7, 2025

[user-guide-url]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/get_started_guide/gsg_content.html
[apache-license]: https://www.apache.org/licenses/LICENSE-2.0
