# Device Discovery

Device Discovery is a Go CLI application that handles device onboarding in the infrastructure platform.

## Project Structure

This project follows the standard Go project layout:

```
device-discovery/
├── cmd/
│   └── device-discovery/          # Main application entry point
│       ├── main.go                # Main function and CLI logic
│       └── main_test.go           # Integration tests
├── internal/                      # Private application code
│   ├── auth/                      # Authentication logic
│   │   └── auth.go                # Client authentication with Keycloak
│   ├── client/                    # gRPC client implementation
│   │   └── client.go              # Onboarding service client
│   ├── config/                    # Configuration and utilities
│   │   └── config.go              # File operations and utilities
│   ├── parser/                    # Kernel argument parsing (legacy)
│   │   └── parser.go              # Command line parser
│   └── sysinfo/                   # System information
│       └── sysinfo.go             # Hardware info retrieval (UUID, serial, IP, MAC)
├── build.sh                       # Build script
├── client-auth.sh                 # Authentication helper script
├── Dockerfile                     # Container image definition
├── go.mod                         # Go module definition
├── go.sum                         # Go module checksums
├── Makefile                       # Build automation
└── README.md                      # This file

```

## Building

### Using Make (recommended)

```bash
# Build the binary
make build

# Run tests
make test

# Format code
make fmt

# Run linter
make vet

# Clean build artifacts
make clean

# Build Docker image
make docker-build

# View all available targets
make help
```

### Using build.sh

```bash
./build.sh
```

### Manual build

```bash
go build -o app ./cmd/device-discovery
```

## Usage

Device Discovery now operates as a CLI utility with command-line flags for configuration.

### Basic Usage

```bash
./app [OPTIONS]
```

### Required Flags

- `-obm-svc` - Onboarding manager service address
- `-obs-svc` - Onboarding stream service address  
- `-obm-port` - Onboarding manager port
- `-keycloak-url` - Keycloak authentication URL
- `-mac` - MAC address of the device (required unless using `-auto-detect`)

### Optional Flags

**Device Information:**
- `-serial` - Serial number (auto-detected if not provided)
- `-uuid` - System UUID (auto-detected if not provided)
- `-ip` - IP address (auto-detected from MAC if not provided)

**Auto-Detection:**
- `-auto-detect` - Auto-detect all system information (MAC, serial, UUID, IP)

**Additional Options:**
- `-extra-hosts` - Additional host mappings (comma-separated: 'host1:ip1,host2:ip2')
- `-ca-cert` - Path to CA certificate (default: /etc/idp/server_cert.pem)
- `-debug` - Enable debug mode with timeout
- `-timeout` - Timeout duration for debug mode (default: 5m0s)

### Examples

#### 1. Auto-detect all system information
```bash
./app -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -auto-detect
```

#### 2. Specify MAC address, auto-detect other info
```bash
./app -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -mac 00:11:22:33:44:55
```

#### 3. Fully manual configuration
```bash
./app -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -mac 00:11:22:33:44:55 \
      -serial ABC123 \
      -uuid 12345678-1234-1234-1234-123456789012 \
      -ip 192.168.1.100
```

#### 4. With debug mode and extra hosts
```bash
./app -obm-svc obm.example.com \
      -obs-svc obs.example.com \
      -obm-port 50051 \
      -keycloak-url keycloak.example.com \
      -auto-detect \
      -debug \
      -timeout 10m \
      -extra-hosts "registry.local:10.0.0.1,api.local:10.0.0.2"
```

### Getting Help

```bash
./app -h
# or
./app --help
```

## Package Overview

### cmd/device-discovery
Main application entry point. Contains the `main()` function, CLI flag parsing, and orchestration logic.

### internal/auth
Handles client authentication with Keycloak and token management:
- JWT access token retrieval
- Release token fetching
- Certificate-based authentication

### internal/client
gRPC client for communicating with the onboarding manager:
- Stream-based non-interactive onboarding
- Interactive onboarding with JWT
- Retry logic with exponential backoff

### internal/config
Configuration management and utility functions:
- File I/O operations
- Host file updates
- Temporary script creation
- Constants for file paths

### internal/parser
Kernel command line argument parsing (legacy support).

### internal/sysinfo
System information retrieval using dmidecode and network interfaces:
- Hardware serial number
- System UUID
- IP address lookup by MAC
- Primary MAC address detection

## Auto-Detection Features

The application can automatically detect system information:

1. **Serial Number** - Retrieved using `dmidecode -s system-serial-number`
2. **UUID** - Retrieved using `dmidecode -s system-uuid`
3. **MAC Address** - Automatically detects the primary network interface MAC
4. **IP Address** - Automatically detected from the specified MAC address

When using `-auto-detect`, all system information is automatically gathered. Individual fields can also be auto-detected by omitting the corresponding flag.

## Configuration Methods

The application supports multiple configuration methods:

### 1. CLI Flags (Recommended)
Use command-line flags for explicit configuration:
```bash
./app -obm-svc obm.example.com -obm-port 50051 -mac 00:11:22:33:44:55 ...
```

### 2. Auto-Detection
Let the application detect system information:
```bash
./app -obm-svc obm.example.com -obm-port 50051 -auto-detect
```

### 3. Hybrid Approach
Specify some values manually, auto-detect others:
```bash
./app -obm-svc obm.example.com -obm-port 50051 -mac 00:11:22:33:44:55
# Serial, UUID, and IP will be auto-detected
```

## Testing

Run the fuzz tests:

```bash
make test
```

Or directly with go:

```bash
go test -v ./...
```

## Docker

Build the Docker image:

```bash
make docker-build
```

Or manually:

```bash
./build.sh
docker build -t device-discovery:latest .
```

## License

SPDX-License-Identifier: Apache-2.0
