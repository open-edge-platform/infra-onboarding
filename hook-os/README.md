
# Hook microOS (Alpine Linuxkit)
This repository holds the scripts needed to get a local copy of Tinkerbell HookOS which is built out of linuxkit yaml file.
Addition functions accomplished by these scripts are listed below.
1. Updates the implementation of HookOS.
2. Signing the HookOS
3. Creating a grub needed to boot HookOS from disk and into RAM

# Build pre-requisites
1. The build requires few containers to be embedded into the hook micro OS. These can be found in the actions repository.
   The version of these container will be specified by the file VERSION in the root dir of this repo.
2. The build requires these container image to be present before running the build script. Verify that these containers are present by running "docker images" cmd.

# Build steps for HookOS
1. update the config file with the correct configurations.
   Essential configurations include
   http_proxy, https_proxy, ftp_proxy, socks_proxy, no_proxy, nameserver

   ```
   Example:
   http_proxy=http://xyz.com:911
   https_proxy=https://xyz.com:911
   ftp_proxy=ftp://xyz.com:911
   socks_proxy=socks://xyz.com:911
   no_proxy=localhost
   nameserver=(192.168.1.1 192.168.1.2 192.168.1.3)
   ```
2. Update NGINX runtime configurations according to Maestro deployment.

   ```
   Example:
   fdo_manufacturer_svc="fdo-mfg.kind.internal"
   fdo_owner_svc="fdo-owner.kind.internal"
   release_svc="files.internal.ledgepark.intel.com"
   tinker_svc="tink-stack.kind.internal"
   ```
3. [Optional] Update host IP/FQDN mapping(comma separated) values in extra_hosts if Maestro is deployed on Kind cluster

   ```
   Example:
   extra_hosts="10.114.181.238 api-proxy.kind.internal,10.114.181.238 app-orch.kind.internal,10.114.181.238 cluster-orch-edge-node.kind.internal,10.114.181.238 fdo-mfg.kind.internal,10.114.181.238 fdo-owner.kind.internal,10.114.181.238 tink-stack.kind.internal"
   ```
4. Run the build hookOS.

   ```
   bash ./build_hookOS.sh
   ```

## Customization of builds
1. secure_hookos.sh will create gpg keys and uses it from folder gpg_keys and public portion is boot.key
   If this folder is kept in the root of the folder no new gpg keys will be created.
   If you need to use a custom gpg key replace the gpg_keys and boot.key.

2. secure_hookos.sh will create secure boot keys and uses it from folder sb_keys.
   If this folder is kept in the root of the folder no new secureboot keys will be created.
   If you need to use a custom secureboot key replace the sb_keys.



# Outputs
1. Alpine linux based HookOS will be placed in alpine_image folder.
1. A signed alpine linux based HookOS will be placed in alpine_image_secureboot folder.

