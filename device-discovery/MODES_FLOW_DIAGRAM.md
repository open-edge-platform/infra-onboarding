# Device Discovery Mode Flow Diagram

## High-Level Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                      Device Discovery Start                      │
│                     (cmd/device-discovery/main.go)               │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
                ┌──────────────────┐
                │  Parse CLI Flags │
                │  & Auto-detect   │
                └────────┬─────────┘
                         │
                         ▼
                ┌──────────────────┐
                │ deviceDiscovery() │
                └────────┬─────────┘
                         │
                         ▼
                 ┌──────────────┐
                 │ grpcClient() │
                 └──────┬───────┘
                        │
                        ▼
        ┌───────────────────────────────┐
        │   Try Non-Interactive Mode    │
        │   GrpcStreamClient()          │
        │   (internal/client/client.go) │
        └───────────┬───────────────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │ Check Response:       │
        │ - fallback flag       │
        │ - error               │
        └───────┬───────────────┘
                │
                ├─────────────────────────────────────┐
                │                                     │
                ▼                                     ▼
    ┌──────────────────────┐          ┌────────────────────────┐
    │ fallback == false    │          │   fallback == true     │
    │ error == nil         │          │   (NotFound error)     │
    │                      │          │                        │
    │ NON-INTERACTIVE MODE │          │   INTERACTIVE MODE     │
    └──────────┬───────────┘          └──────────┬─────────────┘
               │                                  │
               │                                  │
    ┌──────────▼──────────┐          ┌───────────▼────────────┐
    │ Continue with       │          │ Execute client-auth.sh │
    │ clientID +          │          │ (TTY interaction)      │
    │ clientSecret        │          └───────────┬────────────┘
    └──────────┬──────────┘                      │
               │                                  ▼
               │                      ┌──────────────────────┐
               │                      │ User enters:         │
               │                      │ - Username           │
               │                      │ - Password           │
               │                      └───────────┬──────────┘
               │                                  │
               │                                  ▼
               │                      ┌──────────────────────┐
               │                      │ Keycloak Auth        │
               │                      │ Get access_token     │
               │                      └───────────┬──────────┘
               │                                  │
               │                                  ▼
               │                      ┌──────────────────────┐
               │                      │ Get release_token    │
               │                      └───────────┬──────────┘
               │                                  │
               │                                  ▼
               │                      ┌──────────────────────┐
               │                      │ RetryInfraOnboard    │
               │                      │ NodeJWT()            │
               │                      │ (max 3 attempts)     │
               │                      └───────────┬──────────┘
               │                                  │
               ▼                                  ▼
    ┌─────────────────────┐          ┌──────────────────────┐
    │ Save credentials:   │          │ Save project_id      │
    │ - client_id         │          └──────────────────────┘
    │ - client_secret     │
    │ - project_id        │
    └──────────┬──────────┘
               │
               ▼
    ┌─────────────────────┐
    │ ClientAuth()        │
    │ Exchange creds for  │
    │ tokens              │
    └──────────┬──────────┘
               │
               ▼
    ┌─────────────────────┐
    │ Save tokens:        │
    │ - access_token      │
    │ - release_token     │
    └──────────┬──────────┘
               │
               └──────────────────────┬───────────────────────┘
                                      │
                                      ▼
                          ┌───────────────────────┐
                          │  Discovery Complete   │
                          └───────────────────────┘
```

## Non-Interactive Mode Detail

```
┌─────────────────────────────────────────────────────────────────┐
│                    GrpcStreamClient()                            │
│              (internal/client/client.go:59)                      │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
              ┌─────────────────────┐
              │ Create TLS          │
              │ Connection          │
              └──────────┬──────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │ NewNonInteractive   │
              │ OnboardingService   │
              │ Client              │
              └──────────┬──────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │ Open Stream         │
              │ OnboardNodeStream() │
              └──────────┬──────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │ Send Request:       │
              │ - MacId             │
              │ - Uuid              │
              │ - Serialnum         │
              │ - HostIp            │
              └──────────┬──────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │ Receive Response    │
              └──────────┬──────────┘
                         │
         ┌───────────────┴─────────────────┐
         │                                 │
         ▼                                 ▼
┌────────────────────┐        ┌──────────────────────┐
│ Status: NotFound   │        │ Status: OK           │
│ (codes.NotFound)   │        │                      │
└────────┬───────────┘        └──────────┬───────────┘
         │                               │
         ▼                               ▼
