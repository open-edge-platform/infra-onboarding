# Device Discovery Modes - Executive Summary

## Quick Overview

The device-discovery application has **TWO distinct operational modes**:

### 1. **Non-Interactive Mode** (Automatic/Streaming)
- **When**: Device is pre-registered in the system
- **What**: Automatic onboarding without human interaction
- **How**: gRPC streaming, polls until device is ready
- **Auth**: Server provides credentials automatically

### 2. **Interactive Mode** (Manual/Fallback)
- **When**: Device is NOT found in the system (404/NotFound)
- **What**: Manual registration requiring user credentials
- **How**: User enters username/password via TTY, then JWT-based registration
- **Auth**: User provides Keycloak credentials manually

## Key Findings

### 🔍 **Where Modes Are Used**

| Location | Function | Purpose |
|----------|----------|---------|
| `cmd/device-discovery/main.go` | `grpcClient()` | Mode detection and switching |
| `internal/client/client.go` | `GrpcStreamClient()` | Non-interactive streaming |
| `internal/client/client.go` | `GrpcInfraOnboardNodeJWT()` | Interactive unary call |
| `internal/client/client.go` | `RetryInfraOnboardNode()` | Interactive retry logic |
| `cmd/device-discovery/main.go` | `runClientAuthScript()` | Script execution for interactive |
| `client-auth.sh` (embedded) | - | Interactive credential collection |

### 📁 **Current File Organization Issues**

1. **Mixed responsibilities** - Both modes in `internal/client/client.go`
2. **Main.go complexity** - Mode switching + script execution + orchestration
3. **No clear boundaries** - Shared code without dedicated package
4. **Embedded script** - `client-auth.sh` embedded as `[]byte` in binary

### ✅ **Recommended Segregation**

Create dedicated packages for each mode:

```
internal/
  ├── mode/
  │   ├── orchestrator.go              # Mode selection logic
  │   ├── noninteractive/              # Non-interactive mode
  │   │   ├── client.go                # Streaming client
  │   │   └── handler.go               # State handling
  │   └── interactive/                 # Interactive mode
  │       ├── client.go                # JWT client
  │       ├── script.go                # Script execution
  │       └── retry.go                 # Retry logic
  └── connection/                      # Shared connection utilities
      └── grpc.go                      # createSecureConnection()
```

### 🎯 **Benefits of Segregation**

1. **Clear separation** - Each mode in its own package
2. **Better testability** - Test modes independently
3. **Easier maintenance** - Changes to one mode don't affect the other
4. **Improved documentation** - Code structure matches conceptual model
5. **Future extensibility** - Easy to add new modes or modify existing ones

## Mode Decision Flow

```
Start → Try Non-Interactive
          ↓
    Device Found?
      ↙         ↘
    YES          NO (404)
     ↓            ↓
Non-Interactive  Interactive
  (Automatic)    (Manual)
```

## Files Created by Each Mode

### Non-Interactive Files:
- `/dev/shm/io-client-id`
- `/dev/shm/io-client-secret`
- `/dev/shm/project_id`
- `/dev/shm/idp_access_token`
- `/dev/shm/release_token`

### Interactive Files:
- `/dev/shm/idp_access_token`
- `/dev/shm/release_token`
- `/dev/shm/project_id`
- `/var/log/client-auth/client-auth.log`

## Next Steps

1. Review `MODES_ANALYSIS.md` for detailed analysis
2. Review `MODES_FLOW_DIAGRAM.md` for visual flows
3. Consider implementing the proposed package structure
4. Add unit tests for each mode separately
5. Update documentation to reflect mode separation

## Related Documentation

- `MODES_ANALYSIS.md` - Comprehensive analysis with implementation details
- `MODES_FLOW_DIAGRAM.md` - Visual diagrams and flow charts
- `README.md` - General usage documentation
- `CLI_GUIDE.md` - Command-line interface guide
