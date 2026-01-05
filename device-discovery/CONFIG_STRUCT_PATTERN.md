# Config Struct Pattern - Go Best Practice

## The Problem: Parameter Explosion

### Anti-Pattern (Before)
```go
func deviceDiscovery(debug bool, timeout time.Duration, obsSVC string, 
    obmSVC string, obmPort int, keycloakURL string, macAddr string, 
    uuid string, serialNumber string, ipAddress string, caCertPath string) error {
    // 11 parameters! 😱
}
```

**Issues:**
- Hard to read and maintain
- Easy to mix up parameter order
- Difficult to add new parameters
- Refactoring requires changing all call sites
- No clear relationship between parameters
- IDE autocomplete becomes useless

## The Solution: Config Struct Pattern

### Best Practice (After)
```go
func deviceDiscovery(cfg *CLIConfig) error {
    // Single parameter! ✨
    // Access fields as: cfg.Debug, cfg.Timeout, etc.
}
```

**Benefits:**
- Clear, readable function signature
- Named fields prevent mix-ups
- Easy to add new configuration
- Backward compatible with new fields
- Self-documenting code
- IDE autocomplete works perfectly

## Examples from Mature Go Projects

### 1. Kubernetes Client-Go
```go
// k8s.io/client-go/rest
func NewForConfig(c *Config) (*Clientset, error) {
    // Single config struct
}

type Config struct {
    Host            string
    APIPath         string
    Username        string
    Password        string
    BearerToken     string
    TLSClientConfig TLSClientConfig
    // ... many more fields
}
```

### 2. gRPC
```go
// google.golang.org/grpc
func Dial(target string, opts ...DialOption) (*ClientConn, error) {
    // Uses option pattern with functional options
}

// Or the config struct approach:
func DialContext(ctx context.Context, target string, opts ...DialOption) (*ClientConn, error)
```

### 3. Docker SDK
```go
// github.com/docker/docker/client
func NewClientWithOpts(ops ...Opt) (*Client, error) {
    // Functional options pattern
}

type Opt func(*Client) error

func WithHost(host string) Opt {
    return func(c *Client) error {
        c.host = host
        return nil
    }
}
```

### 4. HTTP Server (stdlib)
```go
// net/http
type Server struct {
    Addr           string
    Handler        Handler
    ReadTimeout    time.Duration
    WriteTimeout   time.Duration
    MaxHeaderBytes int
    TLSConfig      *tls.Config
    // ... many more fields
}

func (srv *Server) ListenAndServe() error
```

### 5. Prometheus Client
```go
// github.com/prometheus/client_golang/prometheus
func NewHistogram(opts HistogramOpts) Histogram {
    // Config struct pattern
}

type HistogramOpts struct {
    Namespace   string
    Subsystem   string
    Name        string
    Help        string
    Buckets     []float64
    ConstLabels Labels
}
```

### 6. Cobra CLI Framework
```go
// github.com/spf13/cobra
type Command struct {
    Use                string
    Short              string
    Long               string
    Run                func(cmd *Command, args []string)
    PersistentFlags    *flag.FlagSet
    // ... many more fields
}
```

## Our Implementation

### Before (11 parameters)
```go
deviceDiscovery(
    cfg.Debug,       // bool
    cfg.Timeout,     // time.Duration
    cfg.ObsSvc,      // string
    cfg.ObmSvc,      // string
    cfg.ObmPort,     // int
    cfg.KeycloakURL, // string
    cfg.MacAddr,     // string
    cfg.UUID,        // string
    cfg.SerialNumber,// string
    cfg.IPAddress,   // string
    cfg.CaCertPath,  // string
)
```

### After (1 parameter)
```go
deviceDiscovery(cfg)
```

## When to Use Config Struct

### Use Config Struct When:
- **3+ related parameters** - If you have more than 2-3 parameters that are related
- **Configuration data** - Parameters represent configuration
- **Growing parameter list** - Likely to add more parameters in future
- **Optional parameters** - Many parameters are optional
- **Complex types** - Parameters include complex types or sub-configs

### Use Individual Parameters When:
- **1-2 simple parameters** - Very simple functions
- **Unrelated parameters** - Parameters have no logical grouping
- **Standard library patterns** - Matching stdlib patterns (e.g., `io.Copy(dst, src)`)
- **Context parameter** - Context should always be first, separate parameter

## Best Practices