┌────────────────────┐        ┌──────────────────────┐
│ Set fallback=true  │        │ Check NodeState      │
│ Return error       │        └──────────┬───────────┘
└────────────────────┘                   │
                              ┌──────────┴────────────┐
                              │                       │
                              ▼                       ▼
                   ┌────────────────────┐  ┌─────────────────────┐
                   │ NODE_STATE_        │  │ NODE_STATE_         │
                   │ REGISTERED         │  │ ONBOARDED           │
                   └─────────┬──────────┘  └──────────┬──────────┘
                             │                        │
                             ▼                        ▼
                   ┌────────────────────┐  ┌─────────────────────┐
                   │ Wait with backoff  │  │ Extract:            │
                   │ (2s → 32s)         │  │ - clientID          │
                   │ Loop back          │  │ - clientSecret      │
                   └────────────────────┘  │ - projectID         │
                                           │ Return success      │
                                           └─────────────────────┘
```

## Interactive Mode Detail

```
┌─────────────────────────────────────────────────────────────────┐
│                     Interactive Mode Flow                        │
│                  (Triggered by fallback==true)                   │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
              ┌─────────────────────┐
              │ runClientAuthScript()│
              │ (main.go:373)        │
              └──────────┬──────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │ Create temp file    │
              │ from embedded       │
              │ client-auth.sh      │
              └──────────┬──────────┘
                         │
                         ▼
              ┌─────────────────────┐
              │ Execute script      │
              │ /bin/sh script.tmp  │
              └──────────┬──────────┘
                         │
                         │
        ┌────────────────┴────────────────┐
        │                                 │
        ▼                                 ▼
┌───────────────┐            ┌────────────────────┐
│ Script Loop   │            │ Enable TTY         │
│ (max 3 times) │            │ - ttyS0            │
└───────┬───────┘            │ - ttyS1            │
        │                    │ - tty0             │
        │                    └────────┬───────────┘
        │                             │
        │                             ▼
        │                    ┌────────────────────┐
        │                    │ Prompt user:       │
        │                    │ read -p "Username:"│
        │                    │ read -s "Password:"│
        │                    └────────┬───────────┘
        │                             │
        │                             ▼
        │                    ┌────────────────────┐
        │                    │ Validate input     │
        │                    │ (min 3 chars each) │
        │                    └────────┬───────────┘
        │                             │
        │                             ▼
        │                    ┌────────────────────┐
        │                    │ curl POST to       │
        │                    │ Keycloak           │
        │                    │ /token endpoint    │
        │                    └────────┬───────────┘
        │                             │
        │                             ▼
        │                    ┌────────────────────┐
        │              ┌─────┤ Success?           ├─────┐
        │              │     └────────────────────┘     │
        │              │                                │
        │              ▼ Yes                            ▼ No
        │     ┌────────────────────┐         ┌──────────────────┐
        │     │ Extract            │         │ Loop back        │
        │     │ access_token       │         │ (max 3 attempts) │
        │     └────────┬───────────┘         └──────────────────┘
        │              │
        │              ▼
        │     ┌────────────────────┐
        │     │ curl GET to        │
        │     │ release server     │
        │     │ /token endpoint    │
        │     └────────┬───────────┘
        │              │
        │              ▼
        │     ┌────────────────────┐
        │     │ Save tokens:       │
        │     │ - idp_access_token │
        │     │ - release_token    │
        │     └────────┬───────────┘
        │              │
        └──────────────┘
                       │
                       ▼
            ┌─────────────────────┐
            │ RetryInfraOnboard   │
            │ NodeJWT()           │
            │ (client.go:244)     │
            └──────────┬──────────┘
                       │
              ┌────────┴────────┐
              │ Retry Loop      │
              │ (max 3 times)   │
              │ 2s delay +      │
              │ random jitter   │
              └────────┬────────┘
                       │
                       ▼
            ┌─────────────────────┐
            │ GrpcInfraOnboard    │
            │ NodeJWT()           │
            │ (client.go:162)     │
            └──────────┬──────────┘
                       │
                       ▼
            ┌─────────────────────┐
            │ Create TLS Conn     │
            │ with JWT OAuth      │
            └──────────┬──────────┘
                       │
                       ▼
            ┌─────────────────────┐
            │ NewInteractive      │
            │ OnboardingService   │
            │ Client              │
            └──────────┬──────────┘
                       │
                       ▼
            ┌─────────────────────┐
            │ CreateNodes()       │
            │ RPC Call            │
            └──────────┬──────────┘
                       │
                       ▼
            ┌─────────────────────┐
            │ Receive:            │
            │ - projectID         │
            │ Save to file        │
            └─────────────────────┘
