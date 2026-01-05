# Rename: OnboardingOrchestrator → OnboardingController

## Summary
Renamed `OnboardingOrchestrator` to `OnboardingController` to avoid naming conflicts with the Tinkerbell workflow orchestration concepts used in the onboarding-manager service.

## Rationale

### Naming Conflict
The onboarding-manager service uses **Tinkerbell workflows** extensively for orchestrating provisioning tasks. Using "Orchestrator" in device-discovery could cause conceptual confusion:

- **onboarding-manager**: Uses Tinkerbell for workflow/task orchestration (complex multi-step provisioning)
- **device-discovery**: Coordinates between two onboarding modes (simpler mode routing)

### Why "Controller"?
- ✅ **Standard pattern**: Well-established in Go codebases (e.g., Kubernetes controllers)
- ✅ **Clear intent**: Controls/manages the onboarding flow without implying task orchestration
- ✅ **No conflicts**: Doesn't overlap with onboarding-manager terminology
- ✅ **Appropriate scope**: Reflects the coordination nature (not complex orchestration)

## Changes Made

### 1. Type Rename
```go
// Before
type OnboardingOrchestrator struct { ... }

// After
type OnboardingController struct { ... }
```

### 2. Constructor Rename
```go
// Before
func NewOnboardingOrchestrator(cfg Config) *OnboardingOrchestrator

// After
func NewOnboardingController(cfg Config) *OnboardingController
```

### 3. Method Receivers Updated
All methods now use `*OnboardingController` receiver:
- `Execute(ctx context.Context) error`
- `tryNonInteractiveMode(ctx context.Context) noninteractive.StreamResult`
- `completeNonInteractiveAuth(clientID, clientSecret string) error`
- `executeInteractiveMode(ctx context.Context) error`

### 4. File Rename
```bash
# Old
internal/mode/orchestrator.go

# New
internal/mode/controller.go
```

### 5. main.go Usage Updated
```go
// Before
orchestratorCfg := mode.Config{...}
orchestrator := mode.NewOnboardingOrchestrator(orchestratorCfg)
return orchestrator.Execute(ctx)

// After
controllerCfg := mode.Config{...}
controller := mode.NewOnboardingController(controllerCfg)
return controller.Execute(ctx)
```

## Files Modified

1. **`internal/mode/orchestrator.go`** → **`internal/mode/controller.go`**
   - Renamed type `OnboardingOrchestrator` → `OnboardingController`
   - Renamed function `NewOnboardingOrchestrator` → `NewOnboardingController`
   - Updated all method receivers

2. **`cmd/device-discovery/main.go`**
   - Changed `orchestratorCfg` → `controllerCfg`
   - Changed `orchestrator` → `controller`
   - Updated function call to `mode.NewOnboardingController()`

## Verification

Build successful after rename:
```bash
cd /home/hspe/ppanigra/infra-onboarding/device-discovery
go build -o /tmp/test-build ./cmd/device-discovery
# ✓ Success
```

All errors cleared:
- ✓ controller.go: No errors
- ✓ main.go: No errors

## Documentation Impact

The following documentation files reference "orchestrator" and may need updates for consistency (not required for functionality):

- `MODES_ANALYSIS.md` - Architecture design document
- `REFACTORING_COMPLETE.md` - Refactoring documentation
- `QUICK_START.md` - Developer guide
- `SCRIPT_RELOCATION.md` - Script migration document
- `REFACTORING_VISUAL.md` - Visual diagrams
- `MODES_SUMMARY.md` - Summary document
- `INDEX.md` - Project index

These are **documentation only** and can be updated separately. The code is fully functional with the new naming.

## Alternative Names Considered

1. **DeviceOnboardingCoordinator** - Too verbose
2. **OnboardingModeRouter** - "Router" suggests network routing
3. **OnboardingController** - ✅ **Selected** (standard pattern, clear, no conflicts)
4. **OnboardingFlowManager** - Conflicts with "Onboarding Manager" service name
5. **DeviceOnboardingDirector** - Less common term

## Migration Guide

If you have code that references the old name:

```go
// Old code
import "device-discovery/internal/mode"

orchestrator := mode.NewOnboardingOrchestrator(cfg)
err := orchestrator.Execute(ctx)

// New code
import "device-discovery/internal/mode"

controller := mode.NewOnboardingController(cfg)
err := controller.Execute(ctx)
```

The API remains unchanged - only the type and function names have changed.

## Date
January 5, 2026
