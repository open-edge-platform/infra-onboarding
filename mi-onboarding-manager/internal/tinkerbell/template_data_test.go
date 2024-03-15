// SPDX-FileCopyrightText: (C) 2023 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package tinkerbell

import (
	"reflect"
	"testing"
)

func TestNewTemplateDataProd(t *testing.T) {
	type args struct {
		name       string
		rootPart   string
		rootPartNo string
		hostIP     string
		provIP     string
	}
	wf := Workflow{}
	want, _ := marshalWorkflow(&wf)
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				name: "TestWorkflow",
			},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTemplateDataProd(tt.args.name, tt.args.rootPart, tt.args.rootPartNo, tt.args.hostIP, tt.args.provIP)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTemplateDataProd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTemplateDataProd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTemplateDataProdBKC(t *testing.T) {
	type args struct {
		name            string
		rootPart        string
		rootPartNo      string
		hostIP          string
		clientIP        string
		clientID        string
		clientSecret    string
		gateway         string
		in8             string
		in9             string
		securityFeature uint32
		enableDI        bool
		tinkerVersion   string
		hostname        string
	}
	wf := Workflow{}
	want, _ := marshalWorkflow(&wf)
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				name: "TestWorkflow",
			},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTemplateDataProdBKC(tt.args.name, tt.args.rootPart, tt.args.rootPartNo, tt.args.hostIP, tt.args.clientIP, tt.args.gateway, tt.args.in8, tt.args.in9, tt.args.securityFeature, tt.args.clientID, tt.args.clientSecret, tt.args.enableDI, tt.args.tinkerVersion, tt.args.hostname)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTemplateDataProdBKC() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTemplateDataProdBKC() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewTemplateDataProdMS(t *testing.T) {
	type args struct {
		name       string
		rootPart   string
		rootPartNo string
		hostIP     string
		clientIP   string
		gateway    string
		mac        string
		provIP     string
	}
	wf := Workflow{}
	want, _ := marshalWorkflow(&wf)
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr bool
	}{
		{
			name: "Test Case 1",
			args: args{
				name: "TestWorkflow",
			},
			want:    want,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewTemplateDataProdMS(tt.args.name, tt.args.rootPart, tt.args.rootPartNo,
				tt.args.hostIP, tt.args.clientIP, tt.args.gateway, tt.args.mac, tt.args.provIP)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTemplateDataProdMS() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTemplateDataProdMS() = %v, want %v", got, tt.want)
			}
		})
	}
}
