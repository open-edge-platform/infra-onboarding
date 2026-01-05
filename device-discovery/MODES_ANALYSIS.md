# Device Discovery Modes Analysis

## Overview

The device-discovery application supports **two distinct onboarding modes**:

1. **Non-Interactive Mode** (Streaming/Automatic)
2. **Interactive Mode** (Manual/Fallback)

## Mode Detection and Flow

### Main Flow Decision Point
Located in: `cmd/device-discovery/main.go` → `grpcClient()`

```
Start
  ↓
GrpcStreamClient() → Attempts Non-Interactive Mode
  ↓
Returns (clientID, clientSecret, error, fallback)
  ↓
If fallback == true → Switch to Interactive Mode
If fallback == false && error == nil → Continue Non-Interactive Mode
If fallback == false && error != nil → Fail
```

---

## 1. Non-Interactive Mode (Streaming)

### Purpose
Automatic device onboarding when the device is pre-registered in the system.

### Key Characteristics
- **No human interaction required**
- Uses gRPC streaming protocol
- Device polls the server until onboarded
- Server provides `clientID` and `clientSecret` automatically
- Suitable for pre-registered devices

### Implementation Details

#### Files Involved
- `internal/client/client.go` → `GrpcStreamClient()`
- `cmd/device-discovery/main.go` → Main authentication flow

#### gRPC Service Used
```go
pb.NewNonInteractiveOnboardingServiceClient(conn)
```

#### Stream Endpoint
```go
stream := cli.OnboardNodeStream(ctx)
```

#### Request Structure
```go
&pb.OnboardNodeStreamRequest{
    MacId:     mac,
    Uuid:      uuid,
    Serialnum: serial,
    HostIp:    ipAddress,
}
```

#### Node States Handled
1. **NODE_STATE_REGISTERED** 
   - Device is registered but not ready
   - Implements exponential backoff polling (2s to 32s)
   - Waits for device to become ready

2. **NODE_STATE_ONBOARDED**
   - Device successfully onboarded
   - Receives `clientID`, `clientSecret`, `projectID`
   - Saves credentials to files

3. **NODE_STATE_UNSPECIFIED**
   - Unknown state → Error

#### Authentication Flow (Non-Interactive)
```
1. GrpcStreamClient() polls server
2. Server returns clientID + clientSecret
3. Save credentials to files:
   - /dev/shm/io-client-id
   - /dev/shm/io-client-secret
4. ClientAuth() exchanges credentials for tokens:
   - Keycloak access token
   - Release token
5. Save tokens to files:
   - /dev/shm/idp_access_token
   - /dev/shm/release_token
```

#### Files Written
- `/dev/shm/io-client-id`
- `/dev/shm/io-client-secret`
- `/dev/shm/project_id`
- `/dev/shm/idp_access_token`
- `/dev/shm/release_token`

---

## 2. Interactive Mode (Manual Fallback)

### Purpose
Manual device onboarding when the device is NOT pre-registered in the system.

### Key Characteristics
- **Requires human interaction** (username/password input via TTY)
- Triggered when server returns `codes.NotFound`
- User provides Keycloak credentials manually
- Device uses JWT token for authentication
- Suitable for ad-hoc device registration

### Implementation Details

#### Files Involved
- `client-auth.sh` → Interactive credential collection script
- `internal/client/client.go` → `GrpcInfraOnboardNodeJWT()`, `RetryInfraOnboardNode()`
- `cmd/device-discovery/main.go` → `runClientAuthScript()`, fallback logic

#### Trigger Condition
```go
if resp.Status.Code == int32(codes.NotFound) {
    fallback = true
    return "", "", fmt.Errorf(resp.Status.Message), fallback
}
```

#### gRPC Service Used
```go
pb.NewInteractiveOnboardingServiceClient(conn)
```

#### Endpoint
```go
nodeResponse := cli.CreateNodes(ctx, nodeRequest)
```

#### Request Structure
```go
&pb.CreateNodesRequest{
    Payload: []*pb.NodeData{
        {
            Hwdata: []*pb.HwData{
                {
                    MacId:     mac,
                    SutIp:     ip,
                    Uuid:      uuid,
                    Serialnum: serial,
                },
            },
        },
    },
}
```

#### Authentication Flow (Interactive)
```
1. GrpcStreamClient() fails with NotFound
2. Set fallback = true
3. Execute client-auth.sh:
   - Prompts user for username/password on TTY
   - Authenticates with Keycloak
   - Retrieves access token
   - Retrieves release token
4. Retry logic (max 3 attempts):
   - GrpcInfraOnboardNodeJWT() with JWT token
   - Creates node in system
   - Receives projectID
5. Save projectID to file
```

