diff --git a/hook-docker/main.go b/hook-docker/main.go
index 0908c72..e5998bf 100644
--- a/hook-docker/main.go
+++ b/hook-docker/main.go
@@ -29,6 +29,16 @@ func main() {
 	fmt.Println("Starting Tink-Docker")
 	go rebootWatch()
 
+         fmt.Println("Make /dev/null writeable for all users!")
+         cmd := exec.Command("chmod", "666", "/dev/null")
+         cmd.Stdout = os.Stdout
+         cmd.Stderr = os.Stderr
+         err := cmd.Run()
+         if err != nil {
+             panic(err)
+         }
+
+
 	// Parse the cmdline in order to find the urls for the repository and path to the cert
 	content, err := os.ReadFile("/proc/cmdline")
 	if err != nil {
@@ -43,7 +53,7 @@ func main() {
 		Debug:     true,
 		LogDriver: "syslog",
 		LogOpts: map[string]string{
-			"syslog-address": fmt.Sprintf("udp://%v:514", cfg.syslogHost),
+			"syslog-address": fmt.Sprintf("udp://%v:5140", cfg.syslogHost),
 		},
 		InsecureRegistries: cfg.insecureRegistries,
 	}
@@ -58,7 +68,7 @@ func main() {
 	}
 
 	// Build the command, and execute
-	cmd := exec.Command("/usr/local/bin/docker-init", "/usr/local/bin/dockerd")
+	cmd = exec.Command("/usr/local/bin/docker-init", "/usr/local/bin/dockerd")
 	cmd.Stdout = os.Stdout
 	cmd.Stderr = os.Stderr
 
@@ -121,6 +131,8 @@ func rebootWatch() {
 			cmd := exec.Command("/sbin/reboot")
 			cmd.Stdout = os.Stdout
 			cmd.Stderr = os.Stderr
+			// wait 3 sec to do actual reboot before workflow send back success status
+			time.Sleep(3*time.Second)
 			err := cmd.Run()
 			if err != nil {
 				panic(err)
diff --git a/hook.yaml b/hook.yaml
index 647e792..a3b71df 100644
--- a/hook.yaml
+++ b/hook.yaml
@@ -34,6 +34,26 @@ onboot:
       mkdir:
         - /var/lib/dhcpcd
 
+  - name: client_auth
+    image: client_auth:latest
+    capabilities:
+      - all
+    binds:
+      - /dev:/dev
+      - /dev/console:/dev/console
+      - /dev/ttyS0:/dev/ttyS0
+      - /etc/resolv.conf:/etc/resolv.conf
+      - /etc/idp/server_cert.pem:/usr/local/share/ca-certificates/IDP_keyclock.crt
+      - /var:/var:rshared,rbind
+      - /dev/shm:/dev/shm
+      - /etc/hook/env_config:/etc/hook/env_config
+    rootfsPropagation: shared
+    env:
+      - CLIENT_AUTH_PRE_BIND=TRUE
+      - KEYCLOAK_URL=update_idp_url
+      - EXTRA_HOSTS=update_extra_hosts
+
+
 services:
   - name: getty
     image: linuxkit/getty:76951a596aa5e0867a38e28f0b94d620e948e3e8
@@ -63,6 +83,13 @@ services:
     binds:
       - /var/run:/var/run
 
+  - name: fluent-bit
+    image: fluent/fluent-bit:2.1.9
+    binds.add:
+      - /etc/fluent-bit/fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf
+      - /var/log:/var/log
+    rootfsPropagation: shared
+
   - name: hook-docker
     image: quay.io/tinkerbell/hook-docker:latest
     capabilities:
@@ -80,6 +107,7 @@ services:
       - /var/run/docker:/var/run
       - /var/run/images:/var/lib/docker
       - /var/run/worker:/worker
+      - /dev/shm:/dev/shm
     runtime:
       mkdir:
         - /var/run/images
@@ -100,6 +128,41 @@ services:
       mkdir:
         - /var/run/docker
 
+  - name: caddy
+    image: caddy_proxy:latest
+    capabilities:
+      - all
+    binds.add:
+      - /etc/resolv.conf:/etc/resolv.conf
+      - /etc/idp/ca.pem:/etc/caddy/ensp-orchestrator-ca.crt
+      - /etc/caddy/Caddyfile:/etc/caddy/Caddyfile
+      - /dev/shm/idp_access_token:/dev/shm/idp_access_token
+      - /dev/shm/release_token:/dev/shm/release_token
+      - /etc/hook/env_config:/etc/hook/env_config
+
+    # Intended docker variables to be populated from environment
+    env:
+      - tink_stack_svc=update_tink_stack_svc
+      - tink_server_svc=update_tink_server_svc
+      - release_svc=update_release_svc
+      - logging_svc=update_logging_svc
+      - fdo_manufacturer_svc=update_manufacturer_svc
+      - fdo_owner_svc=update_owner_svc
+      - oci_release_svc=update_oci_release_svc
+      - EXTRA_HOSTS=update_extra_hosts
+
+  - name: fdo
+    image: fdoclient_action:latest
+    capabilities:
+      - all
+    binds.add:
+      - /dev:/dev
+      - /dev/console:/dev/console
+    env:
+      - FDO_RUN_TYPE=to
+      - DATA_PARTITION_LBL=CREDS
+      - FDO_TLS=https
+
 #dbg  - name: sshd
 #dbg    image: linuxkit/sshd:666b4a1a323140aa1f332826164afba506abf597
 
@@ -110,6 +173,14 @@ files:
       alias docker-shell='ctr -n services.linuxkit tasks exec --tty --exec-id shell hook-docker sh'
     mode: "0644"
 
+  - path: etc/idp/ca.pem
+    source: files/idp/ca.pem
+    mode: "0644"
+
+  - path: etc/idp/server_cert.pem
+    source: files/idp/server_cert.pem
+    mode: "0644"
+
   - path: etc/motd
     mode: "0644"
     contents: |
@@ -137,6 +208,18 @@ files:
     source: "files/dhcpcd.conf"
     mode: "0644"
 
+  - path: /etc/fluent-bit/fluent-bit.conf
+    source: "files/fluent-bit/fluent-bit.conf"
+    mode: "0644"
+
+  - path: etc/caddy/Caddyfile
+    source: "files/caddy/Caddyfile"
+    mode: "0644"
+
+  - path: etc/hook/env_config
+    contents: ""
+    mode: "0644"
+
 #dbg  - path: root/.ssh/authorized_keys
 #dbg    source: ~/.ssh/id_rsa.pub
 #dbg    mode: "0600"
@@ -146,3 +229,4 @@ trust:
   org:
     - linuxkit
     - library
+
diff --git a/rules.mk b/rules.mk
index b2c5133..93fcdea 100644
--- a/rules.mk
+++ b/rules.mk
@@ -68,7 +68,12 @@ out/$T/hook-docker-$(arch): $$(hook-docker-deps)
 out/$T/hook-bootkit-$(arch) out/$T/hook-docker-$(arch): platform=linux/$$(lastword $$(subst -, ,$$(notdir $$@)))
 out/$T/hook-bootkit-$(arch) out/$T/hook-docker-$(arch): container=hook-$$(word 2,$$(subst -, ,$$(notdir $$@)))
 out/$T/hook-bootkit-$(arch) out/$T/hook-docker-$(arch):
-	docker buildx build --platform $$(platform) --load -t $(ORG)/$$(container):$T-$(arch) $$(container)
+	docker buildx build --platform $$(platform)  \
+		--build-arg HTTP_PROXY=${http_proxy} \
+		--build-arg HTTPS_PROXY=${https_proxy} \
+		--build-arg http_proxy=${http_proxy} \
+		--build-arg https_proxy=${https_proxy} \
+		--load -t $(ORG)/$$(container):$T-$(arch) $$(container)
 	touch $$@
 
 run-$(arch): out/$T/dbg/$(arch)/hook.tar
@@ -84,16 +89,20 @@ push-hook-bootkit push-hook-docker: container=hook-$(lastword $(subst -, ,$(base
 push-hook-bootkit push-hook-docker:
 	platforms="$(platforms)"
 	platforms=$${platforms// /,}
-	docker buildx build --platform $$platforms --push -t $(ORG)/$(container):$T $(container)
+	docker buildx build --platform $$platforms \
+		--build-arg HTTP_PROXY=${http_proxy} \
+		--build-arg HTTPS_PROXY=${https_proxy} \
+		--build-arg http_proxy=${http_proxy} \
+		--build-arg https_proxy=${https_proxy} \
+		--push -t $(ORG)/$(container):$T $(container)
 
 .PHONY: dist
-dist: out/$T/rel/amd64/hook.tar out/$T/rel/arm64/hook.tar ## Build tarballs for distribution
+dist: out/$T/rel/amd64/hook.tar  ## Build tarballs for distribution
 dbg-dist: out/$T/dbg/$(ARCH)/hook.tar ## Build debug enabled tarball
 dist dbg-dist:
 	for f in $^; do
 	case $$f in
 	*amd64*) arch=x86_64 ;;
-	*arm64*) arch=aarch64 ;;
 	*) echo unknown arch && exit 1;;
 	esac
 	d=$$(dirname $$(dirname $$f))
