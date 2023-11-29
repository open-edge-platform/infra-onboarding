/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package logger

import (
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
)

var logger *Logger

func init() {
	InitLogger(1)
}

// Logger is a Logger
type Logger struct {
	*log.Logger
}

// GetLogger returns Logger
func GetLogger() *Logger {
	return logger
}

// InitLogger initialize logger
func InitLogger(level int) {
	l := &log.Logger{
		Handler: cli.New(os.Stdout),
		Level:   log.Level(level),
	}

	// set package varriable logger
	logger = &Logger{l}
}
