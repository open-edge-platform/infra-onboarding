# Device Discovery CLI - Quick Reference

## Usage
```bash
./app [OPTIONS]
```

## Required Flags
```bash
-obm-svc string       # Onboarding manager service address
-obs-svc string       # Onboarding stream service address
-obm-port int         # Onboarding manager port
-keycloak-url string  # Keycloak authentication URL
-mac string           # MAC address (or use -auto-detect)
```

## Quick Commands

### Auto-detect Everything
```bash
./app -obm-svc OBM_HOST -obs-svc OBS_HOST -obm-port PORT -keycloak-url KC_URL -auto-detect
```

### Specify MAC, Auto-detect Rest
```bash
./app -obm-svc OBM_HOST -obs-svc OBS_HOST -obm-port PORT -keycloak-url KC_URL -mac MAC_ADDR
```

### Full Manual
```bash
./app -obm-svc OBM_HOST -obs-svc OBS_HOST -obm-port PORT -keycloak-url KC_URL \
      -mac MAC -serial SERIAL -uuid UUID -ip IP
```

### With Debug Mode
```bash
./app -obm-svc OBM_HOST -obs-svc OBS_HOST -obm-port PORT -keycloak-url KC_URL \
      -auto-detect -debug -timeout 10m
```

### With Extra Hosts
```bash
./app -obm-svc OBM_HOST -obs-svc OBS_HOST -obm-port PORT -keycloak-url KC_URL \
      -auto-detect -extra-hosts "host1:ip1,host2:ip2"
```

## Optional Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-serial` | string | auto | Serial number |
| `-uuid` | string | auto | System UUID |
| `-ip` | string | auto | IP address |
| `-auto-detect` | bool | false | Auto-detect all system info |
| `-extra-hosts` | string | - | Additional host mappings |
| `-ca-cert` | string | /etc/idp/server_cert.pem | CA certificate path |
| `-debug` | bool | false | Enable debug mode |
| `-timeout` | duration | 5m0s | Debug timeout |

## Help
```bash
./app -h
./app --help
```

## Auto-Detection

When you omit flags, the application auto-detects:
- **Serial Number** - via dmidecode (requires root)
- **UUID** - via dmidecode (requires root)
- **IP Address** - from specified MAC address
- **MAC Address** - with `-auto-detect` flag

## Examples by Scenario

### Scenario 1: New Device (Auto-detect)
```bash
sudo ./app \
  -obm-svc obm.example.com \
  -obs-svc obs.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -auto-detect
```

### Scenario 2: Known MAC Address
```bash
./app \
  -obm-svc obm.example.com \
  -obs-svc obs.example.com \
  -obm-port 50051 \
  -keycloak-url keycloak.example.com \
  -mac 00:11:22:33:44:55
```

### Scenario 3: Development/Testing
```bash
./app \
  -obm-svc localhost \
  -obs-svc localhost \
  -obm-port 50051 \
  -keycloak-url localhost:8080 \
  -mac 00:11:22:33:44:55 \
  -serial TEST-001 \
  -uuid test-uuid-1234 \
  -ip 127.0.0.1 \
  -debug
```

## Common Issues

| Issue | Solution |
|-------|----------|
| "Missing required flags" | Provide all required flags or use `-auto-detect` |
| "Failed to auto-detect serial/UUID" | Run with `sudo` (dmidecode needs root) |
| "No suitable network interface" | Manually specify `-mac` |
| "Failed to read CA certificate" | Use `-ca-cert` with correct path |

## Tips

- Use `-auto-detect` for quickest setup
- Specify `-mac` if you know the interface to use
- Use `-debug` and `-timeout` for troubleshooting
- Run with `sudo` if auto-detecting serial/UUID
- Check `-h` for latest flag options

## Documentation

- **README.md** - Project overview and basic usage
- **CLI_GUIDE.md** - Comprehensive CLI guide with examples
- **CLI_TRANSFORMATION.md** - Migration guide from old version
- **MIGRATION.md** - Code restructuring details

## Build & Test

```bash
make build       # Build binary
make test        # Run tests
make clean       # Clean artifacts
make help        # Show all make targets
```
