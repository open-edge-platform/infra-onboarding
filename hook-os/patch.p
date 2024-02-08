diff --git a/hook-bootkit/Dockerfile b/hook-bootkit/Dockerfile
index 5c38880..d875feb 100644
--- a/hook-bootkit/Dockerfile
+++ b/hook-bootkit/Dockerfile
@@ -1,4 +1,14 @@
 FROM golang:1.20-alpine as dev
+ENV http_proxy "FIX_H_TTP_PROXY"
+ENV https_proxy "FIX_H_TTPS_PROXY"
+ENV HTTP_PROXY "FIX_H_TTP_PROXY"
+ENV HTTPS_PROXY "FIX_H_TTPS_PROXY"
+
+RUN export http_proxy=FIX_H_TTP_PROXY
+RUN export https_proxy=FIX_H_TTPS_PROXY
+RUN export HTTPS_PROXY=FIX_H_TTPS_PROXY
+RUN export HTTPS_PROXY=FIX_H_TTPS_PROXY
+
 COPY . /src/
 WORKDIR /src
 RUN go mod download
diff --git a/hook-docker/Dockerfile b/hook-docker/Dockerfile
index da5bde6..026141a 100644
--- a/hook-docker/Dockerfile
+++ b/hook-docker/Dockerfile
@@ -1,9 +1,26 @@
 FROM golang:1.20-alpine as dev
+ENV http_proxy "FIX_H_TTP_PROXY"
+ENV https_proxy "FIX_H_TTPS_PROXY"
+ENV HTTP_PROXY "FIX_H_TTP_PROXY"
+ENV HTTPS_PROXY "FIX_H_TTPS_PROXY"
+RUN export http_proxy=FIX_H_TTP_PROXY
+RUN export https_proxy=FIX_H_TTPS_PROXY
+RUN export HTTPS_PROXY=FIX_H_TTPS_PROXY
+RUN export HTTPS_PROXY=FIX_H_TTPS_PROXY
+
 COPY . /src/
 WORKDIR /src
 RUN CGO_ENABLED=0 go build -a -ldflags '-w -extldflags "-static"' -o /hook-docker
 
 FROM docker:24.0.4-dind
+ENV http_proxy "FIX_H_TTP_PROXY"
+ENV https_proxy "FIX_H_TTPS_PROXY"
+ENV HTTP_PROXY "FIX_H_TTP_PROXY"
+ENV HTTPS_PROXY "FIX_H_TTPS_PROXY"
+RUN export http_proxy=FIX_H_TTP_PROXY
+RUN export https_proxy=FIX_H_TTPS_PROXY
+RUN export HTTPS_PROXY=FIX_H_TTPS_PROXY
+RUN export HTTPS_PROXY=FIX_H_TTPS_PROXY
 RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/testing" >> /etc/apk/repositories
 RUN apk update; apk add kexec-tools
 COPY --from=dev /hook-docker .
diff --git a/hook-docker/main.go b/hook-docker/main.go
index 0908c72..94d4299 100644
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
@@ -58,7 +68,7 @@ func main() {
 	}
 
 	// Build the command, and execute
-	cmd := exec.Command("/usr/local/bin/docker-init", "/usr/local/bin/dockerd")
+	cmd = exec.Command("/usr/local/bin/docker-init", "/usr/local/bin/dockerd")
 	cmd.Stdout = os.Stdout
 	cmd.Stderr = os.Stderr
 
diff --git a/hook.yaml b/hook.yaml
index 647e792..060d287 100644
--- a/hook.yaml
+++ b/hook.yaml
@@ -34,6 +34,25 @@ onboot:
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
@@ -63,6 +82,11 @@ services:
     binds:
       - /var/run:/var/run
 
+  - name: fluent-bit
+    image: fluent/fluent-bit:2.1.9
+    binds.add:
+      - /etc/fluent-bit/fluent-bit.conf:/fluent-bit/etc/fluent-bit.conf
+
   - name: hook-docker
     image: quay.io/tinkerbell/hook-docker:latest
     capabilities:
@@ -80,6 +104,7 @@ services:
       - /var/run/docker:/var/run
       - /var/run/images:/var/lib/docker
       - /var/run/worker:/worker
+      - /dev/shm:/dev/shm
     runtime:
       mkdir:
         - /var/run/images
@@ -100,6 +125,38 @@ services:
       mkdir:
         - /var/run/docker
 
+  - name: nginx
+    image: nginx_proxy_action:latest
+    capabilities:
+      - all
+    binds.add:
+      - /etc/resolv.conf:/etc/resolv.conf
+      - /etc/idp/server_cert.pem:/usr/local/share/ca-certificates/maestro.crt
+      - /etc/idp/server_cert.pem:/etc/nginx/ssl/ensp-orchestrator-ca.crt
+      - /etc/nginx/templates/nginx.conf.template:/etc/nginx/templates/nginx.conf.template
+      - /dev/shm/idp_access_token:/dev/shm/idp_access_token
+      - /dev/shm/release_token:/dev/shm/release_token
+
+    # Intended docker variables to be populated from environment
+    env:
+      - tink_svc=update_tink_svc
+      - release_svc=update_release_svc
+      - manufacturer_svc=update_manufacturer_svc
+      - owner_svc=update_owner_svc
+      - oci_release_svc=update_oci_release_svc
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
 
@@ -110,6 +167,14 @@ files:
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
@@ -137,6 +202,14 @@ files:
     source: "files/dhcpcd.conf"
     mode: "0644"
 
+  - path: /etc/fluent-bit/fluent-bit.conf
+    source: "files/fluent-bit/fluent-bit.conf"
+    mode: "0644"
+
+  - path: etc/nginx/templates/nginx.conf.template
+    source: "files/nginx/nginx.conf.template"
+    mode: "0644"
+
 #dbg  - path: root/.ssh/authorized_keys
 #dbg    source: ~/.ssh/id_rsa.pub
 #dbg    mode: "0600"
@@ -146,3 +219,12 @@ trust:
   org:
     - linuxkit
     - library
+
+onshutdown:
+  - name: efibootset
+    image: efibootset:latest
+    capabilities:
+      - all
+    binds.add:
+      - /dev:/dev
+      - /dev/console:/dev/console
diff --git a/rules.mk b/rules.mk
index b2c5133..7b1da7b 100644
--- a/rules.mk
+++ b/rules.mk
@@ -87,13 +87,12 @@ push-hook-bootkit push-hook-docker:
 	docker buildx build --platform $$platforms --push -t $(ORG)/$(container):$T $(container)
 
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
