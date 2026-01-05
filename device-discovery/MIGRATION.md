# Migration Guide: Device Discovery Restructuring

This document describes the changes made to restructure the device-discovery project into standard Go project format.

## Summary of Changes

The device-discovery project has been reorganized from a flat structure with all Go files in the root directory to a standard Go project layout with proper package organization.

## Old Structure vs New Structure

### Before (Flat Structure)
```
device-discovery/
├── device-discovery.go
├── onboarding-client.go
├── client-secret-auth.go
├── parse-kernel-arguments.go
├── system_info_by_mac.go
├── device-discovery_test.go
├── build.sh
├── client-auth.sh
├── Dockerfile
├── go.mod
└── go.sum
```

### After (Standard Go Layout)
```
device-discovery/
├── cmd/
│   └── device-discovery/
│       ├── main.go                    # Main entry point
│       └── main_test.go               # Integration tests
├── internal/
│   ├── auth/
│   │   └── auth.go                    # Authentication logic
│   ├── client/
│   │   └── client.go                  # gRPC client
│   ├── config/
│   │   └── config.go                  # Configuration & utilities
│   ├── parser/
│   │   └── parser.go                  # Kernel argument parsing
│   └── sysinfo/
│       └── sysinfo.go                 # System information
├── build.sh
├── client-auth.sh
├── Dockerfile
├── go.mod
├── go.sum
├── Makefile                            # NEW: Build automation
└── README.md                           # NEW: Documentation
```

## File Mappings

| Old File | New Location | Package | Notes |
|----------|--------------|---------|-------|
| `device-discovery.go` | `cmd/device-discovery/main.go` | `main` | Main application logic |
| `onboarding-client.go` | `internal/client/client.go` | `client` | gRPC streaming client |
| `client-secret-auth.go` | `internal/auth/auth.go` | `auth` | Authentication functions |
| `parse-kernel-arguments.go` | `internal/parser/parser.go` | `parser` | Kernel arg parsing |
| `system_info_by_mac.go` | `internal/sysinfo/sysinfo.go` | `sysinfo` | System info retrieval |
| `device-discovery_test.go` | `cmd/device-discovery/main_test.go` | `main` | Updated with new imports |

## Key Changes

### 1. Package Organization
- **Main package** moved to `cmd/device-discovery/`
- **Internal packages** created under `internal/` for better code organization
- Each package has a clear, single responsibility

### 2. Function Exports
Functions have been renamed to follow Go conventions:
- Private functions remain lowercase (e.g., `loadCACertPool`)
- Public functions are PascalCase (e.g., `ClientAuth`, `GetSerialNumber`)

### 3. Constants
Constants moved from main package to `internal/config/` for better organization and reusability.

### 4. Build Process
The build command has been updated:

**Old:**
```bash
go build -v -o app
```

**New:**
```bash
go build -v -o app ./cmd/device-discovery
```

### 5. Import Paths
All internal imports now use the module path:
```go
import (
    "device-discovery/internal/auth"
    "device-discovery/internal/client"
    "device-discovery/internal/config"
    "device-discovery/internal/parser"
    "device-discovery/internal/sysinfo"
)
```

## Updated Build Scripts

### build.sh
Updated to build from the new `cmd/device-discovery` path:
```bash
go build -v -o app ./cmd/device-discovery
```

### Dockerfile
No changes required - still copies the `app` binary built by `build.sh`.

### NEW: Makefile
A comprehensive Makefile has been added with targets:
- `make build` - Build the application
- `make test` - Run tests
- `make clean` - Clean build artifacts
- `make fmt` - Format code
- `make vet` - Run go vet
- `make tidy` - Tidy go modules
- `make docker-build` - Build Docker image
- `make help` - Show all available targets

## Benefits of New Structure

1. **Standard Layout**: Follows the [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
2. **Better Organization**: Code is organized by functionality, not by file
3. **Improved Maintainability**: Easier to locate and modify specific functionality
4. **Testability**: Each package can be tested independently
5. **Reusability**: Internal packages are clearly separated and can be reused
6. **Scalability**: Easy to add new packages and features
7. **Documentation**: Clear package structure makes the codebase self-documenting

## Breaking Changes

### For Developers
- Import paths have changed - any code importing these packages must be updated
- Function names have been capitalized for export

### For Build Systems
- Build commands must specify `./cmd/device-discovery` as the target
- CI/CD pipelines may need to be updated if they reference old file paths

## Migration Checklist

If you're integrating this change:

- [ ] Update build scripts to use `./cmd/device-discovery`
- [ ] Update CI/CD pipelines with new build paths
- [ ] Update any documentation referencing old file paths
- [ ] Run `go mod tidy` to ensure dependencies are correct
- [ ] Test the build process: `make build`
- [ ] Test the Docker build: `make docker-build`
- [ ] Run tests: `make test`

## Backward Compatibility

The **binary output** and **runtime behavior** remain unchanged:
- ✅ Same binary name (`app`)
- ✅ Same command-line interface
- ✅ Same environment variables
- ✅ Same Docker image structure
- ✅ Same functionality

Only the **source code structure** has changed.

## Questions or Issues?

If you encounter any issues with the restructuring, please:
1. Check this migration guide
2. Review the new README.md
3. Verify your build commands match the new structure
4. Ensure `go mod tidy` has been run

## Additional Resources

- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
