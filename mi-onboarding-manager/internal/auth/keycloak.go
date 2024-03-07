// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package auth

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.secure-os-provision-onboarding-service/internal/secrets"
	inv_errors "github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/errors"
	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.inventory/pkg/logging"
	"google.golang.org/grpc/codes"
)

var zlog = logging.GetLogger("OMKeycloakService")

const (
	DefaultKeycloakURL = "http://platform-keycloak.maestro-platform-system:8080"

	EnvNameKeycloakURL = "KEYCLOAK_URL"

	KeycloakRealm               = "master"
	OnboardingManagerClientName = "host-manager-m2m-client"

	ENCredentialsPrefix = "edgenode-"
)

var AuthServiceFactory = newKeycloakSecretService

type keycloakService struct {
	keycloakClient *gocloak.GoCloak
	jwtToken       *gocloak.JWT
}

func newKeycloakSecretService(ctx context.Context) (AuthService, error) {
	kss := &keycloakService{}

	keycloakURL := os.Getenv(EnvNameKeycloakURL)
	if keycloakURL == "" {
		zlog.MiSec().Warn().Msgf("%s env variable is not set, using default value", EnvNameKeycloakURL)
		keycloakURL = DefaultKeycloakURL
	}

	err := kss.login(ctx, keycloakURL)
	if err != nil {
		return nil, err
	}
	return kss, nil
}

func (k *keycloakService) login(ctx context.Context, keycloakURL string) error {
	client := gocloak.NewClient(keycloakURL)

	jwtToken, err := client.LoginClient(ctx, OnboardingManagerClientName, secrets.GetClientSecret(), KeycloakRealm)
	if err != nil {
		errMsg := "Failed to login to Keycloak"
		zlog.MiSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf(errMsg)
	}

	k.keycloakClient = client
	k.jwtToken = jwtToken

	zlog.MiSec().Debug().Msgf("Keycloak client logged in successfully")
	return nil
}

func (k *keycloakService) CreateCredentialsWithUUID(ctx context.Context, uuid string) (string, string, error) {
	edgeNodeClient := getEdgeNodeClientFromTemplate(uuid)

	zlog.Info().Msgf("Creating Keycloak credentials for host %s", uuid)

	id, err := k.keycloakClient.CreateClient(ctx, k.jwtToken.AccessToken, KeycloakRealm, edgeNodeClient)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to create Keycloak client with UUID %s", uuid)
		zlog.MiSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorf(errMsg)
	}

	zlog.MiSec().Debug().Msgf("Keycloak credentials for host %s created successfully, ID: %s",
		uuid, id)

	creds, err := k.keycloakClient.GetClientSecret(ctx, k.jwtToken.AccessToken, KeycloakRealm, id)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get Keycloak client secret for client ID %s (host UUID %s)", id, uuid)
		zlog.MiSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorf(errMsg)
	}

	if creds.Value == nil {
		err = inv_errors.Errorf("Received empty client secret for client ID %s (host UUID %s)", id, uuid)
		zlog.MiSec().Err(err).Msg("")
		return "", "", err
	}

	zlog.MiSec().Debug().Msgf("Keycloak client secret for host %s obtained successfully, ID: %s",
		uuid, id)

	return *edgeNodeClient.ClientID, *creds.Value, nil
}

func (k *keycloakService) GetCredentialsByUUID(ctx context.Context, uuid string) (string, string, error) {
	edgeNodeClientID := getEdgenodeClientName(uuid)

	zlog.Info().Msgf("Getting Keycloak credentials for host %s", uuid)

	clients, err := k.keycloakClient.GetClients(ctx, k.jwtToken.AccessToken, KeycloakRealm, gocloak.GetClientsParams{
		ClientID: &edgeNodeClientID,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Keycloak client for edge node by UUID %s does not exist", uuid)
		zlog.MiSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorf(errMsg)
	}

	if len(clients) == 0 {
		errMsg := fmt.Sprintf("No Keycloak clients found for UUID %s", uuid)
		zlog.MiSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorfc(codes.NotFound, errMsg)
	}

	// This should never happen but we could have more than one Keycloak client for a UUID.
	// We print warning and get first.
	if len(clients) > 1 {
		zlog.Warn().Msgf("More than one Keycloak client found for UUID %s, getting first one", uuid)
	}

	secret := clients[0].Secret
	// if we received secret as part of GetClients(), return it. Otherwise, use GetClientSecret().
	if secret != nil {
		return edgeNodeClientID, *secret, nil
	}

	id := *clients[0].ID
	creds, err := k.keycloakClient.GetClientSecret(ctx, k.jwtToken.AccessToken, KeycloakRealm, id)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get Keycloak client secret for client ID %s (host UUID %s)", id, uuid)
		zlog.MiSec().Err(err).Msg(errMsg)
		return "", "", inv_errors.Errorf(errMsg)
	}

	if creds.Value == nil {
		err = inv_errors.Errorf("Received empty client secret for client ID %s (host UUID %s)", id, uuid)
		zlog.MiSec().Err(err).Msg("")
		return "", "", err
	}

	return edgeNodeClientID, *creds.Value, nil
}