### 1. Context Separate from Config
```go
// ✅ Good - Context is first, separate from config
func deviceDiscovery(ctx context.Context, cfg *CLIConfig) error

// ❌ Bad - Context buried in config
type Config struct {
    Ctx context.Context  // Don't do this
}
```

### 2. Pointer vs Value
```go
// ✅ Good - Pointer for large structs, allows nil
func process(cfg *Config) error

// ✅ Also good - Value for small, immutable configs
func validate(opts ValidationOpts) error

// Rule: If struct is > 3-4 fields or might grow, use pointer
```

### 3. Validation in Constructor
```go
// ✅ Good - Validate and return error
func NewClient(cfg *Config) (*Client, error) {
    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }
    return &Client{config: cfg}, nil
}

// Add validation method
func (c *Config) Validate() error {
    if c.Host == "" {
        return errors.New("host is required")
    }
    return nil
}
```

### 4. Embedded Configs for Organization
```go
type CLIConfig struct {
    // Service configuration
    Services ServiceConfig
    
    // Device information
    Device DeviceInfo
    
    // Optional settings
    Options RuntimeOptions
}

type ServiceConfig struct {
    ObmSvc      string
    ObsSvc      string
    ObmPort     int
    KeycloakURL string
}

type DeviceInfo struct {
    MacAddr      string
    SerialNumber string
    UUID         string
    IPAddress    string
}
```

### 5. Functional Options Pattern (Alternative)
```go
// For more flexibility, use functional options
type Option func(*Config)

func WithDebug(debug bool) Option {
    return func(c *Config) {
        c.Debug = debug
    }
}

func WithTimeout(timeout time.Duration) Option {
    return func(c *Config) {
        c.Timeout = timeout
    }
}

// Usage
client, err := NewClient(
    WithDebug(true),
    WithTimeout(5*time.Minute),
)
```

## Comparison Table

| Aspect | Many Parameters | Config Struct | Functional Options |
|--------|----------------|---------------|-------------------|
| Readability | ❌ Poor | ✅ Good | ✅ Excellent |
| Maintainability | ❌ Difficult | ✅ Easy | ✅ Easy |
| Adding fields | ❌ Breaking | ✅ Non-breaking | ✅ Non-breaking |
| Optional params | ❌ Awkward | ✅ Natural | ✅ Perfect |
| Validation | ⚠️ Per param | ✅ Centralized | ✅ Per option |
| Complexity | ✅ Simple | ✅ Simple | ⚠️ More complex |

## Migration Strategy

### Step 1: Create Config Struct (if not exists)
```go
type CLIConfig struct {
    // All your parameters become fields
}
```

### Step 2: Update Function Signature
```go
// Before
func myFunc(param1 string, param2 int, param3 bool) error

// After
func myFunc(cfg *Config) error
```

### Step 3: Update Call Sites
```go
// Before
err := myFunc("value", 42, true)

// After
err := myFunc(&Config{
    Param1: "value",
    Param2: 42,
    Param3: true,
})

// Or if you already have config:
err := myFunc(cfg)
```

## Real-World Impact

### Code Diff Example

**Before (Hard to read):**
```go
if err := deviceDiscovery(cfg.Debug, cfg.Timeout, cfg.ObsSvc, cfg.ObmSvc, 
    cfg.ObmPort, cfg.KeycloakURL, cfg.MacAddr, cfg.UUID, cfg.SerialNumber, 
    cfg.IPAddress, cfg.CaCertPath); err != nil {
    return err
}
```

**After (Crystal clear):**
```go
if err := deviceDiscovery(cfg); err != nil {
    return err
}
```

### Adding New Configuration

**Before (Breaking change):**
```go
// Have to update every call site
func deviceDiscovery(...old params..., newParam string) error
```

**After (Non-breaking change):**
```go
type CLIConfig struct {
    // ... existing fields ...
    NewParam string  // Just add new field
}
// No changes needed to function signature or call sites!
```

## Conclusion

The config struct pattern is:
- ✅ **Standard practice** in mature Go projects
- ✅ **More maintainable** - easier to extend
- ✅ **More readable** - self-documenting
- ✅ **Type-safe** - can't mix up parameter order
- ✅ **Testable** - easy to create test configs
- ✅ **Future-proof** - easy to add new fields

When you see more than 2-3 related parameters, reach for a config struct. Your future self (and your team) will thank you! 🎉
