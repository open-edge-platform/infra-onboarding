# Mode Segregation - Quick Reference Card

## 🎯 What Was Done

Refactored device-discovery to provide **clear demarcation between Interactive and Non-Interactive modes**.

## 📦 New Package Structure

```
internal/
├── connection/          ← Shared gRPC utilities
│   └── grpc.go
├── mode/                ← Mode orchestration & implementations
│   ├── orchestrator.go       [Coordinates between modes]
│   ├── noninteractive/       [🔵 Streaming mode]
│   │   └── client.go
│   └── interactive/          [🟢 JWT + TTY mode]
│       ├── client.go
│       └── script.go
├── auth/                ← Non-interactive auth only
├── config/              ← Shared configuration
├── parser/              ← Legacy kernel args
└── sysinfo/             ← Hardware detection
```

## 🔵 Non-Interactive Mode

**Location**: `internal/mode/noninteractive/client.go`

**Usage**:
```go
import "device-discovery/internal/mode/noninteractive"

client := noninteractive.NewClient(addr, port, mac, uuid, serial, ip, certPath)
result := client.Onboard(ctx)

if result.ShouldFallback {
    // Device not found - switch to interactive
} else if result.Error != nil {
    // Handle error
} else {
    // Success: use result.ClientID, result.ClientSecret, result.ProjectID
}
```

**Features**:
- gRPC streaming with `NonInteractiveOnboardingServiceClient`
- Exponential backoff polling (2s → 32s)
- Returns `StreamResult` with clear fallback flag
- Handles `NODE_STATE_REGISTERED` and `NODE_STATE_ONBOARDED`

## 🟢 Interactive Mode

**Location**: `internal/mode/interactive/`

**Usage**:
```go
import "device-discovery/internal/mode/interactive"

// Step 1: Execute TTY script for user authentication
err := interactive.ExecuteAuthScript(ctx, scriptBytes)

// Step 2: Create client with JWT token
client := interactive.NewClient(addr, port, mac, ip, uuid, serial, certPath, tokenPath)

// Step 3: Onboard with retry logic
err = client.OnboardWithRetry(ctx)  // Retries up to 3 times
```

**Features**:
- TTY-based user authentication via `client-auth.sh`
- JWT token with OAuth2 for gRPC
- Unary RPC with `InteractiveOnboardingServiceClient`
- Built-in retry logic (3 attempts with jitter)

## 🟠 Orchestrator (Recommended)

**Location**: `internal/mode/orchestrator.go`

**Usage** (simplest approach):
```go
import "device-discovery/internal/mode"

// Create config
cfg := mode.Config{
    ObmSvc:       "obm.example.com",
    ObsSvc:       "obs.example.com",
    ObmPort:      50051,
    KeycloakURL:  "keycloak.example.com",
    MacAddr:      "00:11:22:33:44:55",
    SerialNumber: "ABC123",
    UUID:         "12345678-...",
    IPAddress:    "192.168.1.100",
    CaCertPath:   "/path/to/cert.pem",
    AuthScript:   scriptBytes,
}

// Execute (handles mode selection automatically)
orchestrator := mode.NewOrchestrator(cfg)
err := orchestrator.Execute(ctx)
```

**What it does**:
1. Tries non-interactive mode first
2. If device not found (404), automatically falls back to interactive
3. Completes authentication based on which mode succeeded
4. Returns single error or nil

## 🔄 Mode Flow

```
Start
  ↓
Try Non-Interactive (streaming)
  ↓
Device Found?
  ├─ YES → Complete non-interactive auth → Success ✅
  └─ NO (404) → Fall back to Interactive
                  ↓
                Execute TTY script
                  ↓
                JWT authentication
                  ↓
                Create node → Success ✅
```

## 📁 Files Created by Each Mode

### Non-Interactive
- `/dev/shm/io-client-id`
- `/dev/shm/io-client-secret`
- `/dev/shm/project_id`
- `/dev/shm/idp_access_token`
- `/dev/shm/release_token`

### Interactive
- `/dev/shm/idp_access_token` (from script)
- `/dev/shm/release_token` (from script)
- `/dev/shm/project_id` (from CreateNodes)

## 🧪 Testing

### Unit Tests (per mode)
```bash
go test ./internal/mode/noninteractive/...   # Non-interactive tests
go test ./internal/mode/interactive/...      # Interactive tests
go test ./internal/mode/...                  # Orchestrator tests
```

### Integration Test
```bash
go test ./cmd/device-discovery/...           # Full flow test
```

## 📚 Documentation

| Document | Purpose |
|----------|---------|
| `REFACTORING_COMPLETE.md` | **Complete refactoring details** |
| `REFACTORING_VISUAL.md` | **Visual diagrams and flows** |
| `MODES_ANALYSIS.md` | Original analysis and plan |
| `MODES_SUMMARY.md` | Executive summary |
| `CODE_OWNERSHIP_BY_MODE.md` | Function-by-function breakdown |
| `INDEX.md` | Navigation guide |

## ✅ What Changed

| Before | After |
|--------|-------|
| Mixed modes in `client.go` | Separate packages per mode |
| 265-line monolithic file | ~180 lines per focused file |
| Mode switching in main | Orchestrator handles it |
| Unclear ownership | Clear package boundaries |
| Hard to test | Easy unit testing |

## 🚀 Quick Start

### For New Code
Use the orchestrator - it's the simplest:
```go
orch := mode.NewOrchestrator(cfg)
err := orch.Execute(ctx)
```

### For Direct Mode Access
If you need mode-specific behavior:
```go
// Non-interactive only
client := noninteractive.NewClient(...)
result := client.Onboard(ctx)

// Interactive only
client := interactive.NewClient(...)
err := client.OnboardWithRetry(ctx)
```

## 💡 Key Benefits

1. **Clear Separation** - Each mode in its own package
2. **Type Safety** - `StreamResult` instead of multiple return values
3. **Testability** - Test each mode independently
4. **Maintainability** - Changes isolated per mode
5. **Documentation** - Code matches conceptual model

## 🎓 Learning Path

1. Read `MODES_SUMMARY.md` - Understand the two modes
2. Read `REFACTORING_VISUAL.md` - See the structure
3. Check `internal/mode/orchestrator.go` - See how modes coordinate
4. Explore `noninteractive/client.go` - Streaming implementation
5. Explore `interactive/client.go` - JWT implementation

## 🔗 Import Paths

```go
import (
    "device-discovery/internal/mode"                    // Orchestrator
    "device-discovery/internal/mode/noninteractive"     // Non-interactive
    "device-discovery/internal/mode/interactive"        // Interactive
    "device-discovery/internal/connection"              // Shared utilities
)
```

---

**Remember**: The orchestrator handles everything. Use it unless you need direct mode control!
