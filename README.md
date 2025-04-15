# Edge Infrastructure Manager Onboarding and OS provisioning

## Overview

The repository includes the onboarding and os provisioning micro-services of the Edge Infrastructure Manager of the
Edge Manageability Framework.

## Get Started

The repository comprises the following components and services:

- [**Onboarding-Manager**](onboarding-manager/): implements a resource manager to onboard and provision edge nodes.
- [**DKAM**](dkam/): Dynamic Kit Adaptation Module, manages OS profiles, builds iPXE binaries with digital signatures,
  and facilitates MicroOS operating system provisioning, based on the manifest file in the orchestrator environment.
- [**HookOS**](hook-os/): contains the Tinkerbell installation environment for bare-metal. It runs in-memory, installs
  operating system, and handles deprovisioning.
- [**Tinker Actions**](tinker-actions/): contains custom Tinkerbell Actions that are used to compose Tinkerbell Workflows.

Read more about Edge Orchestrator in the [User Guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/user_guide/index.html$0).

## Develop

To develop one of the Managers, please follow its guide in README.md located in its respective folder.

## Contribute

To learn how to contribute to the project, see the [Contributor's
Guide](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/contributor_guide/index.html).

## Community and Support

To learn more about the project, its community, and governance, visit
the [Edge Orchestrator Community](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/index.html).

For support, start with [Troubleshooting](https://docs.openedgeplatform.intel.com/edge-manage-docs/main/developer_guide/troubleshooting/index.html)

## License

Each component of the Edge Infrastructure onboarding is licensed under [Apache 2.0][apache-license].

Last Updated Date: April 7, 2025

[apache-license]: https://www.apache.org/licenses/LICENSE-2.0