#### Script Behavior (client-auth.sh)
- **TTY Devices Used**: `ttyS0`, `ttyS1`, `tty0`
- **Max Attempts**: 3
- **Wait Time per Attempt**: 50 seconds (10 checks × 5 seconds)
- **Prompts User For**:
  - Username (min 3 chars)
  - Password (min 3 chars, hidden input)
- **Authentication Steps**:
  1. Read credentials from TTY
  2. POST to Keycloak token endpoint
  3. Validate access token
  4. GET release token from release server
  5. Save both tokens

#### Files Written
- `/dev/shm/idp_access_token`
- `/dev/shm/release_token`
- `/dev/shm/project_id`
- `/idp_username` (temporary)
- `/idp_password` (temporary)
- `/var/log/client-auth/client-auth.log`

#### Retry Logic
- **Max Retries**: 3
- **Delay**: 2 seconds + random jitter (0-1000ms)
- **Function**: `RetryInfraOnboardNode()`

---

## Mode Comparison Matrix

| Aspect | Non-Interactive Mode | Interactive Mode |
|--------|---------------------|------------------|
| **Trigger** | Device pre-registered | Device not found (404) |
| **User Input** | None | Username + Password |
| **gRPC Service** | NonInteractiveOnboardingServiceClient | InteractiveOnboardingServiceClient |
| **Protocol** | Streaming (OnboardNodeStream) | Unary RPC (CreateNodes) |
| **Authentication** | Client credentials (ID + Secret) | JWT token (user credentials) |
| **Polling** | Yes (exponential backoff) | No (direct call with retry) |
| **Credentials Source** | Server-generated | User-provided |
| **TTY Interaction** | No | Yes (ttyS0, ttyS1, tty0) |
| **External Script** | No | Yes (client-auth.sh) |
| **Files Created** | 5 files | 3 files (+2 temp) |
| **Retry Count** | Unlimited (until timeout) | 3 attempts |
| **Use Case** | Production automation | Manual registration |

---

## Code Organization Issues

### Current Problems

1. **Mixed Responsibilities in client.go**
   - Both non-interactive and interactive logic in same file
   - 3 functions handling different concerns:
     - `GrpcStreamClient()` → Non-interactive streaming
     - `GrpcInfraOnboardNodeJWT()` → Interactive unary call
     - `RetryInfraOnboardNode()` → Interactive retry logic

2. **Main.go Complexity**
   - Mode switching logic in `grpcClient()`
   - Script execution in `runClientAuthScript()`
   - Mixed concerns: CLI parsing, mode orchestration, script execution

3. **Shared Code Without Clear Boundaries**
   - `createSecureConnection()` used by both modes
   - No clear package structure for mode-specific logic
   - Authentication split between `auth.go` and `client-auth.sh`

4. **Embedded Script**
   - `client-auth.sh` embedded in binary as `[]byte`
   - Hard to maintain and test separately

---

## Recommended Refactoring

### Proposed Package Structure

```
device-discovery/
├── cmd/
│   └── device-discovery/
│       └── main.go                    # Minimal main, delegates to orchestrator
├── internal/
│   ├── auth/
│   │   └── auth.go                    # Common auth utilities
│   ├── config/
│   │   └── config.go                  # Configuration and constants
│   ├── connection/                    # NEW: Shared connection logic
│   │   └── grpc.go                    # createSecureConnection, common setup
│   ├── mode/                          # NEW: Mode orchestrator
│   │   ├── orchestrator.go            # Mode detection and switching
│   │   ├── noninteractive/            # NEW: Non-interactive mode package
│   │   │   ├── client.go              # GrpcStreamClient
│   │   │   └── handler.go             # Non-interactive flow handler
│   │   └── interactive/               # NEW: Interactive mode package
│   │       ├── client.go              # GrpcInfraOnboardNodeJWT
│   │       ├── script.go              # Script execution logic
│   │       └── retry.go               # Retry logic
│   ├── parser/
│   │   └── parser.go                  # Kernel argument parsing
│   └── sysinfo/
│       └── sysinfo.go                 # System information
└── scripts/
    └── client-auth.sh                 # External script (not embedded)
```

### Benefits of Segregation

1. **Clear Separation of Concerns**
   - Each mode in its own package
   - Mode-specific logic isolated
   - Easy to test independently

