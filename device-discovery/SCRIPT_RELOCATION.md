# Script Relocation: client-auth.sh

## Summary
The `client-auth.sh` script has been relocated from the repository root to `internal/mode/interactive/` to improve code organization and encapsulation.

## Changes Made

### 1. Script File Location
- **Old:** `/device-discovery/client-auth.sh`
- **New:** `/device-discovery/internal/mode/interactive/client-auth.sh`

### 2. Embedding Changes

#### Before: Embedded in main.go
```go
// cmd/device-discovery/main.go
//go:embed ../../client-auth.sh
var ioOnboardingScript []byte

// Passed to orchestrator
orchestratorCfg := mode.Config{
    AuthScript: ioOnboardingScript,
    // ... other fields
}
```

#### After: Embedded in script.go
```go
// internal/mode/interactive/script.go
//go:embed client-auth.sh
var authScript []byte

// Used directly by ExecuteAuthScript
func ExecuteAuthScript(ctx context.Context) error {
    // Uses authScript variable internally
}
```

### 3. Function Signature Changes

#### interactive.ExecuteAuthScript()
- **Before:** `ExecuteAuthScript(ctx context.Context, scriptContent []byte) error`
- **After:** `ExecuteAuthScript(ctx context.Context) error`

The script content is now embedded internally, so it doesn't need to be passed as a parameter.

### 4. Orchestrator Changes

#### Removed AuthScript Field
```go
// internal/mode/orchestrator.go

// REMOVED from OnboardingOrchestrator struct:
// authScript []byte

// REMOVED from Config struct:
// AuthScript []byte

// UPDATED function call:
// Before: interactive.ExecuteAuthScript(ctx, o.authScript)
// After:  interactive.ExecuteAuthScript(ctx)
```

### 5. main.go Changes
```go
// cmd/device-discovery/main.go

// REMOVED:
// import _ "embed"
// //go:embed ../../client-auth.sh
// var ioOnboardingScript []byte

// REMOVED from orchestratorCfg:
// AuthScript: ioOnboardingScript,
```

### 6. Dockerfile Changes
```dockerfile
# device-discovery/Dockerfile

# REMOVED (script now embedded in binary):
# COPY client-auth.sh .

# ADDED comment for clarity:
# Note: client-auth.sh is now embedded in the binary via go:embed
```

## Benefits

### 1. Better Encapsulation
- The script is now located with the code that uses it
- Interactive mode package is self-contained
- No need to pass script content around

### 2. Simplified Interfaces
- `ExecuteAuthScript()` has simpler signature (no parameters except context)
- `OnboardingOrchestrator` doesn't need to store script content
- `Config` struct is simpler

### 3. Clearer Separation of Concerns
- Main.go doesn't need to know about interactive mode's internal resources
- Script management is entirely within the interactive package
- Orchestrator just coordinates, doesn't manage resources

### 4. Docker Image Cleanup
- No need to copy external script file during Docker build
- Everything is in the compiled binary
- Simpler deployment

## File Structure

```
device-discovery/
├── cmd/
│   └── device-discovery/
│       └── main.go                          # No longer embeds script
├── internal/
│   └── mode/
│       ├── orchestrator.go                  # No AuthScript field
│       ├── noninteractive/
│       │   └── client.go
│       └── interactive/
│           ├── client.go
│           ├── script.go                    # Embeds client-auth.sh
│           └── client-auth.sh               # ← Script location
└── Dockerfile                                # No longer copies script
```

## Old File Status

The original `/device-discovery/client-auth.sh` can now be safely removed as it's:
1. No longer referenced in any Go code
2. No longer copied by Dockerfile
3. Fully replaced by `internal/mode/interactive/client-auth.sh`

**Note:** Documentation files (MODES_ANALYSIS.md, etc.) still reference the script by name, but these are conceptual references, not file paths.

## Verification

Build succeeds with embedded script:
```bash
cd /home/hspe/ppanigra/infra-onboarding/device-discovery
go build -o /tmp/test-build ./cmd/device-discovery
# ✓ Build successful
```

All errors cleared:
- ✓ script.go: No errors
- ✓ main.go: No errors
- ✓ orchestrator.go: No errors

## Migration Notes

If you're updating existing code that calls `ExecuteAuthScript()`:

```go
// Old way
scriptContent := []byte("...")
err := interactive.ExecuteAuthScript(ctx, scriptContent)

// New way
err := interactive.ExecuteAuthScript(ctx)
```

The script is now embedded internally and doesn't need to be provided.
