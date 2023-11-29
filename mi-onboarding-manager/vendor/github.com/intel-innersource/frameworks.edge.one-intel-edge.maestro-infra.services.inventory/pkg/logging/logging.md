# Logging Package for Maestro-I

This package implements a common logging package for IaaS applications, based
on [zerolog](https://github.com/rs/zerolog).

## Controlling the Log Level

This logging package defines a CLI `flag` named `-globalLogLevel`. As the name
suggests, it sets the global log level exposed by zerolog. Apps that want to
expose this flag, must call `flag.Parse()` in their `main` function. This is the
preferred approach to ensure a consistent UX. Should an app need to deviate, it
can call `zerolog.SetGlobalLevel(...)` as required.

## Output Formatting

By default, logging output is in machine-readable JSON format. For use cases
where a more human-readable format is desired, the `HUMAN` environment variable
should be set.

## Security Logging

Logging package exposes a tag called `MiSec` which can be used to identify
security events happening across MI components

```go
// zlog is MILogger, printing a security event
zlog.MiSec().Info().Msgf("Client %s authorized", client.UUID)

// zlog is MICtxLogger, printing a security event
zlog := zlog.TraceCtx(ctx)
zlog.MiSec().Info().Msgf("Client %s authorized", client.UUID)
```

## Error Logging

Logging package exposes utilities to append `error` into the logs which can be easily
scraped by external tools

```go
// zlog is MILogger, printing a security event and error
err := errors.Errorfc(codes.PermissionDenied, "Permission denied for client: %s", "1")
zlog.MiSec().MiErr(err).Msg("")

// zlog is MICtxLogger, printing a security event and error
zlog := zlog.TraceCtx(ctx)
zlog.MiSec().MiErr(err).Msg("")

// zlog is MILogger, printing a security event and error string
zlog.MiSec().MiError("Permission denied for client: %s", "1").Msg("CreateResource")

// zlog is MICtxLogger, printing a security event and error string
zlog := zlog.TraceCtx(ctx)
zlog.MiSec().MiError("Permission denied for client: %s", "1").Msg("CreateResource")
```