2. **Better Maintainability**
   - Changes to one mode don't affect the other
   - Clear file naming indicates responsibility
   - Reduced coupling

3. **Improved Testability**
   - Mock mode interfaces easily
   - Test streaming vs unary separately
   - Test retry logic in isolation

4. **Easier Onboarding**
   - Developers can understand one mode at a time
   - Documentation maps directly to code structure
   - Clear entry points

5. **Future Extensibility**
   - Easy to add new modes
   - Plugin architecture possible
   - Mode selection can be configurable

---

## Mode Selection Logic (Proposed)

### orchestrator.go
```go
type ModeOrchestrator struct {
    config *CLIConfig
}

func (o *ModeOrchestrator) Execute(ctx context.Context) error {
    // Try non-interactive first
    if err := o.tryNonInteractive(ctx); err != nil {
        if isNotFoundError(err) {
            // Fall back to interactive
            return o.runInteractive(ctx)
        }
        return err
    }
    return nil
}

func (o *ModeOrchestrator) tryNonInteractive(ctx context.Context) error {
    client := noninteractive.NewClient(o.config)
    return client.Onboard(ctx)
}

func (o *ModeOrchestrator) runInteractive(ctx context.Context) error {
    client := interactive.NewClient(o.config)
    return client.Onboard(ctx)
}
```

---

## Implementation Checklist

### Phase 1: Create New Packages
- [ ] Create `internal/connection/` package
- [ ] Create `internal/mode/` package
- [ ] Create `internal/mode/noninteractive/` package
- [ ] Create `internal/mode/interactive/` package

### Phase 2: Move Non-Interactive Code
- [ ] Move `GrpcStreamClient()` to `mode/noninteractive/client.go`
- [ ] Create `noninteractive.Handler` interface
- [ ] Move streaming logic and state handling
- [ ] Add tests for non-interactive mode

### Phase 3: Move Interactive Code
- [ ] Move `GrpcInfraOnboardNodeJWT()` to `mode/interactive/client.go`
- [ ] Move `RetryInfraOnboardNode()` to `mode/interactive/retry.go`
- [ ] Move `runClientAuthScript()` to `mode/interactive/script.go`
- [ ] Add tests for interactive mode

### Phase 4: Create Orchestrator
- [ ] Implement `mode/orchestrator.go`
- [ ] Define mode interfaces
- [ ] Implement mode selection logic
- [ ] Update main.go to use orchestrator

### Phase 5: Extract Shared Code
- [ ] Move `createSecureConnection()` to `internal/connection/grpc.go`
- [ ] Create common interfaces for both modes
- [ ] Consolidate configuration handling

### Phase 6: Update Documentation
- [ ] Update README with new structure
- [ ] Document mode selection criteria
- [ ] Add examples for each mode
- [ ] Update API documentation

---

## Testing Strategy

### Non-Interactive Mode Tests
```go
// Test streaming flow
TestStreamingOnboarding_Success()
TestStreamingOnboarding_RegisteredState()
TestStreamingOnboarding_OnboardedState()
TestStreamingOnboarding_Backoff()

// Test error handling
TestStreamingOnboarding_NotFound()
TestStreamingOnboarding_StreamClosed()
TestStreamingOnboarding_Timeout()
```

### Interactive Mode Tests
```go
// Test interactive flow
TestInteractiveOnboarding_Success()
TestInteractiveOnboarding_WithRetry()
TestInteractiveOnboarding_MaxRetries()

// Test script execution
TestScriptExecution_Success()
TestScriptExecution_Timeout()
TestScriptExecution_InvalidCredentials()
```

### Orchestrator Tests
```go
TestOrchestrator_NonInteractiveSuccess()
TestOrchestrator_FallbackToInteractive()
TestOrchestrator_NonInteractiveFail()
```

---

## Summary

### Current State
- Two modes implemented but intermixed in single files
- Hard to understand which code belongs to which mode
- Testing requires understanding both modes simultaneously
- Mode switching logic embedded in main flow

### Proposed State
- Clear package separation: `noninteractive/` and `interactive/`
- Orchestrator handles mode selection and fallback
- Shared code in dedicated `connection/` package
- Each mode can be developed and tested independently
- Better alignment with documentation and architecture

### Migration Impact
- **Low risk**: Refactoring without changing behavior
- **High value**: Improved maintainability and clarity
- **Testability**: Easier to write comprehensive tests
- **Documentation**: Code structure matches conceptual model