```

## File Interaction Map

```
┌────────────────────────────────────────────────────────────────┐
│                       File Dependencies                         │
└────────────────────────────────────────────────────────────────┘

Non-Interactive Mode Files Created:
┌─────────────────────────┐
│ /dev/shm/               │
│  ├── io-client-id       │  ← GrpcStreamClient()
│  ├── io-client-secret   │  ← GrpcStreamClient()
│  ├── project_id         │  ← GrpcStreamClient()
│  ├── idp_access_token   │  ← ClientAuth()
│  └── release_token      │  ← ClientAuth()
└─────────────────────────┘

Interactive Mode Files Created:
┌─────────────────────────────────┐
│ /dev/shm/                        │
│  ├── idp_access_token            │  ← client-auth.sh
│  ├── release_token               │  ← client-auth.sh
│  └── project_id                  │  ← GrpcInfraOnboardNodeJWT()
│                                  │
│ / (root)                         │
│  ├── idp_username (temp)         │  ← client-auth.sh
│  └── idp_password (temp)         │  ← client-auth.sh
│                                  │
│ /var/log/client-auth/            │
│  └── client-auth.log             │  ← client-auth.sh
└─────────────────────────────────┘

Shared Configuration Files Read:
┌────────────────────────────────────────┐
│ /etc/pki/ca-trust/source/anchors/      │
│  └── server_cert.pem                   │  ← Both modes
│                                        │
│ /etc/hook/                             │
│  └── env_config                        │  ← client-auth.sh
└────────────────────────────────────────┘
```

## Package Dependencies

```
cmd/device-discovery/main.go
    ├── import: internal/auth
    ├── import: internal/client
    ├── import: internal/config
    ├── import: internal/sysinfo
    └── embed: client-auth.sh

internal/client/client.go
    ├── import: internal/config
    ├── import: github.com/.../onboarding-manager/pkg/api/...
    ├── Uses: NonInteractiveOnboardingServiceClient
    └── Uses: InteractiveOnboardingServiceClient

internal/auth/auth.go
    ├── import: internal/config
    └── HTTP calls to Keycloak

client-auth.sh (embedded script)
    ├── Uses: /etc/hook/env_config
    ├── Uses: TTY devices
    ├── curl → Keycloak
    └── curl → Release server
```

## Current vs Proposed Structure

```
CURRENT:                                PROPOSED:
──────────────────────────────────     ─────────────────────────────────

cmd/device-discovery/                   cmd/device-discovery/
  └── main.go                             └── main.go (minimal)
      ├── Mode switching                      └── calls orchestrator
      ├── Script execution
      ├── Non-interactive flow
      └── Interactive flow

internal/client/                        internal/mode/
  └── client.go                           ├── orchestrator.go
      ├── GrpcStreamClient()              │     └── Mode selection
      ├── GrpcInfraOnboardNodeJWT()       ├── noninteractive/
      └── RetryInfraOnboardNode()         │   ├── client.go
                                          │   │     └── GrpcStreamClient()
                                          │   └── handler.go
                                          │         └── Stream handling
                                          └── interactive/
                                              ├── client.go
                                              │     └── GrpcInfraOnboardNodeJWT()
                                              ├── script.go
                                              │     └── runClientAuthScript()
                                              └── retry.go
                                                    └── RetryInfraOnboardNode()

internal/                               internal/
  ├── auth/                               ├── auth/
  ├── config/                             ├── config/
  ├── parser/                             ├── connection/  (NEW)
  └── sysinfo/                            │   └── grpc.go
                                          │         └── createSecureConnection()
                                          ├── parser/
                                          └── sysinfo/
```

## API Surface Comparison

```
Non-Interactive Mode API:
─────────────────────────────────────────────────────────────
Service:     NonInteractiveOnboardingServiceClient
RPC:         OnboardNodeStream(stream)
Request:     OnboardNodeStreamRequest {MacId, Uuid, Serialnum, HostIp}
Response:    OnboardNodeStreamResponse {NodeState, ClientId, ClientSecret, ProjectId, Status}
States:      NODE_STATE_REGISTERED, NODE_STATE_ONBOARDED, NODE_STATE_UNSPECIFIED
Protocol:    Bidirectional Streaming
Auth:        TLS only (no OAuth)

Interactive Mode API:
─────────────────────────────────────────────────────────────
Service:     InteractiveOnboardingServiceClient
RPC:         CreateNodes(unary)
Request:     CreateNodesRequest {Payload: [NodeData {Hwdata: [HwData]}]}
Response:    CreateNodesResponse {ProjectId}
Protocol:    Unary RPC
Auth:        TLS + OAuth2 Bearer Token (JWT)
```
