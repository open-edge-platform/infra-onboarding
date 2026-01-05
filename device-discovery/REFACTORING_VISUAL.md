# Mode Segregation - Visual Guide

## Before & After Comparison

### BEFORE: Mixed Responsibilities ❌
```
┌─────────────────────────────────────────────────────────────┐
│                    internal/client/                          │
│                      client.go (265 lines)                   │
│                                                              │
│  🟡 createSecureConnection()      [Shared - 29 lines]       │
│  🔵 GrpcStreamClient()            [Non-Interactive - 101]   │
│  🟢 GrpcInfraOnboardNodeJWT()     [Interactive - 81 lines]  │
│  🟢 RetryInfraOnboardNode()       [Interactive - 20 lines]  │
│                                                              │
│  ⚠️  Problem: All modes mixed in one file!                   │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│              cmd/device-discovery/main.go                    │
│                                                              │
│  🟠 grpcClient()                  [Mode Switching - 97 lines]│
│     ├─ Try non-interactive                                   │
│     ├─ Check fallback flag                                   │
│     ├─ If fallback: Interactive logic                        │
│     └─ Else: Non-interactive auth                            │
│                                                              │
│  �� runClientAuthScript()         [Interactive - 36 lines]   │
│                                                              │
│  ⚠️  Problem: Mode orchestration mixed with main logic!      │
└─────────────────────────────────────────────────────────────┘
```

### AFTER: Clear Segregation ✅
```
┌────────────────────────────────────────────────────────────────────┐
│                       internal/connection/                          │
│                          grpc.go (46 lines)                         │
│                                                                     │
│  🟡 CreateSecureConnection()    [Shared by both modes]             │
│                                                                     │
│  ✓ Benefit: Single source of truth for gRPC connections            │
└────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────┐
│              internal/mode/noninteractive/                          │
│                   client.go (180 lines)                             │
│                                                                     │
│  🔵 NON-INTERACTIVE MODE ONLY                                       │
│                                                                     │
│  • Client struct                                                    │
│  • NewClient()                                                      │
│  • Onboard(ctx) StreamResult                                        │
│  • StreamResult struct                                              │
│                                                                     │
│  ✓ Benefit: All streaming logic in one place                       │
└────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────┐
│                internal/mode/interactive/                           │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │              client.go (160 lines)                          │   │
│  │                                                             │   │
│  │  🟢 INTERACTIVE MODE - CLIENT                               │   │
│  │                                                             │   │
│  │  • Client struct                                            │   │
│  │  • NewClient()                                              │   │
│  │  • Onboard(ctx) error                                       │   │
│  │  • OnboardWithRetry(ctx) error                              │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │              script.go (57 lines)                           │   │
│  │                                                             │   │
│  │  🟢 INTERACTIVE MODE - SCRIPT EXECUTION                     │   │
│  │                                                             │   │
│  │  • ExecuteAuthScript(ctx, []byte) error                     │   │
│  │                                                             │   │
│  │  ✓ Handles TTY authentication                               │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  ✓ Benefit: All JWT & script logic isolated                        │
└────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────┐
│                  internal/mode/orchestrator.go                      │
│                        (180 lines)                                  │
│                                                                     │
│  🟠 MODE ORCHESTRATION                                              │
│                                                                     │
│  • Orchestrator struct                                              │
│  • NewOrchestrator(Config)                                          │
│  • Execute(ctx) error                                               │
│  • tryNonInteractiveMode(ctx) StreamResult                          │
│  • executeInteractiveMode(ctx) error                                │
│  • completeNonInteractiveAuth(clientID, secret) error               │
│                                                                     │
│  ✓ Benefit: Clean mode switching logic                             │
└────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────┐
│              cmd/device-discovery/main.go                           │
│                      (~300 lines)                                   │
│                                                                     │
│  deviceDiscovery(cfg) {                                             │
│      ctx := createContext()                                         │
│      orchestratorCfg := mode.Config{...}                            │
│      orchestrator := mode.NewOrchestrator(orchestratorCfg)          │
│      return orchestrator.Execute(ctx)  ← Simple!                    │
│  }                                                                  │
│                                                                     │
│  ✓ Benefit: Main is now just CLI parsing + orchestrator call       │
└────────────────────────────────────────────────────────────────────┘
```

## Execution Flow

