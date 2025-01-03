# How to obtain client_id and client_secret for Onboarding Manager and store them in Vault?

```bash
curl --location --request POST 'http://localhost:8090/realms/master/protocol/openid-connect/token' \
--header 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'grant_type=password' \
--data-urlencode 'client_id=ledge-park-system' \
--data-urlencode 'username=lp-admin-user' \
--data-urlencode 'password=ChangeMeOn1stLogin!' \
--data-urlencode 'scope=openid profile email groups'
```

Copy access token and save it to `payload.json` (note: `iam-admin` role is required to give write permissions to /secret): 

```json
{
  "role": "iam-admin",
  "jwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

Onboarding Manager will use `host-manager-m2m-client` client name. 

Obtain `client_id` and `client_secret` for `host-manager-m2m-client`:

```bash
curl http://localhost:8090/admin/realms/master/clients\?clientId\=host-manager-m2m-client \
-H "Content-Type: application/json" \
-H  "Authorization: Bearer ${JWT_TOKEN}
# output
[{"id":"e94f1ef4-82f3-4730-a819-e645eca7a515","clientId":"host-manager-m2m-client","name":"Host Manager Client","description":"...
```

Keycloak's `client_id` for `host-manager-m2m-client` is `e94f1ef4-82f3-4730-a819-e645eca7a515`. Obtain `client_secret`:

```bash
# Set the secret in an environment variable
export CLIENT_SECRET="client-secret"
```

```bash
curl http://localhost:8090/admin/realms/master/clients/e94f1ef4-82f3-4730-a819-e645eca7a515/client-secret \
-H "Content-Type: application/json" \
-H  "Authorization: Bearer ${JWT_TOKEN}" \
-d '{"type":"secret","value":"'"${CLIENT_SECRET}"'"}'
```

We have both `id` and `client_secret` at this point. Now, we store credentials to Vault.

Log in to Vault first:

```bash
curl \
    --request POST \
    -H "Content-Type: application/json" --data @payload.json \
    http://localhost:8200/v1/auth/jwt/login
{"request_id":"42530750-da7c-b021-06de-87464b8d459e","lease_id":"","renewable":false,"lease_duration":0,"data":null,"wrap_info":null,"warnings":null,"auth":{"client_token":"hvs.CAESIJa-t5NaZytTJ-p1oYpzKbAN1Y4dqQuhMBHn1f2uWDErGh4KHGh2cy5IVUVJZjdvcnlNcHpnbU9KN0ttYkQ3bzM","accessor":"zDpyKDrpabSd9Kz4TJMxXT2i","policies":["default","lp-admin"],"token_policies":["default","lp-admin"],"metadata":{"role":"lp-admin"},"lease_duration":3600,"renewable":true,"entity_id":"775ad6da-833f-2165-3306-c97b9fac46b8","token_type":"service","orphan":true,"mfa_requirement":null,"num_uses":0}}
```

Note the `client_token` above. It will be later used as `X-Vault-Token:`. 

Craft the `secret.json` file with Keycloak credentials to store:

Replace actual secrets with placeholder values in the json

```json
{
  "data": {
    "client_id": "e94f1ef4-82f3-4730-a819-e645eca7a515",
    "client_secret": "client-secret-placeholder"
  }
}
```

Then, send request to Vault to create a new secret (`host-manager-m2m-client-secret`):

```bash
curl \
    --header "X-Vault-Token: hvs.CAESIL7gErRERzkiikKdNAcSYkIi4nMqZT3cqb8DDR39Hhb-Gh4KHGh2cy5FR3RjNG9VMVlFbnY5RXU0ZkdVdjg2Zkc" \
    --request POST \
    --data @secret.json \
    http://127.0.0.1:8200/v1/secret/data/host-manager-m2m-client-secret
```

At this point the secret is stored in Vault and Onboarding Manager will be able to read it.  

Verify that the secret has been successfully created (not part of secret onboarding workflow):

NOTE: Actual client secrete will be fetched in place of "client-secret-placeholder"
VAULT_TOKEN should be replaced by the actual Vault token when the command is executed

```bash
# Set the VAULT_TOKEN in an environment variable
export VAULT_TOKEN="actual-vault-token"
```

```bash
curl \
    --header "X-Vault-Token: $VAULT_TOKEN" \
    --request GET http://127.0.0.1:8200/v1/secret/data/host-manager-m2m-client-secret
{"request_id":"39f5f0ae-2880-be2e-1061-a342b1988f6b","lease_id":"","renewable":false,"lease_duration":0,"data":{"data":{"client_id":"e94f1ef4-82f3-4730-a819-e645eca7a515","client_secret":"client-secret-placeholder"},"metadata":{"created_time":"2024-02-08T11:57:05.307142Z","custom_metadata":null,"deletion_time":"","destroyed":false,"version":2}},"wrap_info":null,"warnings":null,"auth":null}
```
