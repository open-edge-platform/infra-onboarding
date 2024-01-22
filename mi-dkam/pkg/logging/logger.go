// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package logging

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/trace"
)

//nolint:gochecknoinits // Using init for defining flags is a valid exception.
func init() {
	flag.Func(
		"globalLogLevel",
		"Sets the application-wide logging level. Must be a valid zerolog.Level. Defaults to 'info'",
		handleLogLevel,
	)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
}

func handleLogLevel(l string) error {
	level, err := zerolog.ParseLevel(l)
	if err != nil {
		return err
	}
	zerolog.SetGlobalLevel(level)
	return nil
}

type MILogger struct {
	zerolog.Logger
}

type MICtxLogger struct {
	zerolog.Logger
}

type spanlogHook struct {
	span trace.Span
}

func (h spanlogHook) Run(_ *zerolog.Event, _ zerolog.Level, msg string) {
	if h.span.IsRecording() {
		h.span.AddEvent(msg)
	}
}

// Deprecated: use zerolog.SetGlobalLevel directly.
func SetLevel(debug bool) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
}

// Deprecated: use zerolog.SetGlobalLevel directly.
func DisableLog() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}

func GetLogger(component string) MILogger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.TimestampFieldName = "timestamp"
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// use UTC time
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}

	var logger zerolog.Logger
	if _, present := os.LookupEnv("HUMAN"); present {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339Nano})
	} else {
		logger = zerolog.New(os.Stdout)
	}

	logger = logger.With().Caller().Timestamp().Str("component", component).Logger()

	return MILogger{logger}
}

func (l MILogger) TraceCtx(ctx context.Context) MICtxLogger {
	span := trace.SpanFromContext(ctx)
	newlogger := l.With().
		Str("span_id", span.SpanContext().SpanID().String()).
		Str("trace_id", span.SpanContext().TraceID().String()).
		Logger()
	newlogger = newlogger.Hook(spanlogHook{span})
	return MICtxLogger{newlogger}
}

// MiSec is a logging decorator for MILogger intended to be used for security related events.
// Check LPIO-98 for an extensive list.
func (l *MILogger) MiSec() *MILogger {
	return &MILogger{l.With().Str("MISec", "true").Logger()}
}

// MiSec is a logging decorator MICtxLogger intended to be used for security related events.
// Check LPIO-98 for an extensive list.
func (l *MICtxLogger) MiSec() *MICtxLogger {
	return &MICtxLogger{l.With().Str("MISec", "true").Logger()}
}

// MiErr is an extension for MILogger intended to be used for error logging.
func (l *MILogger) MiErr(err error) *zerolog.Event {
	miLogger := &MILogger{l.With().Err(err).Logger()}
	return miLogger.Error()
}

// MiErr is an extension for MICtxLogger intended to be used for error logging.
func (l *MICtxLogger) MiErr(err error) *zerolog.Event {
	miLogger := &MICtxLogger{l.With().Err(err).Logger()}
	return miLogger.Error()
}

// MiError is an extension for MILogger intended to be used for logging of inline errors.
func (l *MILogger) MiError(format string, args ...interface{}) *zerolog.Event {
	logger := &MILogger{l.With().Str("error", fmt.Sprintf(format, args...)).Logger()}
	return logger.Error()
}

// MiError is an extension for MICtxLogger intended to be used for logging of inline errors.
func (l *MICtxLogger) MiError(format string, args ...interface{}) *zerolog.Event {
	logger := &MICtxLogger{l.With().Str("error", fmt.Sprintf(format, args...)).Logger()}
	return logger.Error()
}
