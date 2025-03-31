<!-- markdownlint-disable -->
This subdirectory stores a per-OS platform bundle, a script that customizes a base OS (e.g., Ubuntu 22.04) with additional
packages and configuration. Currently, the only supported OS is Ubuntu 22.04 and there is only one script under `ubuntu-22.04` folder.

> **NOTE1**: The platform bundle scripts are kept in the DKAM repo temporarily. Eventually they should be versioned and maintained outside of this repository.
> 
> **NOTE2**: The platform bundle will come from the EEF curation. The scripts here can be used as a reference for the EEF framework to build custom platform bundles on top of it.
> 

## Platform Bundle Structure

The platform bundle script is a bash script that installs additional packages and configures the base OS.

First and foremost, we decouple the provisioning of Day0/Day1 configuration from the platform bundle. The Day0/Day1 configuration is provisioned via [inframanager cloud-init](./../pkg/cloudinit/infra.cfg).
The platform bundle script should rely on the existence of the following configuration files that cloud-init writes to the OS file system:

- `/etc/intel_edge_node/client-credentials/client_id` - a file that contains the client ID for JWT authorization
- `/etc/intel_edge_node/client-credentials/client_secret` - a file that contains the client secret for JWT authorization. Both `client_id` and `client_secret` are required to authenticate to Edge Infrastructure Manager orchestrator services.
- `/etc/edge-node/node/agent_variables` - an environment file that contains all environment variables required for bare metal agents (mainly orchestrator URLs).
- `/etc/intel_edge_node/agent_versions` - an environment file that specifies a version of bare metal agents that should be installed on the EN. This file should be used by the platform bundle script to install the correct version of the agents.
<!-- markdownlint-enable-->