func (k *keycloakService) RevokeCredentialsByUUID(ctx context.Context, uuid string) error {
	edgeNodeClientID := getEdgenodeClientName(uuid)

	clients, err := k.keycloakClient.GetClients(ctx, k.jwtToken.AccessToken, KeycloakRealm, gocloak.GetClientsParams{
		ClientID: &edgeNodeClientID,
	})
	if err != nil {
		errMsg := fmt.Sprintf("Keycloak client for edge node by UUID %s does not exist", uuid)
		zlog.MiSec().Err(err).Msg(errMsg)
		return inv_errors.Errorf(errMsg)
	}

	if len(clients) == 0 {
		errMsg := fmt.Sprintf("No Keycloak clients found for UUID %s", uuid)
		zlog.MiSec().Err(err).Msg(errMsg)
		return inv_errors.Errorfc(codes.NotFound, errMsg)
	}

	// This should never happen but we could have more than one Keycloak client for a UUID.
	// We print warning and remove all clients.
	if len(clients) > 1 {
		zlog.Warn().Msgf("More than one Keycloak client found for UUID %s, deleting all", uuid)
	}

	for _, edgeNodeClient := range clients {
		if edgeNodeClient.ID == nil {
			zlog.Debug().Msgf("Found Keycloak client for UUID %s with empty ID, skipping deletion",
				uuid)
			continue
		}

		err = k.keycloakClient.DeleteClient(ctx, k.jwtToken.AccessToken, KeycloakRealm, *edgeNodeClient.ID)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to delete Keycloak client for edge node with UUID %s", uuid)
			zlog.MiSec().Err(err).Msg(errMsg)
			return inv_errors.Errorf(errMsg)
		}

		zlog.MiSec().Debug().Msgf("Keycloak credentials for host %s revoked successfully, ID: %s",
			uuid, *edgeNodeClient.ID)
	}

	return nil
}

func (k *keycloakService) Logout(ctx context.Context) {
	if err := k.keycloakClient.Logout(ctx, OnboardingManagerClientName, secrets.GetClientSecret(), KeycloakRealm, k.jwtToken.RefreshToken); err != nil {
		zlog.MiSec().Err(err).Msgf("Failed to logout from Keycloak")
		return
	}
}

func getEdgenodeClientName(uuid string) string {
	return fmt.Sprintf("%s%s", ENCredentialsPrefix, uuid)
}

// based on https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-deployment/blob/6589185238d3c17c168b2f072a2588b6621688ae/helmfile.d/environments/mp-keycloak/base/values.yaml#L366
func getEdgeNodeClientFromTemplate(uuid string) gocloak.Client {
	description := fmt.Sprintf("Client to use by Edge Node %s, created by Onboarding Manager at %s",
		uuid, time.Now().UTC().String())
	clientID := getEdgenodeClientName(uuid)
	name := fmt.Sprintf("Edge Node %s", uuid)
	authTypeClientSecret := "client-secret"
	protocolOpenidConnect := "openid-connect"
	boolTrue := true
	boolFalse := false
	zero := int32(0)
	attributes := map[string]string{
		"oidc.ciba.grant.enabled":                   "false",
		"oauth2.device.authorization.grant.enabled": "false",
		"backchannel.logout.revoke.offline.tokens":  "false",
	}
	defaultClientScopes := []string{
		"web-origins",
		"acr",
		"profile",
		"roles",
		"email",
	}
	optionalClientScopes := []string{
		"address",
		"phone",
		"offline_access",
		"microprofile-jwt",
	}
	return gocloak.Client{
		ClientID:                  &clientID,
		Name:                      &name,
		Description:               &description,
		SurrogateAuthRequired:     &boolFalse,
		Enabled:                   &boolTrue,
		ClientAuthenticatorType:   &authTypeClientSecret,
		NotBefore:                 &zero,
		BearerOnly:                &boolFalse,
		ConsentRequired:           &boolFalse,
		StandardFlowEnabled:       &boolFalse,
		ImplicitFlowEnabled:       &boolFalse,
		DirectAccessGrantsEnabled: &boolFalse,
		ServiceAccountsEnabled:    &boolTrue,
		PublicClient:              &boolFalse,
		Protocol:                  &protocolOpenidConnect,
		Attributes:                &attributes,
		FullScopeAllowed:          &boolTrue,
		DefaultClientScopes:       &defaultClientScopes,
		OptionalClientScopes:      &optionalClientScopes,
	}
}
