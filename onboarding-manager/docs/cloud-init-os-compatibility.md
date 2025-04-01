<!-- markdownlint-disable -->
# cloud-init OS compatibility

Edge Infrastructure Manager onboarding uses a cloud-init as a first-citizen configuration management tool that is meant to be portable among different OSes.

This document provides guidance on how to ensure compatibility with cloud-init when integrating with custom OS images.

# Overview of Edge Infrastructure Manager cloud-init

The [inframanager cloud-init](./../pkg/cloudinit/infra.cfg) is intended to be an OS-independent mechanism that is limited to providing a Day0/Day1 configuration for Edge Nodes. 

It is worth noting that the goal of cloud-init should be to provide a static configuration files or perform OS configurations that are self-contained and don't require network connectivity. Any more advanced steps should be performed by the [platform bundle script](./../platform-bundle/README.md).

# OS requirements

Although cloud-init is a widely supported tool, there are some OS-specific requirements that need to be met to ensure 
compatibility with Edge Infrastructure Manager cloud-init. An OS distribution:

- Should support the following cloud-init modules: `ca_certs`, `write_files`, `ntp`, `runcmd`.
- Should rely on `/etc/environment` for global environment variable configuration.
- Should give cloud-init write permissions and persist the following OS paths:
  - `/etc/intel_edge_node/agent_versions`
  - `/etc/edge-node/node/agent_variables`
  - `/opt/intel_edge_node/bootmgr.sh`
- Should be able to configure NTP settings using `systemd-timesyncd`.
- Should have `efibootmgr` installed for boot options configuration.
- Should support `ufw` or `iptables` for firewall settings.
<!-- markdownlint-enable-->
