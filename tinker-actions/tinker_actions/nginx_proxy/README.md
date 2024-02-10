# NGINX Client Proxy

NGINX client proxy takes care of facilitating JWT token Authorization for client which can't use JWT implicitly.
Currently NGINX client proxies are configured for FDO, Tink Agent and APT.

## Steps to build and deploy NGINX client proxy

1. Copy Maestro Root CA certificate `ensp-orchestrator-ca.crt` to `/usr/local/share/ca-certificates` directory.
2. Build NGINX docker image with self signed certificate and key
	`bash build.sh`
3. Get JWT token from Keycloak service

	curl -vv -k -X POST https://keycloak.<cluster_fqdn>/realms/master/protocol/openid-connect/token \
    		-d "username=lp-admin-user" -d "password=ChangeMeOn1stLogin!" \
    		-d "grant_type=password" -d "client_id=ledge-park-system" \
    		-d "scope=openid" | jq -r '.access_token'`
	
   Example cluster fqdn - `kind.internal` (for coder based deployment)
4. Use JWT Access token obtained from step #3 in place of `$TOKEN` to get release service token. This is needed for authenticating APT clients against release service
	curl -vv -XGET https://release.<cluster_fqdn>/token -H "Authorization: Bearer $TOKEN"

5. Run NGINX container by using below cmd

	`docker run --name nginx-proxy --network host --env owner_svc="<owner_svc_fqdn>" --env auth_token="<jwit_token>" --env nameserver="<dns_server>" --env manufacturer_svc="<fdo_mfg_svc_fqdn>" --env tink_stack_svc="<tink_stack_fqdn>" --env tink_server_svc="<tink_server_fqdn>" --env release_svc="<release_svc_fqdn>"  --env oci_release_svc="<oci_release_svc_fqdn>" -v $PWD/nginx.conf.template:/etc/nginx/templates/nginx.conf.template -v /usr/local/share/ca-certificates/ensp-orchestrator-ca.crt:/etc/nginx/ssl/ensp-orchestrator-ca.crt  -d nginx_proxy`

NOTE: 
1. Inject JWT token obtained in step 3 as env arg `auth_token`
2. To add customization to proxy routes, add it in location block in nginx.conf.template file and pass it as volume mount in docker run command.
3. Pass Orchestrator root CA certificate as volume mount to NGINX docker container
4. Make sure to add cluster FQDN (for eg, kind.internal) to `no_proxy` list in `/etc/environment` for Coder based deployments
5. manufacuturer_svc and owner_svc FQDNS are not yet available on FM. So as workaround pass any available FM fqdns. `For eg: tink-stack.kind.internal` This is workaround for now to proceed with NGINX proxy for TINK and APT clients.
