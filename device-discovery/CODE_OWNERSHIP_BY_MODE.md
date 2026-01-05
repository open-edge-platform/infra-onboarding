# Code Ownership by Mode - Detailed Breakdown

This document maps every function and code block to its respective mode for easy identification and future refactoring.

## Color Legend
- 🔵 **Non-Interactive Mode** - Automatic streaming onboarding
- 🟢 **Interactive Mode** - Manual fallback with TTY interaction
- 🟡 **Shared** - Used by both modes
- 🟠 **Orchestration** - Mode selection and coordination

---

## File: `cmd/device-discovery/main.go`

### Functions by Mode

| Line Range | Function | Mode | Description |
|------------|----------|------|-------------|
| 48-102 | `main()` | 🟡 Shared | Entry point, calls both modes via orchestration |
| 104-144 | `parseCLIFlags()` | 🟡 Shared | CLI argument parsing (used by both) |
| 146-181 | Custom help | 🟡 Shared | Help text generation |
| 183-221 | `validateConfig()` | 🟡 Shared | Configuration validation |
| 223-254 | `autoDetectSystemInfo()` | 🟡 Shared | Hardware info detection |
| 256-271 | `deviceDiscovery()` | 🟡 Shared | Timeout setup, calls grpcClient |
| **273-371** | **`grpcClient()`** | **🟠 Orchestration** | **MODE SWITCHING LOGIC** |
| 290-299 | Non-interactive block | 🔵 Non-Interactive | GrpcStreamClient() call |
| 301-325 | Interactive block | 🟢 Interactive | Fallback handling |
| 327-369 | Non-interactive auth | 🔵 Non-Interactive | ClientAuth(), token saving |
| **373-408** | **`runClientAuthScript()`** | **🟢 Interactive** | **Script execution** |

### Detailed Code Ownership

#### 🟠 MODE ORCHESTRATION (Lines 273-371)
```go
func grpcClient(ctx context.Context, cfg *CLIConfig) error {
    // Line 290-299: 🔵 NON-INTERACTIVE MODE
    clientID, clientSecret, err, fallback := client.GrpcStreamClient(
        ctx, cfg.ObsSvc, cfg.ObmPort, cfg.MacAddr, 
        cfg.UUID, cfg.SerialNumber, cfg.IPAddress, cfg.CaCertPath,
    )
    
    // Line 301: 🟠 ORCHESTRATION - Mode decision point
    if fallback {
        // Line 302-325: 🟢 INTERACTIVE MODE
        fmt.Printf("Executing fallback method because of error: %s\n", err)
        
        // Interactive client Auth starts here
        if err := runClientAuthScript(ctx, ioOnboardingScript); err != nil {
            return fmt.Errorf("failed to run client auth script: %w", err)
        }
        
        // Retry logic for interactive onboarding
        if err := client.RetryInfraOnboardNode(
            ctx, cfg.ObmSvc, cfg.ObmPort, cfg.MacAddr, cfg.IPAddress,
            cfg.UUID, cfg.SerialNumber, cfg.CaCertPath, config.AccessTokenFile,
        ); err != nil {
            return fmt.Errorf("max retries reached: %w", err)
        }
        
        fmt.Println("Device discovery completed (interactive mode)")
        return nil
    }
    
    // Line 327-369: 🔵 NON-INTERACTIVE MODE CONTINUATION
    if err != nil {
        return fmt.Errorf("gRPC stream client error: %w", err)
    }
    
    // Save client credentials
    config.SaveToFile(config.ClientIDPath, clientID)
    config.SaveToFile(config.ClientSecretPath, clientSecret)
    
    // Client authentication
    idpAccessToken, releaseToken, err := auth.ClientAuth(
        clientID, clientSecret, cfg.KeycloakURL, ...
    )
    
    // Write tokens
    config.SaveToFile(config.AccessTokenFile, idpAccessToken)
    config.SaveToFile(config.ReleaseTokenFile, releaseToken)
    
    return nil
}
```

