# Device Discovery CLI Guide

This guide provides comprehensive information about using the Device Discovery CLI tool.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Command-Line Flags](#command-line-flags)
- [Usage Examples](#usage-examples)
- [Auto-Detection](#auto-detection)
- [Advanced Usage](#advanced-usage)
- [Troubleshooting](#troubleshooting)

## Overview

Device Discovery is a CLI tool for onboarding devices to the infrastructure platform. It communicates with the onboarding manager service via gRPC and handles authentication through Keycloak.

## Installation

Build the binary using:

```bash
make build
```

The compiled binary will be named `device-discovery`.

## Quick Start

### Minimal Command (Auto-detect system info)

```bash
./device-discovery -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -auto-detect
```

### Specify MAC, Auto-detect Rest

```bash
./device-discovery -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -mac 00:11:22:33:44:55
```

## Command-Line Flags

### Required Flags

| Flag | Type | Description |
|------|------|-------------|
| `-obm-svc` | string | Onboarding manager service address |
| `-obs-svc` | string | Onboarding stream service address |
| `-obm-port` | int | Onboarding manager port number |
| `-keycloak-url` | string | Keycloak authentication server URL |
| `-mac` | string | Device MAC address (or use `-auto-detect`) |

### Optional Device Information Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-serial` | string | auto-detect | Device serial number |
| `-uuid` | string | auto-detect | System UUID |
| `-ip` | string | auto-detect | Device IP address |

### Auto-Detection Flag

| Flag | Type | Description |
|------|------|-------------|
| `-auto-detect` | bool | Automatically detect all system information (MAC, serial, UUID, IP) |

### Additional Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-extra-hosts` | string | "" | Additional host:IP mappings (comma-separated) |
| `-ca-cert` | string | /etc/idp/server_cert.pem | Path to CA certificate |
| `-debug` | bool | false | Enable debug mode with timeout |
| `-timeout` | duration | 5m0s | Timeout for debug mode |

## Usage Examples

### Example 1: Full Auto-Detection

Automatically detect all system information:

```bash
./device-discovery \
  -obm-svc onboarding-manager.example.com \
  -obs-svc onboarding-stream.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -auto-detect
```

**Output:**
```
Auto-detected MAC address: 00:11:22:33:44:55
Auto-detected serial number: ABCD1234
Auto-detected UUID: 12345678-1234-1234-1234-123456789012
Auto-detected IP address: 192.168.1.100
Device Discovery Configuration:
  Onboarding Manager: onboarding-manager.example.com:50051
  Onboarding Stream: onboarding-stream.example.com:50051
  Keycloak URL: keycloak.example.com
  MAC Address: 00:11:22:33:44:55
  Serial Number: ABCD1234
  UUID: 12345678-1234-1234-1234-123456789012
  IP Address: 192.168.1.100
  Debug Mode: false
```

### Example 2: Specify MAC Address

Provide MAC address, auto-detect other information:

```bash
./device-discovery \
  -obm-svc onboarding-manager.example.com \
  -obs-svc onboarding-stream.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -mac 00:11:22:33:44:55
```

### Example 3: Fully Manual Configuration

Specify all device information manually:

```bash
./device-discovery \
  -obm-svc onboarding-manager.example.com \
  -obs-svc onboarding-stream.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -mac 00:11:22:33:44:55 \
  -serial DEVICE-SERIAL-123 \
  -uuid 87654321-4321-4321-4321-210987654321 \
  -ip 10.0.1.50
```

### Example 4: With Extra Hosts

Add custom host mappings to /etc/hosts:

```bash
./device-discovery \
  -obm-svc onboarding-manager.example.com \
  -obs-svc onboarding-stream.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -auto-detect \
  -extra-hosts "registry.local:10.0.0.10,api.internal:10.0.0.20"
```

### Example 5: Debug Mode with Custom Timeout

Enable debug mode with a 10-minute timeout:

```bash
./device-discovery \
  -obm-svc onboarding-manager.example.com \
  -obs-svc onboarding-stream.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -auto-detect \
  -debug \
  -timeout 10m
```

### Example 6: Custom CA Certificate

Use a custom CA certificate location:

```bash
./device-discovery \
  -obm-svc onboarding-manager.example.com \
  -obs-svc onboarding-stream.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -auto-detect \
  -ca-cert /custom/path/to/ca-cert.pem
```

## Auto-Detection

The application can automatically detect system information using standard Linux tools:

### What Gets Auto-Detected?

1. **MAC Address** (with `-auto-detect`)
   - Detects the primary network interface MAC address
   - Skips loopback and interfaces without IP addresses

2. **Serial Number** (always if not provided)
   - Uses `dmidecode -s system-serial-number`
   - Requires root/sudo privileges

3. **UUID** (always if not provided)
   - Uses `dmidecode -s system-uuid`
   - Requires root/sudo privileges

4. **IP Address** (always if MAC is known)
   - Looks up the IP address for the specified MAC address
   - Returns the first non-loopback IPv4 address

### Auto-Detection Requirements

- **Root privileges** required for dmidecode commands (serial, UUID)
- **Active network interface** required for MAC/IP detection
- **dmidecode** must be installed on the system

### Partial Auto-Detection

You can specify some values and let others be auto-detected:

```bash
# Specify MAC, auto-detect serial/UUID/IP
./device-discovery -obm-svc ... -mac 00:11:22:33:44:55

# Specify MAC and serial, auto-detect UUID/IP
./device-discovery -obm-svc ... -mac 00:11:22:33:44:55 -serial CUSTOM-SERIAL
```

## Advanced Usage

### Integration with Scripts

```bash
#!/bin/bash
# onboard-device.sh

OBM_SVC="${OBM_SVC:-onboarding-manager.example.com}"
OBS_SVC="${OBS_SVC:-onboarding-stream.example.com}"
OBM_PORT="${OBM_PORT:-50051}"
KEYCLOAK_URL="${KEYCLOAK_URL:-keycloak.example.com}"

./device-discovery \
  -obm-svc "$OBM_SVC" \
  -obs-svc "$OBS_SVC" \
  -obm-port "$OBM_PORT" \
  -keycloak-url "$KEYCLOAK_URL" \
  -auto-detect \
  "$@"  # Pass through additional arguments
```

### Using with Environment Variables

While the tool uses CLI flags, you can wrap it with environment variables:

```bash
#!/bin/bash
./device-discovery \
  -obm-svc "${OBM_SVC}" \
  -obs-svc "${OBS_SVC}" \
  -obm-port "${OBM_PORT}" \
  -keycloak-url "${KEYCLOAK_URL}" \
  -mac "${MAC_ADDRESS}" \
  ${EXTRA_FLAGS}
```

### Docker Container Usage

```dockerfile
FROM device-discovery:latest
ENTRYPOINT ["./device-discovery"]
CMD ["-auto-detect"]
```

Run with:
```bash
docker run device-discovery:latest \
  -obm-svc obm.example.com \
  -obs-svc obs.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -auto-detect
```

## Troubleshooting

### Common Issues

#### 1. "Missing required flags" error

**Problem:** Not all required flags were provided.

**Solution:** Ensure you provide all required flags:
```bash
./device-discovery -obm-svc <value> -obs-svc <value> -obm-port <value> -keycloak-url <value> -mac <value>
```

Or use auto-detection:
```bash
./device-discovery -obm-svc <value> -obs-svc <value> -obm-port <value> -keycloak-url <value> -auto-detect
```

#### 2. "Failed to auto-detect serial number" or UUID

**Problem:** dmidecode requires root privileges.

**Solution:** Run with sudo:
```bash
sudo ./device-discovery -obm-svc ... -auto-detect
```

#### 3. "No suitable network interface found"

**Problem:** No active network interface with an IP address.

**Solution:** 
- Check network connectivity: `ip addr show`
- Manually specify MAC address: `./device-discovery ... -mac 00:11:22:33:44:55`

#### 4. "Failed to read CA certificate"

**Problem:** CA certificate not found at the specified path.

**Solution:** Specify correct CA certificate path:
```bash
./device-discovery ... -ca-cert /path/to/ca-cert.pem
```

#### 5. Connection timeout in debug mode

**Problem:** Operation taking longer than the timeout duration.

**Solution:** Increase timeout:
```bash
./device-discovery ... -debug -timeout 15m
```

### Getting Help

Display help information:
```bash
./device-discovery -h
# or
./device-discovery --help
```

### Verbose Output

For debugging, the application prints:
- Auto-detected values
- Configuration summary
- Connection status
- Authentication progress

Example output:
```
Auto-detected serial number: ABCD1234
Auto-detected UUID: 12345678-1234-1234-1234-123456789012
Auto-detected IP address: 192.168.1.100
Device Discovery Configuration:
  Onboarding Manager: obm.example.com:50051
  ...
Starting gRPC client without timeout
Edge node registered. Waiting for the edge node to become ready for onboarding...
Credentials written successfully.
Device discovery done
```

## Support

For additional help or to report issues, please refer to the project documentation or contact the infrastructure team.
