/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"reflect"
	"testing"
)

func TestGetConfig(t *testing.T) {
	tests := []struct {
		name string
		want *Config
	}{
		{
			name: "Test Case 1",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetConfig(t *testing.T) {
	type args struct {
		conf *Config
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test Case 1",
			args: args{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetConfig(tt.args.conf)
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name string
		want *string
	}{
		{
			name: "Test Case 1",
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEnv(); got != tt.want {
				t.Errorf("GetEnv() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Test Case 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Load()
		})
	}
}

