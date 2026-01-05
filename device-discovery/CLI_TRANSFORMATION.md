# CLI Transformation Summary

This document summarizes the transformation of device-discovery from an environment-variable-based application to a full-featured CLI utility.

## Overview

The device-discovery application has been transformed from reading configuration via environment variables and kernel arguments to a modern CLI tool with command-line flags.

## Key Changes

### 1. Configuration Method

**Before:**
- Environment variables (`onboarding_manager_svc`, `OBM_PORT`, etc.)
- Kernel command-line arguments (`worker_id`, `DEBUG`, `TIMEOUT`)
- Configuration file at `/etc/hook/env_config`

**After:**
- Command-line flags (`-obm-svc`, `-obm-port`, etc.)
- Auto-detection of system information
- No external configuration files required

### 2. New CLI Flags

#### Required Flags
- `-obm-svc` - Onboarding manager service address
- `-obs-svc` - Onboarding stream service address
- `-obm-port` - Onboarding manager port
- `-keycloak-url` - Keycloak URL
- `-mac` - MAC address (or use `-auto-detect`)

#### Optional Flags
- `-serial` - Serial number (auto-detected if omitted)
- `-uuid` - System UUID (auto-detected if omitted)
- `-ip` - IP address (auto-detected if omitted)
- `-auto-detect` - Auto-detect all system info
- `-extra-hosts` - Additional host mappings
- `-ca-cert` - CA certificate path
- `-debug` - Enable debug mode
- `-timeout` - Debug timeout duration

### 3. Auto-Detection Features

The application can now automatically detect:
- **MAC Address** - Primary network interface MAC
- **Serial Number** - Using dmidecode
- **UUID** - Using dmidecode
- **IP Address** - From the specified MAC address

### 4. Code Changes

#### main.go Restructure
```go
// Before: Read environment variables
envVars, err := config.ReadEnvVars(requiredVars, optionalVars)

// After: Parse CLI flags
cfg := parseCLIFlags()
```

#### New CLIConfig Structure
```go
type CLIConfig struct {
    ObmSvc       string
    ObsSvc       string
    ObmPort      int
    KeycloakURL  string
    MacAddr      string
    SerialNumber string
    UUID         string
    IPAddress    string
    ExtraHosts   string
    CaCertPath   string
    Debug        bool
    Timeout      time.Duration
    AutoDetect   bool
}
```

#### New Functions Added
- `parseCLIFlags()` - Parse command-line arguments
- `printUsage()` - Display comprehensive help
- `validateConfig()` - Validate required flags
- `autoDetectSystemInfo()` - Auto-detect system information

### 5. Enhanced sysinfo Package

Added new function:
```go
func GetPrimaryMAC() (string, error)
```

This function automatically detects the primary network interface MAC address.

### 6. Updated Documentation

Created/Updated:
- `README.md` - Updated with CLI usage
- `CLI_GUIDE.md` - Comprehensive CLI guide
- Help text with examples (`./app -h`)

## Usage Comparison

### Before (Environment Variables + Kernel Args)

1. Set environment variables:
```bash
export onboarding_manager_svc=obm.example.com
export onboarding_stream_svc=obs.example.com
export OBM_PORT=50051
export KEYCLOAK_URL=keycloak.example.com
```

2. Boot with kernel arguments:
```
worker_id=00:11:22:33:44:55 DEBUG=false TIMEOUT=5m
```

3. Run:
```bash
./app
```

### After (CLI Flags)

Single command with all configuration:
```bash
./app \
  -obm-svc obm.example.com \
  -obs-svc obs.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -auto-detect
```

## Benefits

### 1. Ease of Use
- **Single command** with all configuration
- **No external dependencies** on environment files or kernel arguments
- **Clear, self-documenting** interface

### 2. Flexibility
- **Auto-detection** for common scenarios
- **Manual override** for specific values
- **Hybrid approach** - specify some, auto-detect others

### 3. Better User Experience
- **Comprehensive help** with `-h` flag
- **Usage examples** built into help text
- **Validation** with clear error messages
- **Configuration summary** displayed before execution

### 4. Maintainability
- **Standard Go flag package** - no external dependencies
- **Type-safe** flag parsing
- **Clear structure** with CLIConfig struct

### 5. Scriptability
- Easy to integrate into shell scripts
- Can be wrapped with environment variables if needed
- Works well in containers and automation

## Migration Path

For users migrating from the old method:

### Option 1: Wrapper Script
Create a wrapper that reads environment variables and converts to flags:

```bash
#!/bin/bash
# legacy-wrapper.sh

# Read old environment variables
source /etc/hook/env_config

# Convert to new CLI flags
./app \
  -obm-svc "$onboarding_manager_svc" \
  -obs-svc "$onboarding_stream_svc" \
  -obm-port "$OBM_PORT" \
  -keycloak-url "$KEYCLOAK_URL" \
  -mac "$worker_id" \
  ${EXTRA_HOSTS:+-extra-hosts "$EXTRA_HOSTS"} \
  ${DEBUG:+-debug}
```

### Option 2: Direct Migration
Update automation scripts to use new CLI flags directly:

```bash
# Old approach
export onboarding_manager_svc=obm.example.com
./app

# New approach
./app -obm-svc obm.example.com -obs-svc obs.example.com -obm-port 50051 -keycloak-url keycloak.example.com -auto-detect
```

## Backward Compatibility

### What's Preserved
- ✅ Same binary output (`app`)
- ✅ Same runtime behavior
- ✅ Same gRPC communication
- ✅ Same authentication flow
- ✅ Same Docker image structure

### What's Changed
- ❌ No longer reads environment variables
- ❌ No longer parses kernel arguments
- ❌ No longer reads `/etc/hook/env_config`
- ✅ Now uses CLI flags
- ✅ Auto-detection capabilities added

## Testing

The test suite has been updated:
- `TestAutoDetectSystemInfo()` - Tests auto-detection
- `TestValidateConfig()` - Tests configuration validation
- `FuzzTestDeviceDiscovery()` - Fuzz testing with various inputs

Run tests:
```bash
make test
```

## Examples

### Common Use Cases

#### 1. Production Deployment
```bash
./app -obm-svc obm.prod.example.com -obs-svc obs.prod.example.com \
      -obm-port 50051 -keycloak-url keycloak.prod.example.com -auto-detect
```

#### 2. Development/Testing
```bash
./app -obm-svc localhost -obs-svc localhost -obm-port 50051 \
      -keycloak-url localhost:8080 -mac 00:11:22:33:44:55 \
      -serial TEST-001 -uuid test-uuid -ip 127.0.0.1 -debug -timeout 10m
```

#### 3. Container Deployment
```bash
docker run device-discovery:latest \
  -obm-svc obm.example.com -obs-svc obs.example.com \
  -obm-port 50051 -keycloak-url keycloak.example.com -auto-detect
```

## Getting Help

```bash
# Display help
./app -h

# View CLI guide
cat CLI_GUIDE.md

# View README
cat README.md
```

## Next Steps

Consider future enhancements:
1. **Configuration file support** - YAML/JSON config files as alternative to flags
2. **Environment variable fallback** - Support env vars as fallback for flags
3. **Multiple output formats** - JSON, YAML output for automation
4. **Verbose/quiet modes** - Control output verbosity
5. **Dry-run mode** - Validate configuration without executing

## Conclusion

The transformation to a CLI utility makes device-discovery more user-friendly, flexible, and maintainable while preserving all core functionality. The auto-detection features reduce manual configuration, and the comprehensive help system makes it easier for new users to get started.
