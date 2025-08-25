# Multi-NIC Configuration Guide

This guide explains how to configure multiple Network Interface Cards (NICs) for edge nodes during provisioning using configuration files.

## Overview

The multi-NIC configuration system now uses YAML configuration files instead of hardcoded values. This approach allows you to:
- Configure multiple network interfaces with different IP addresses and VLANs
- Set up network bonds for high availability  
- Create VLAN-based network segmentation
- Define custom routing rules for different traffic types
- Easily modify network configurations without code changes

## Configuration File Approach

### File Location Priority

The system searches for network configuration files in the following order:
1. `/etc/infra-onboarding/network-config.yaml`
2. `/etc/infra-onboarding/network-config.yml` 
3. `./configs/network-config.yaml`
4. `./configs/network-config.yml`

**If no configuration file is found or the file is invalid, the system defaults to single NIC configuration.**

### Configuration File Structure

```yaml
# Network Configuration for Edge Node Provisioning
interfaces:
  - name: interface_name
    mac_address: "XX:XX:XX:XX:XX:XX"
    addresses:          # Optional: IP addresses (required if dhcp: false)
      - "IP/CIDR"
    gateway: "IP"       # Optional: Gateway IP
    dhcp: false         # Optional: Use DHCP (default: false)
    dns:                # Optional: DNS servers
      - "DNS_IP"
    mtu: 1500          # Optional: MTU size
    optional: false     # Optional: Interface is optional (default: false)

vlans:                 # Optional: VLAN configurations
  - name: vlan_name
    id: vlan_id
    link: parent_interface
    addresses:
      - "IP/CIDR"
    gateway: "IP"
    dns:
      - "DNS_IP"

bonds:                 # Optional: Bond configurations
  - name: bond_name
    interfaces:
      - interface1
      - interface2
    mode: bond_mode    # active-backup, balance-rr, etc.
    addresses:
      - "IP/CIDR"
    gateway: "IP"
    dns:
      - "DNS_IP"

routes:                # Optional: Custom routes
  - to: "destination_network"
    via: "gateway_ip"
    interface: "interface_name"
    metric: 100
```

## Quick Start

### 1. Single NIC Configuration

Create `/etc/infra-onboarding/network-config.yaml`:

```yaml
interfaces:
  - name: eth0
    mac_address: "00:11:22:33:44:55"
    dhcp: true
```

### 2. Dual NIC Configuration

```yaml
interfaces:
  # Management interface
  - name: mgmt
    mac_address: "00:11:22:33:44:66"
    addresses:
      - "192.168.100.10/24"
    gateway: "192.168.100.1"
    dns:
      - "192.168.100.1"

  # Production interface  
  - name: prod
    mac_address: "00:11:22:33:44:77"
    addresses:
      - "10.0.1.100/24"
    dns:
      - "10.0.1.1"
```

### 3. Triple NIC with Routes

```yaml
interfaces:
  - name: mgmt
    mac_address: "00:11:22:33:44:66"
    addresses:
      - "192.168.100.10/24"
    gateway: "192.168.100.1"
    dns:
      - "192.168.100.1"

  - name: prod
    mac_address: "00:11:22:33:44:77"
    addresses:
      - "10.0.1.100/24"
    dns:
      - "10.0.1.1"

  - name: storage
    mac_address: "00:11:22:33:44:88"
    addresses:
      - "192.168.1.10/24"
    optional: true

routes:
  - to: "192.168.0.0/16"
    via: "192.168.100.1"
    interface: mgmt
    metric: 100
  - to: "10.0.0.0/8"
    via: "10.0.1.1"
    interface: prod
    metric: 200
```

## Advanced Configurations

### Bond Configuration for High Availability

```yaml
interfaces:
  # Bond member interfaces (no IP addresses)
  - name: eth0
    mac_address: "00:11:22:33:44:66"
  - name: eth1
    mac_address: "00:11:22:33:44:77"

bonds:
  - name: bond0
    interfaces:
      - eth0
      - eth1
    mode: active-backup
    addresses:
      - "192.168.1.100/24"
    gateway: "192.168.1.1"
    dns:
      - "192.168.1.1"
```

