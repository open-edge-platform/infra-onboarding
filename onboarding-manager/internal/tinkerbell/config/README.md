# Network Configuration System

This directory contains the network configuration system for multi-NIC edge node provisioning.

## Overview

The network configuration system allows you to define network interfaces, VLANs, bonds, and routing using YAML configuration files instead of hardcoded values. This provides:

- **Flexibility**: Easy to modify network configurations without code changes
- **Validation**: Automatic validation of configuration syntax and requirements
- **Fallback**: Graceful fallback to single NIC if configuration is missing or invalid
- **Maintainability**: Configuration files can be version controlled and managed separately

## File Structure

```
config/
├── network_config.go           # Configuration parsing and validation
└── README.md                   # This file
```

## Configuration File Locations

The system searches for configuration files in order:
1. `/etc/infra-onboarding/network-config.yaml`
2. `/etc/infra-onboarding/network-config.yml`
3. `./configs/network-config.yaml`
4. `./configs/network-config.yml`

## Configuration Schema

### Full Schema
```yaml
interfaces:           # Required: List of network interfaces
  - name: string      # Required: Interface name
    mac_address: string # Required: MAC address (XX:XX:XX:XX:XX:XX)
    addresses:        # Optional: List of IP addresses (required if dhcp: false)
      - "IP/CIDR"
    gateway: string   # Optional: Gateway IP address
    dhcp: boolean     # Optional: Use DHCP (default: false)
    dns:              # Optional: DNS servers
      - "DNS_IP"
    mtu: integer      # Optional: MTU size
    optional: boolean # Optional: Interface is optional (default: false)

vlans:                # Optional: VLAN configurations
  - name: string      # Required: VLAN interface name
    id: integer       # Required: VLAN ID (1-4094)
    link: string      # Required: Parent interface name
    addresses:        # Optional: IP addresses
      - "IP/CIDR"
    gateway: string   # Optional: Gateway IP
    dns:              # Optional: DNS servers
      - "DNS_IP"

bonds:                # Optional: Bond configurations  
  - name: string      # Required: Bond interface name
    interfaces:       # Required: List of member interfaces
      - "interface_name"
    mode: string      # Required: Bond mode (active-backup, balance-rr, etc.)
    addresses:        # Optional: IP addresses
      - "IP/CIDR"
    gateway: string   # Optional: Gateway IP
    dns:              # Optional: DNS servers
      - "DNS_IP"

routes:               # Optional: Custom routes
  - to: string        # Required: Destination network (CIDR)
    via: string       # Optional: Gateway IP (required if interface not specified)
    interface: string # Optional: Interface name (required if via not specified)
    metric: integer   # Optional: Route metric
```

## Examples

### Single NIC (DHCP)
```yaml
interfaces:
  - name: eth0
    mac_address: "00:11:22:33:44:55"
    dhcp: true
```

### Dual NIC (Static IPs)
```yaml
interfaces:
  - name: mgmt
    mac_address: "00:11:22:33:44:66"
    addresses: ["192.168.100.10/24"]
    gateway: "192.168.100.1"
    dns: ["192.168.100.1"]
  - name: prod
    mac_address: "00:11:22:33:44:77"
    addresses: ["10.0.1.100/24"]
    dns: ["10.0.1.1"]
```

### Bond Configuration
```yaml
interfaces:
  - name: eth0
    mac_address: "00:11:22:33:44:66"
  - name: eth1
    mac_address: "00:11:22:33:44:77"

bonds:
  - name: bond0
    interfaces: [eth0, eth1]
    mode: active-backup
    addresses: ["192.168.1.100/24"]
    gateway: "192.168.1.1"
```

## Validation Rules

The system validates:
- **Required fields**: Interface name and MAC address are mandatory
- **Unique values**: No duplicate interface names or MAC addresses
- **Valid formats**: MAC addresses, IP addresses/CIDR, VLAN IDs
- **Dependencies**: Referenced interfaces exist (for VLANs and bonds)
- **Logic**: Addresses required when DHCP is disabled

## Error Handling

If configuration validation fails:
1. Detailed error is logged with field and line information
2. System falls back to single NIC configuration
3. Provisioning continues with base cloud-init template
4. Error details help identify configuration issues

## Integration

The configuration system integrates with:
- **template_data.go**: Main processing logic
- **Cloud-init**: Generated configurations use cloud-init format
- **Tinkerbell**: Configurations processed during provisioning workflow
- **Logging**: All operations logged for debugging

## Best Practices

1. **Validation**: Always validate configurations before deployment
2. **Version Control**: Keep configuration files in version control
3. **Testing**: Test configurations in staging environment
4. **Documentation**: Document network topology and IP assignments
5. **Backup**: Maintain backups of working configurations
6. **Templates**: Create reusable templates for different node types

## Migration Guide

To migrate from hardcoded configurations:

1. **Identify current network settings** in your provisioning code
2. **Create configuration file** using the appropriate example as template
3. **Update MAC addresses** to match your hardware
4. **Update IP addresses** to match your network topology
5. **Place configuration file** in search path
6. **Test with single node** before mass deployment
7. **Remove hardcoded network configuration** from code

The system will automatically detect and use the configuration file, falling back to single NIC if needed.