#### 🟢 INTERACTIVE MODE ONLY (Lines 373-408)
```go
func runClientAuthScript(ctx context.Context, scriptContent []byte) error {
    // Entire function is interactive mode
    // Executes client-auth.sh for TTY credential collection
}
```

---

## File: `internal/client/client.go`

### Functions by Mode

| Line Range | Function | Mode | Description |
|------------|----------|------|-------------|
| 28-57 | `createSecureConnection()` | 🟡 Shared | TLS connection setup (both modes) |
| **59-159** | **`GrpcStreamClient()`** | **🔵 Non-Interactive** | **Streaming client** |
| **162-242** | **`GrpcInfraOnboardNodeJWT()`** | **🟢 Interactive** | **JWT-based registration** |
| **244-263** | **`RetryInfraOnboardNode()`** | **🟢 Interactive** | **Retry logic** |

### Detailed Code Ownership

#### 🟡 SHARED (Lines 28-57)
```go
func createSecureConnection(ctx context.Context, target string, caCertPath string) (*grpc.ClientConn, error) {
    // Used by both modes for TLS setup
    // Load CA cert, create cert pool, establish connection
}
```

#### 🔵 NON-INTERACTIVE MODE (Lines 59-159)
```go
func GrpcStreamClient(ctx context.Context, address string, port int, mac string, 
                      uuid string, serial string, ipAddress string, caCertPath string) 
                      (string, string, error, bool) {
    
    // Line 67-68: Create connection
    conn, err := createSecureConnection(ctx, target, caCertPath)
    
    // Line 68: NON-INTERACTIVE CLIENT
    cli := pb.NewNonInteractiveOnboardingServiceClient(conn)
    
    // Line 71: STREAMING RPC
    stream, err := cli.OnboardNodeStream(ctx)
    
    // Line 76-82: Send request
    request := &pb.OnboardNodeStreamRequest{
        MacId: mac, Uuid: uuid, Serialnum: serial, HostIp: ipAddress,
    }
    
    // Line 85-158: Polling loop with state handling
    for {
        stream.Send(request)
        resp, err := stream.Recv()
        
        switch resp.NodeState {
            case pb.OnboardNodeStreamResponse_NODE_STATE_REGISTERED:
                // Wait with exponential backoff (2s → 32s)
                time.Sleep(backoff + jitter)
                
            case pb.OnboardNodeStreamResponse_NODE_STATE_ONBOARDED:
                // SUCCESS - return credentials
                return clientID, clientSecret, nil, fallback
                
            case pb.OnboardNodeStreamResponse_NODE_STATE_UNSPECIFIED:
                // ERROR
                return "", "", fmt.Errorf("unspecified state"), fallback
        }
        
        if resp.Status.Code == codes.NotFound {
            // TRIGGER FALLBACK TO INTERACTIVE
            fallback = true
            return "", "", fmt.Errorf(resp.Status.Message), fallback
        }
    }
}
```

#### 🟢 INTERACTIVE MODE (Lines 162-242)
```go
func GrpcInfraOnboardNodeJWT(ctx context.Context, address string, port int, 
                             mac string, ip string, uuid string, serial string,
                             caCertPath string, accessTokenPath string) error {
    
    // Line 164-173: Load CA certificate
    caCert, err := os.ReadFile(caCertPath)
    certPool := x509.NewCertPool()
    certPool.AppendCertsFromPEM(caCert)
    creds := credentials.NewClientTLSFromCert(certPool, "")
    
    // Line 176-179: Read JWT token from file (from client-auth.sh)
    jwtToken, err := os.ReadFile(accessTokenPath)
    tokenString := strings.TrimSpace(string(jwtToken))
    
    // Line 181-200: Create connection with OAuth2 credentials
    conn, err := grpc.DialContext(
        ctx, target,
        grpc.WithBlock(),
        grpc.WithTransportCredentials(creds),
        grpc.WithPerRPCCredentials(
            oauth.TokenSource{
                TokenSource: oauth2.StaticTokenSource(
                    &oauth2.Token{AccessToken: tokenString},
                ),
            },
        ),
    )
    
    // Line 209: INTERACTIVE CLIENT
    cli := pb.NewInteractiveOnboardingServiceClient(conn)
    
    // Line 211-221: Prepare node data
    nodeData := &pb.NodeData{
        Hwdata: []*pb.HwData{{
            MacId: mac, SutIp: ip, Uuid: uuid, Serialnum: serial,
        }},
    }
    nodeRequest := &pb.CreateNodesRequest{Payload: []*pb.NodeData{nodeData}}
    
    // Line 224: UNARY RPC (not streaming)
    nodeResponse, err = cli.CreateNodes(ctx, nodeRequest)
    
    // Line 231-237: Save project ID
    config.SaveToFile(config.ProjectIDPath, nodeResponse.ProjectId)
    
    return nil
}
```

