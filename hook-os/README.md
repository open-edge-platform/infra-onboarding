# Hook microOS (Alpine Linuxkit)

This repository holds the scripts needed to get a local copy of Tinkerbell
HookOS which is built out of linuxkit yaml file.

Addition functions accomplished by these scripts are listed below.

1. Updates the implementation of HookOS.
2. Signing the HookOS
3. Creating a grub needed to boot HookOS from disk and into RAM

# Build pre-requisites

1. The build requires few containers to be embedded into the hook micro OS,
   which are pulled from the actions repository.

   The version of these container will be specified by the file
   `TINKER_ACTIONS_VERSION` in the root dir of this repo.

2. The build requires these container image to be present before running the
   build script. Verify that these containers are present by running

       docker images

3. Optionally this can be automated by running the `prereqs.sh` script (see
   step 4 below)

# Build steps for HookOS

1. update the config file with the correct configurations.
   Essential configurations include

   http_proxy, https_proxy, ftp_proxy, socks_proxy, no_proxy, nameserver, and
   deployment_dns_extension

   ```
   Example:
   http_proxy=http://xyz.com:911
   https_proxy=https://xyz.com:911
   ftp_proxy=ftp://xyz.com:911
   socks_proxy=socks://xyz.com:911
   no_proxy=localhost
   nameserver=(192.168.1.1 192.168.1.2 192.168.1.3)
   deployment_dns_extension=kind.internal
   ```

2. Update Caddy runtime configurations according to Maestro deployment.

   ```
   Example:
   fdo_manufacturer_svc="fdo-mfg.kind.internal"
   fdo_owner_svc="fdo-owner.kind.internal"
   release_svc="files.internal.ledgepark.intel.com"
   logging_svc="logs-node.kind.internal"
   oci_release_svc="registry-rs.internal.ledgepark.intel.com"
   tink_stack_svc="tinkerbell-nginx.kind.internal"
   tink_server_svc="tinkerbell-server.kind.internal"
   ```

3. [Optional] Update host IP/FQDN mapping(comma separated) values in
   `extra_hosts` if Maestro is deployed on Kind cluster, or in Coder.

   ```
   Example:
   extra_hosts="10.114.181.238 api-proxy.kind.internal,10.114.181.238 app-orch.kind.internal,10.114.181.238 cluster-orch-edge-node.kind.internal,10.114.181.238 fdo-mfg.kind.internal,10.114.181.238 fdo-owner.kind.internal,10.114.181.238 tinkerbell-nginx.kind.internal,10.114.181.238 tinkerbell-server.kind.internal,10.114.181.238 logs-node.kind.internal"
   ```

4. [Optional] Create Intel-specific certificate chain

   Run the cert_prep.sh script

   ```
   bash cert_prep.sh
   ```

   This downloads and creates the `client_auth/files/ca.pem` file with
   Intel internal CAs certs and that of the deployment Vault.

5. [Optional] Pull all tinker action containers

   Run the `prereq.sh` script to download the tinker action images from a
   docker registry.

   ```
   bash prereq.sh
   ```

6. Run the build hookOS.

   ```
   bash build_hookos.sh
   ```

   This will perform the build and create a `hook_x86_64.tar.gz` in both
   the `alpine_image/` and `alpine_image_secureboot/` directories.


7. [Optional] Upload the signed HookOS onto the deployment

   If you run the copy_to_nginx.sh script, the secure version of the HookOS will
   be uploaded to othe `tink-stack` pod's nginx public dir and decompressed:

   ```
   bash copy_to_nginx.sh
   ```

   NOTE: This expects that a K8s config and credentials already exist on the
   local system, as it uses `kubectl` to identify the `tink-stack` pod.

8. [Optional] Copy signing keys

   The HookOS signing keys are located in the `sb_keys/` directory.  You will want
   to install the `db.der` file on the target machine to be able to verify the signed
   images.

   This is also included in the secure version of the `hook_x86_64.tar.gz` archive,
   renamed to be `hookos_db.der`.

Once this is done, you need to add the secure boot keys to the target machine
being booted, then attempt to boot it from the image.

## Customization of builds

1. `secure_hookos.sh` will create gpg keys and uses it from folder gpg_keys and
   public portion is boot.key If this folder is kept in the root of the folder
   no new gpg keys will be created.  If you need to use a custom gpg key
   replace the gpg_keys and boot.key.

2. `secure_hookos.sh` will create secure boot keys and uses it from folder
   sb_keys.  If this folder is kept in the root of the folder no new secureboot
   keys will be created.  If you need to use a custom secureboot key replace
   the sb_keys.

# Outputs

1. Alpine Linux based HookOS will be placed in alpine_image folder.

2. A signed alpine Linux based HookOS will be placed in alpine_image_secureboot folder.