### VLAN Configuration

```yaml
interfaces:
  # Physical interface (no IP, VLANs will have IPs)
  - name: data
    mac_address: "00:11:22:33:44:66"

vlans:
  - name: data.100
    id: 100
    link: data
    addresses:
      - "172.16.100.10/24"
  - name: data.200
    id: 200
    link: data
    addresses:
      - "172.16.200.10/24"
```

## Configuration Validation

The system automatically validates configuration files for:

### Required Fields
- **Interface name**: Must be unique and non-empty
- **MAC address**: Must be valid format (XX:XX:XX:XX:XX:XX)
- **Addresses**: Required when `dhcp: false` (default)

### Validation Rules
- No duplicate interface names or MAC addresses
- Valid IP address/CIDR format
- Valid VLAN IDs (1-4094)
- Valid bond modes
- Referenced interfaces exist (for VLANs and bonds)
- Valid gateway IP addresses

### Error Handling
If configuration validation fails:
1. Error is logged with specific details
2. System falls back to single NIC configuration
3. Provisioning continues with base cloud-init template

## Deployment Workflow

1. **Create Configuration**: Write network-config.yaml with your requirements
2. **Validate Configuration**: System validates on startup
3. **Generate Cloud-Init**: Configuration is converted to cloud-init format
4. **Provision Device**: Tinkerbell applies configuration during provisioning
5. **Verify Network**: Check interfaces after provisioning

## Configuration Examples

### Production Three-Tier Setup
```yaml
interfaces:
  - name: mgmt      # Management network
    mac_address: "00:11:22:33:44:66"
    addresses: ["192.168.100.10/24"]
    gateway: "192.168.100.1"
  - name: prod      # Production workload network
    mac_address: "00:11:22:33:44:77" 
    addresses: ["10.0.1.100/24"]
  - name: storage   # Storage network
    mac_address: "00:11:22:33:44:88"
    addresses: ["192.168.1.10/24"]
```

### High Availability with Bonding
```yaml
interfaces:
  - name: mgmt
    mac_address: "00:11:22:33:44:66"
    addresses: ["192.168.100.10/24"]
    gateway: "192.168.100.1"
  - name: bond-mem1
    mac_address: "00:11:22:33:44:77"
  - name: bond-mem2
    mac_address: "00:11:22:33:44:88"

bonds:
  - name: prod-bond
    interfaces: [bond-mem1, bond-mem2]
    mode: active-backup
    addresses: ["10.0.1.100/24"]
```

## Troubleshooting

### Check Configuration File
```bash
# Verify file exists and is readable
ls -la /etc/infra-onboarding/network-config.yaml

# Validate YAML syntax
yamllint /etc/infra-onboarding/network-config.yaml
```

### Check Logs
```bash
# Check onboarding manager logs
journalctl -u onboarding-manager | grep "NetworkConfig"

# Check cloud-init logs on provisioned node
sudo journalctl -u cloud-init
sudo cat /var/log/cloud-init.log
```

### Validate Network Configuration
```bash
# On provisioned node, check interfaces
ip addr show
ip route show

# Check specific interface
ip addr show mgmt
ip addr show prod
```

## Migration from Hardcoded Configs

If you were previously using hardcoded configurations:

1. **Extract existing network settings** from your code
2. **Create configuration file** using the YAML format above
3. **Place file** in one of the search paths
4. **Remove hardcoded values** from code
5. **Test with single node** before mass deployment

## Best Practices

1. **Version Control**: Keep configuration files in version control
2. **Validation**: Always validate YAML syntax before deployment
3. **Testing**: Test configurations in staging environment first
4. **Documentation**: Document your network layout and IP assignments
5. **Backup**: Keep backup of working configurations
6. **Monitoring**: Monitor network connectivity after provisioning
7. **Templates**: Create configuration templates for different node types

This configuration file approach provides much more flexibility while maintaining the robustness of the previous implementation.
