# Error Handling Refactoring

## Overview

Refactored the device-discovery application to follow standard Go error handling patterns, ensuring proper error propagation and handling throughout the codebase.

## Changes Made

### Before (Anti-Pattern)

```go
func deviceDiscovery(...) {
    if debug {
        ctx, cancel := context.WithTimeout(...)
        defer cancel()
        grpcClient(ctx, ...)  // No error handling
    } else {
        grpcClient(context.Background(), ...)  // No error handling
    }
}

func grpcClient(...) {
    // Multiple log.Fatalf() calls scattered throughout
    if err != nil {
        log.Fatalf("error: %v", err)  // Exits immediately
    }
}
```

**Issues:**
- Functions don't return errors
- Multiple `log.Fatalf()` calls exit without cleanup
- Error context is lost
- Difficult to test
- No way to handle errors gracefully

### After (Standard Go Pattern)

```go
func deviceDiscovery(...) error {
    var ctx context.Context
    var cancel context.CancelFunc
    
    if debug {
        ctx, cancel = context.WithTimeout(...)
        defer cancel()
    } else {
        ctx = context.Background()
    }
    
    return grpcClient(ctx, ...)  // Propagate errors
}

func grpcClient(...) error {
    // Return descriptive errors with context
    if err != nil {
        return fmt.Errorf("failed to save client ID: %w", err)
    }
    return nil
}
```

**Benefits:**
- Errors propagate up the call stack
- Descriptive error messages with context
- Proper error wrapping using `%w`
- Testable functions
- Clean resource cleanup
- Single point of exit in `main()`

## Detailed Changes

### 1. Function Signatures Updated

#### `deviceDiscovery()`
```go
// Before
func deviceDiscovery(...) { }

// After  
func deviceDiscovery(...) error { }
```

#### `grpcClient()`
```go
// Before
func grpcClient(...) { }

// After
func grpcClient(...) error { }
```

### 2. Error Handling in main()

```go
// Single error handling point
if err := deviceDiscovery(...); err != nil {
    log.Fatalf("Device discovery failed: %v", err)
}
fmt.Println("Device discovery completed successfully")
```

### 3. Error Wrapping with Context

All errors now include context using `fmt.Errorf()` with `%w`:

```go
// Before
log.Fatalf("error writing clientID: %v", err)

// After
return fmt.Errorf("failed to save client ID: %w", err)
```

**Error Context Examples:**
- `"failed to save client ID: %w"`
- `"failed to save client secret: %w"`
- `"authentication failed: %w"`
- `"failed to save access token: %w"`
- `"gRPC stream client error: %w"`
- `"max retries reached, could not complete device discovery: %w"`
- `"failed to run client auth script: %w"`

### 4. Context Management

Improved context handling in `deviceDiscovery()`:

```go
var ctx context.Context
var cancel context.CancelFunc

if debug {
    ctx, cancel = context.WithTimeout(context.Background(), timeout)
    defer cancel()
} else {
    ctx = context.Background()
}

return grpcClient(ctx, ...)
```

### 5. Script Execution Error Handling

Enhanced `runClientAuthScript()`:

```go
defer func() {
    tmpfile.Close()
    os.Remove(tmpfile.Name())  // Ensure cleanup
}()

// ...

case <-ctx.Done():
    syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
    return fmt.Errorf("client-auth.sh timed out: %w", ctx.Err())
```

### 6. Fallback Path Error Handling

```go
if fallback {
    fmt.Printf("Executing fallback method because of error: %s\n", err)
    
    if err := runClientAuthScript(ctx, ioOnboardingScript); err != nil {
        return fmt.Errorf("failed to run client auth script: %w", err)
    }
    
    if err := client.RetryInfraOnboardNode(...); err != nil {
        return fmt.Errorf("max retries reached, could not complete device discovery: %w", err)
    }
    
    fmt.Println("Device discovery completed (interactive mode)")
    return nil
}
```

### 7. Success Path Error Handling

```go
// Handle non-fallback case
if err != nil {
    return fmt.Errorf("gRPC stream client error: %w", err)
}

// Save credentials with error propagation
if err := config.SaveToFile(config.ClientIDPath, clientID); err != nil {
    return fmt.Errorf("failed to save client ID: %w", err)
}

// Authenticate with error propagation
idpAccessToken, releaseToken, err := auth.ClientAuth(...)
if err != nil {
    return fmt.Errorf("authentication failed: %w", err)
}

// Save tokens with error propagation
if err := config.SaveToFile(config.AccessTokenFile, idpAccessToken); err != nil {
    return fmt.Errorf("failed to save access token: %w", err)
}

return nil  // Success
```

## Benefits of This Approach

### 1. **Better Error Messages**
Errors now include full context:
```
Device discovery failed: authentication failed: failed to get JWT access token from Keycloak: Post "https://...": connection refused
```

### 2. **Easier Debugging**
Error chain shows exactly where failure occurred:
- Which function failed
- Why it failed
- Original error from system/library

### 3. **Testability**
Functions can now be tested with error scenarios:
```go
func TestGrpcClientError(t *testing.T) {
    err := grpcClient(ctx, ...)
    if err == nil {
        t.Error("expected error")
    }
}
```

### 4. **Resource Cleanup**
Deferred cleanup works properly:
```go
defer func() {
    tmpfile.Close()
    os.Remove(tmpfile.Name())
}()
```

### 5. **Graceful Degradation**
Allows for future enhancement:
```go
// Could implement retry logic
for retries := 0; retries < maxRetries; retries++ {
    if err := deviceDiscovery(...); err != nil {
        log.Printf("Attempt %d failed: %v", retries+1, err)
        continue
    }
    break
}
```

### 6. **Standard Go Idioms**
Follows Go best practices:
- Errors are values
- Check errors immediately
- Wrap errors with context
- Single point of exit

## Error Flow

```
main()
  ├─> deviceDiscovery()
  │     └─> grpcClient()
  │           ├─> client.GrpcStreamClient()  [external]
  │           ├─> runClientAuthScript()
  │           │     └─> config.CreateTempScript()  [external]
  │           ├─> client.RetryInfraOnboardNode()  [external]
  │           ├─> config.SaveToFile()  [external]
  │           └─> auth.ClientAuth()  [external]
  │
  └─> log.Fatalf() [only on error]
```

## Comparison

### Anti-Pattern (Before)
```go
func process() {
    result, err := operation()
    if err != nil {
        log.Fatalf("operation failed: %v", err)  // EXIT
    }
    // This code may never execute
    cleanup()
}
```

### Standard Pattern (After)
```go
func process() error {
    result, err := operation()
    if err != nil {
        return fmt.Errorf("operation failed: %w", err)  // RETURN
    }
    return nil
}

func main() {
    if err := process(); err != nil {
        cleanup()
        log.Fatalf("Process failed: %v", err)  // EXIT
    }
}
```

## Testing Impact

Functions can now be tested without process exits:

```go
func TestDeviceDiscovery(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
    }{
        {
            name: "success",
            config: validConfig,
            wantErr: false,
        },
        {
            name: "auth failure",
            config: invalidAuthConfig,
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := deviceDiscovery(tt.config)
            if (err != nil) != tt.wantErr {
                t.Errorf("deviceDiscovery() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Conclusion

The refactoring transforms the codebase from using scattered `log.Fatalf()` calls (which are essentially `panic` in disguise) to proper error propagation following Go best practices. This makes the code more maintainable, testable, and debuggable while providing better error messages to users.
