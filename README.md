# Edge Infrastructure Manager Onboarding and OS provisioning

## Overview

The repository includes the onboarding and os provisioning micro-services of the Edge Infrastructure Manager of the
Intel® Tiber™ Edge Platform. In particular, the repository comprises the following components and services:

- [**Onboarding-Manager**](onboarding-manager/): implements a resource manager to onboard and provision edge nodes.
- [**DKAM**](dkam/): manages OS profiles, builds iPXE binaries with digital signatures, and facilitates MicroOS provisioning
  based on the manifest file in the orchestrator environment.
- [**HookOS**](hook-os/): contains the Tinkerbell installation environment for bare-metal. It runs in-memory, installs
  operating system, and handles deprovisioning.
- [**Tinker Actions**](tinker-actions/): contains custom Tinkerbell Actions that are used to compose Tinkerbell Workflows.
- [**Provisioning Artifacts Server**](pa-server/): NGINX based server which hosts files (EFI file, HookOS binaries,
  iPXE files, GRUB files) required for provisioning.

Read more about Edge Orchestrator in the [user guide on onboarding an edge node][user-guide-onboard-edge-node].

[user-guide-onboard-edge-node]: https://literate-adventure-7vjeyem.pages.github.io/edge_orchestrator/user_guide_main/content/user_guide/set_up_edge_infra/edge_node_onboard.html

Navigate through the folders to get started, develop, and contribute to Orchestrator Infrastructure.

Last Updated Date: February 17, 2025