#### 🟢 INTERACTIVE MODE RETRY (Lines 244-263)
```go
func RetryInfraOnboardNode(ctx context.Context, obmSVC string, obmPort int,
                          macAddr string, ipAddress string, uuid string,
                          serialNumber string, caCertPath string, 
                          accessTokenFile string) error {
    
    maxRetries := 3
    retryDelay := 2 * time.Second
    
    for retries := 0; retries < maxRetries; retries++ {
        // Call interactive JWT function
        err := GrpcInfraOnboardNodeJWT(
            ctx, obmSVC, obmPort, macAddr, ipAddress, 
            uuid, serialNumber, caCertPath, accessTokenFile,
        )
        
        if err == nil {
            return nil
        }
        
        // Retry with jitter
        time.Sleep(retryDelay + jitter)
    }
    
    return fmt.Errorf("max retries reached")
}
```

---

## File: `internal/auth/auth.go`

### Functions by Mode

| Line Range | Function | Mode | Description |
|------------|----------|------|-------------|
| 40-84 | `ClientAuth()` | 🔵 Non-Interactive | Exchanges client credentials for tokens |
| 86-131 | `fetchAccessToken()` | 🔵 Non-Interactive | Gets Keycloak access token |
| 133-178 | `fetchReleaseToken()` | 🔵 Non-Interactive | Gets release token |

**Note:** All auth.go functions are **Non-Interactive Mode only**. Interactive mode uses `client-auth.sh` for authentication.

---

## File: `client-auth.sh` (Embedded Script)

### Entire File

| Component | Mode | Description |
|-----------|------|-------------|
| **Entire script** | 🟢 Interactive | TTY credential collection and Keycloak authentication |

### Key Sections

```bash
# Lines 23-44: enable_tty() - 🟢 Interactive
# Prompts user for username/password on TTY devices (ttyS0, ttyS1, tty0)

# Lines 46-50: show_incorrect_credentials() - 🟢 Interactive
# Displays error message on TTY

# Lines 52-169: main() - 🟢 Interactive
# Main authentication flow:
# 1. Loop up to 3 times
# 2. Enable TTY input
# 3. Read username/password
# 4. Validate credentials (min 3 chars)
# 5. POST to Keycloak token endpoint
# 6. GET release token
# 7. Save tokens to /dev/shm/
```

---

## File: `internal/config/config.go`

### Constants and Functions

| Item | Mode | Description |
|------|------|-------------|
| `TokenFolder` | 🟡 Shared | `/dev/shm` - used by both |
| `CaCertPath` | 🟡 Shared | CA cert location - used by both |
| `ClientIDPath` | 🔵 Non-Interactive | Client ID file path |
| `ClientSecretPath` | 🔵 Non-Interactive | Client secret file path |
| `AccessTokenFile` | 🟡 Shared | Access token - used by both |
| `ReleaseTokenFile` | 🟡 Shared | Release token - used by both |
| `ProjectIDPath` | 🟡 Shared | Project ID - used by both |
| `KeycloakTokenURL` | 🔵 Non-Interactive | Token endpoint for client auth |
| `ReleaseTokenURL` | 🔵 Non-Interactive | Release token endpoint |
| `SaveToFile()` | 🟡 Shared | File saving - used by both |
| `UpdateHosts()` | 🟡 Shared | Hosts file update - used by both |
| `CreateTempScript()` | 🟢 Interactive | Temp script creation |