### Non-Interactive Mode Success Path
```
main.go
  │
  ├─ deviceDiscovery(cfg)
  │    │
  │    ├─ Create context (with/without timeout)
  │    │
  │    └─ Create & Execute Orchestrator
  │         │
  │         └─ orchestrator.Execute(ctx)
  │              │
  │              ├─ tryNonInteractiveMode()
  │              │    │
  │              │    └─ noninteractive.Client.Onboard()
  │              │         │
  │              │         ├─ connection.CreateSecureConnection()
  │              │         │
  │              │         ├─ NewNonInteractiveOnboardingServiceClient
  │              │         │
  │              │         ├─ OnboardNodeStream() [streaming]
  │              │         │
  │              │         └─ Poll until NODE_STATE_ONBOARDED
  │              │              │
  │              │              └─ Return StreamResult {
  │              │                    ClientID, ClientSecret,
  │              │                    ProjectID, ShouldFallback=false
  │              │                  }
  │              │
  │              ├─ completeNonInteractiveAuth(clientID, secret)
  │              │    │
  │              │    ├─ Save credentials to files
  │              │    │
  │              │    ├─ auth.ClientAuth() [exchange for tokens]
  │              │    │
  │              │    └─ Save tokens to files
  │              │
  │              └─ Success! ✅
```

### Interactive Mode (Fallback) Path
```
main.go
  │
  ├─ deviceDiscovery(cfg)
  │    │
  │    └─ orchestrator.Execute(ctx)
  │         │
  │         ├─ tryNonInteractiveMode()
  │         │    │
  │         │    └─ noninteractive.Client.Onboard()
  │         │         │
  │         │         └─ Server returns codes.NotFound
  │         │              │
  │         │              └─ Return StreamResult {
  │         │                    ShouldFallback=true ⚠️
  │         │                    Error="device not found"
  │         │                  }
  │         │
  │         ├─ Detect fallback flag
  │         │
  │         └─ executeInteractiveMode()
  │              │
  │              ├─ interactive.ExecuteAuthScript()
  │              │    │
  │              │    ├─ Create temp file from embedded script
  │              │    │
  │              │    ├─ Execute /bin/sh script
  │              │    │
  │              │    ├─ Script prompts TTY for user/pass
  │              │    │
  │              │    ├─ Script authenticates with Keycloak
  │              │    │
  │              │    └─ Script saves access_token & release_token
  │              │
  │              ├─ interactive.Client.OnboardWithRetry()
  │              │    │
  │              │    └─ Retry loop (max 3 times):
  │              │         │
  │              │         ├─ Client.Onboard()
  │              │         │    │
  │              │         │    ├─ connection.CreateSecureConnection()
  │              │         │    │    + OAuth2 JWT credentials
  │              │         │    │
  │              │         │    ├─ NewInteractiveOnboardingServiceClient
  │              │         │    │
  │              │         │    ├─ CreateNodes() [unary RPC]
  │              │         │    │
  │              │         │    └─ Save ProjectID
  │              │         │
  │              │         └─ If error: sleep & retry
  │              │
  │              └─ Success! ✅
```

## Package Dependencies

```
cmd/device-discovery/main.go
    │
    ├───► internal/mode/
    │       │
    │       ├───► orchestrator.go
    │       │       │
    │       │       ├───► noninteractive/client.go
    │       │       │       │
    │       │       │       └───► connection/grpc.go
    │       │       │
    │       │       └───► interactive/
    │       │               ├───► client.go
    │       │               │       │
    │       │               │       └───► connection/grpc.go
    │       │               │
    │       │               └───► script.go
    │       │
    │       └───► Uses: auth/, config/
    │
    ├───► internal/config/
    ├───► internal/sysinfo/
    └───► internal/parser/
```

## Color Legend
- 🔵 **Non-Interactive Mode** - Automatic streaming onboarding
- 🟢 **Interactive Mode** - Manual TTY-based onboarding
- �� **Shared** - Used by both modes
- 🟠 **Orchestration** - Mode selection and coordination

## Key Improvements Summary

| Before | After |
|--------|-------|
| ❌ Mixed modes in client.go | ✅ Separate packages per mode |
| ❌ Mode switching in main | ✅ Orchestrator handles switching |
| ❌ 265-line monolithic file | ✅ ~180 lines per focused file |
| ❌ Unclear function ownership | ✅ Clear package boundaries |
| ❌ Hard to test modes separately | ✅ Easy unit testing per mode |
| ❌ Shared code duplicated | ✅ connection/ package for shared code |
| ❌ No type for mode results | ✅ StreamResult struct |
| ❌ Complex imports | ✅ Import what you need |
