/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"flag"
	"fmt"

	"github.com/jinzhu/configor"
)

type (
	// Config represents the settings.
	Config struct {
		Keyspace    string `default:"inventory_manager"`
		Replica     string `default:"1"`
		CreateTable bool   `default:"false"`

		Node struct {
			Database Database
		}

		Artifact struct {
			Database Database
		}

		Log struct {
			Level int `default:"1"` // info
		}
	}

	Database struct {
		Dialect   string `default:"cassandra"`
		Endpoints string `default:"intel-cassandra:9042"` // localhost:9042,localhost:29042
		Username  string `default:"admin"`
		Password  string `default:"intel@2023"`
	}
)

var (
	config *Config
	env    *string
)

// Load reads the settings from the yml file.
func Load() {
	env = flag.String("env", "develop", "To switch configurations")
	flag.Parse()
	config = &Config{}
	cnfgor := configor.New(&configor.Config{Debug: false, ENVPrefix: "INVENTORY", Environment: *env})
	if err := cnfgor.Load(config); err != nil {
		fmt.Println("failed to load config", err)
	}
	fmt.Println("config loaded")
}

// GetConfig returns the configuration data.
func GetConfig() *Config {
	return config
}

// SetConfig sets configuration data.
func SetConfig(conf *Config) {
	config = conf
}

// GetEnv returns the environment variable.
func GetEnv() *string {
	return env
}
