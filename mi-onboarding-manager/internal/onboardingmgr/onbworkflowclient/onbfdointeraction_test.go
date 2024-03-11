/*
Copyright (C) 2023 Intel Corporation
SPDX-License-Identifier: Apache-2.0
*/

package onbworkflowclient

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSendFileToOwner(t *testing.T) {
	type args struct {
		ownerIP        string
		ownerSvcPort   string
		guid           string
		clientidsuffix string
		key            string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test Case",
			args:    args{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SendFileToOwner(tt.args.ownerIP, tt.args.ownerSvcPort, tt.args.guid, tt.args.clientidsuffix, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("SendFileToOwner() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSendFileToOwner_Case(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:58040")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/owner/resource?filename=123_suf" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	type args struct {
		ownerIP        string
		ownerSvcPort   string
		guid           string
		clientidsuffix string
		key            string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ownerIP:        "localhost",
				ownerSvcPort:   "58040",
				guid:           "123",
				clientidsuffix: "suf",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SendFileToOwner(tt.args.ownerIP, tt.args.ownerSvcPort, tt.args.guid, tt.args.clientidsuffix, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("SendFileToOwner() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestExecuteSVI(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:58040")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/owner/svi" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Mock owner voucher response"))
		} else {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	server.Listener = listener
	server.Start()
	defer server.Close()
	type args struct {
		ownerIP            string
		ownerSvcPort       string
		clientidsuffix     string
		clientsecretsuffix string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ownerIP:      "localhost",
				ownerSvcPort: "58040",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ExecuteSVI(tt.args.ownerIP, tt.args.ownerSvcPort, tt.args.clientidsuffix, tt.args.clientsecretsuffix); (err != nil) != tt.wantErr {
				t.Errorf("ExecuteSVI() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
