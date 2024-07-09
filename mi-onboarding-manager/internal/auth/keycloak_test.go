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
			name: "Test case for login to Keyclock",
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
			name: "Test Case for to get edge node client from templete",
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
			name: "Test Case for keyclock service login",
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

func Test_keycloakService_fetchAndSetDefaultEdgeNodeClientRoles(t *testing.T) {
	type fields struct {
		keycloakClient *gocloak.GoCloak
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			fields: fields{
				keycloakClient: gocloak.NewClient(""),
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
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
			if err := k.fetchAndSetDefaultEdgeNodeClientRoles(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.fetchAndSetDefaultEdgeNodeClientRoles() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_keycloakService_getServiceAccountUserIDByClientID(t *testing.T) {
	type fields struct {
		keycloakClient *gocloak.GoCloak
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx        context.Context
		clientName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			fields: fields{
				keycloakClient: gocloak.NewClient(""),
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
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
			got, err := k.getServiceAccountUserIDByClientID(tt.args.ctx, tt.args.clientName)
			if (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.getServiceAccountUserIDByClientID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("keycloakService.getServiceAccountUserIDByClientID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_keycloakService_addDefaultRolesToEdgeNodeClient(t *testing.T) {
	type fields struct {
		keycloakClient *gocloak.GoCloak
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx        context.Context
		enClientID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			fields: fields{
				keycloakClient: gocloak.NewClient(""),
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
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
			if err := k.addDefaultRolesToEdgeNodeClient(tt.args.ctx, tt.args.enClientID); (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.addDefaultRolesToEdgeNodeClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_keycloakService_CreateCredentialsWithUUID(t *testing.T) {
	type fields struct {
		keycloakClient *gocloak.GoCloak
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx  context.Context
		uuid string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			fields: fields{
				keycloakClient: gocloak.NewClient(""),
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
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
			got, got1, err := k.CreateCredentialsWithUUID(tt.args.ctx, tt.args.uuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.CreateCredentialsWithUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("keycloakService.CreateCredentialsWithUUID() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("keycloakService.CreateCredentialsWithUUID() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_keycloakService_GetCredentialsByUUID(t *testing.T) {
	type fields struct {
		keycloakClient *gocloak.GoCloak
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx  context.Context
		uuid string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		want1   string
		wantErr bool
	}{
		{
			fields: fields{
				keycloakClient: gocloak.NewClient(""),
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
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
			got, got1, err := k.GetCredentialsByUUID(tt.args.ctx, tt.args.uuid)
			if (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.GetCredentialsByUUID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("keycloakService.GetCredentialsByUUID() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("keycloakService.GetCredentialsByUUID() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_keycloakService_RevokeCredentialsByUUID(t *testing.T) {
	type fields struct {
		keycloakClient *gocloak.GoCloak
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx  context.Context
		uuid string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			fields: fields{
				keycloakClient: gocloak.NewClient(""),
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
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
			if err := k.RevokeCredentialsByUUID(tt.args.ctx, tt.args.uuid); (err != nil) != tt.wantErr {
				t.Errorf("keycloakService.RevokeCredentialsByUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_keycloakService_Logout(t *testing.T) {
	type fields struct {
		keycloakClient *gocloak.GoCloak
		jwtToken       *gocloak.JWT
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			fields: fields{
				keycloakClient: gocloak.NewClient(""),
				jwtToken:       &gocloak.JWT{},
			},
			args: args{
				ctx: context.Background(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &keycloakService{
				keycloakClient: tt.fields.keycloakClient,
				jwtToken:       tt.fields.jwtToken,
			}
			k.Logout(tt.args.ctx)
		})
	}
}
