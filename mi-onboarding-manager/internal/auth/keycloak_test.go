// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package auth

import (
	"context"
	"reflect"
	"testing"

	"github.com/Nerzal/gocloak/v13"
)

func Test_newKeycloakSecretService(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    AuthService
		wantErr bool
	}{
		{
			name: "Test Case",
			args: args{
				ctx: context.Background(),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newKeycloakSecretService(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("newKeycloakSecretService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newKeycloakSecretService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getEdgeNodeClientFromTemplate(t *testing.T) {
	type args struct {
		uuid string
	}
	tests := []struct {
		name string
		args args
		want gocloak.Client
	}{
		{
			name: "Test Case",
			args: args{
				uuid: "",
			},
			want: gocloak.Client{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getEdgeNodeClientFromTemplate(tt.args.uuid); reflect.DeepEqual(got, tt.want) {
				t.Errorf("getEdgeNodeClientFromTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_keycloakService_login(t *testing.T) {
	type fields struct {
		keycloakClient *gocloak.GoCloak
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx         context.Context
		keycloakURL string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test Case",
			fields: fields{
				keycloakClient: &gocloak.GoCloak{},
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx:         context.Background(),
				keycloakURL: "",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &keycloakService{
				keycloakClient: tt.fields.keycloakClient,
				jwtToken:       tt.fields.jwtToken,
			}
			if err := k.login(tt.args.ctx, tt.args.keycloakURL); (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.login() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

