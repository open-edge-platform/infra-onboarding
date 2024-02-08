/*
   Copyright (C) 2023 Intel Corporation
   SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"reflect"
	"testing"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.managers.onboarding/pkg/logger"
)

type example struct {
	ID  string `json:"id"`
	Tag string `json:"tag"`
}

type TestStruct struct {
	Field1 string
	Field2 int
	Field3 bool
}

func TestGetFields(t *testing.T) {
	type args struct {
		b interface{}
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "Test with valid struct",
			args: args{
				b: TestStruct{
					Field1: "Value1",
					Field2: 42,
					Field3: true,
				},
			},
			want: []string{"Field1", "Field2", "Field3"},
		},
		{
			name: "Test with struct containing ignored field",
			args: args{
				b: example{
					ID:  "1",
					Tag: "",
				},
			},
			want: []string{"id", "tag"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFields(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetFields() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetValues(t *testing.T) {
	type args struct {
		b interface{}
	}
	tests := []struct {
		name string
		args args
		want []interface{}
	}{
		{
			name: "Test with struct",
			args: args{b: TestStruct{
				Field1: "Value1",
				Field2: 42,
				Field3: true,
			}},
			want: []interface{}{"Value1", 42, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetValues(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetMapValues(t *testing.T) {
	type args struct {
		b interface{}
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{
			name: "Test with struct",
			args: args{b: TestStruct{"value1", 42, true}},
			want: map[string]interface{}{
				"Field1": "value1",
				"Field2": 42,
				"Field3": true,
			},
		},
		{
			name: "Test with empty struct",
			args: args{b: struct{}{}},
			want: map[string]interface{}{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetMapValues(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMapValues() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetLogger(t *testing.T) {
	tests := []struct {
		name string
		want *logger.Logger
	}{
		{
			name: "Test Case 1",
			want: &logger.Logger{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := logger.GetLogger(); reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetLogger() = %v, want %v", got, tt.want)
			}
		})
	}
}