---

## Summary Table: Functions by Mode

### 🔵 Non-Interactive Mode Functions (7 functions)

1. `internal/client/client.go` → `GrpcStreamClient()`
2. `internal/auth/auth.go` → `ClientAuth()`
3. `internal/auth/auth.go` → `fetchAccessToken()`
4. `internal/auth/auth.go` → `fetchReleaseToken()`
5. `cmd/device-discovery/main.go` → `grpcClient()` (lines 327-369 only)

### 🟢 Interactive Mode Functions (5 functions + 1 script)

1. `internal/client/client.go` → `GrpcInfraOnboardNodeJWT()`
2. `internal/client/client.go` → `RetryInfraOnboardNode()`
3. `cmd/device-discovery/main.go` → `runClientAuthScript()`
4. `cmd/device-discovery/main.go` → `grpcClient()` (lines 301-325 only)
5. `client-auth.sh` → Entire script

### 🟡 Shared Functions (8 functions)

1. `internal/client/client.go` → `createSecureConnection()`
2. `internal/config/config.go` → `SaveToFile()`
3. `internal/config/config.go` → `UpdateHosts()`
4. `internal/sysinfo/sysinfo.go` → All functions (4 total)
5. `cmd/device-discovery/main.go` → `main()`, `parseCLIFlags()`, `validateConfig()`, `autoDetectSystemInfo()`

### 🟠 Orchestration Functions (1 function)

1. `cmd/device-discovery/main.go` → `grpcClient()` (mode switching logic)

---

## Refactoring Checklist

When segregating modes, move these functions:

### To `internal/mode/noninteractive/`
- [ ] `GrpcStreamClient()` → `client.go`
- [ ] Non-interactive auth flow from `grpcClient()` → `handler.go`

### To `internal/mode/interactive/`
- [ ] `GrpcInfraOnboardNodeJWT()` → `client.go`
- [ ] `RetryInfraOnboardNode()` → `retry.go`
- [ ] `runClientAuthScript()` → `script.go`
- [ ] Interactive fallback from `grpcClient()` → `handler.go`

### To `internal/connection/`
- [ ] `createSecureConnection()` → `grpc.go`

### To `internal/mode/`
- [ ] Mode orchestration logic from `grpcClient()` → `orchestrator.go`

---

## Files Created During Execution

### Non-Interactive Mode Creates:
```
/dev/shm/io-client-id          ← GrpcStreamClient()
/dev/shm/io-client-secret      ← GrpcStreamClient()
/dev/shm/project_id            ← GrpcStreamClient()
/dev/shm/idp_access_token      ← ClientAuth()
/dev/shm/release_token         ← ClientAuth()
```

### Interactive Mode Creates:
```
/dev/shm/idp_access_token      ← client-auth.sh
/dev/shm/release_token         ← client-auth.sh
/dev/shm/project_id            ← GrpcInfraOnboardNodeJWT()
/idp_username                  ← client-auth.sh (temporary)
/idp_password                  ← client-auth.sh (temporary)
/var/log/client-auth/          ← client-auth.sh
  └── client-auth.log
```

### Shared Files Read:
```
/etc/pki/ca-trust/source/anchors/server_cert.pem  ← Both modes
/etc/hook/env_config                              ← client-auth.sh
```

---

## Conclusion

This breakdown shows that while the modes are conceptually distinct, they are currently **mixed within the same files**, particularly in:
- `internal/client/client.go` (3 functions, 2 different modes)
- `cmd/device-discovery/main.go` (1 function with branching logic)

**Recommendation:** Follow the proposed package structure to clearly separate these concerns.
