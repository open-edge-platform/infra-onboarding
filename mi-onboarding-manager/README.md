# Secure Onboarding and provisioning controller

This repo has the service controller module to do onboarding and provisioning
It will do the following things

1. OS image download as mentioned in the profile
2. FDO voucher extension
3. Tinker workflow management - DI and Final OS installation
4. Interfacing with inventory service to do profile management
5. Interfacing with DKAM service to get the curated software artifacts

## Setup

### Create Custom HTTPS supported NGINX server

1. Refer to this section [Server Certificates for HTTPS boot](https://github.com/intel-innersource/documentation.edge.one-edge.maestro/blob/762b2526abd36203f2ee5c20b45ccaea9ebb2140/content/docs/specs/secure-boot.md#server-certificates-for-htts-boot) for creating certificates. The file ```full_server.crt``` will be required in the next steps.

2. Clone the [repository](https://github.com/intel-sandbox/nginx/tree/main)

3. Go inside the repository and build and run the nginx container as per the [README](https://github.com/intel-sandbox/nginx/blob/main/README.md).
As seen in the docker run command example we are mounting two folders to the container, which are referred to in the next steps.

    ```bash
        -v ./certs:/etc/ssl/cert/ \
        -v ./data:/usr/share/nginx/html \
    ```

    ```certs``` : server certificates are present here
    ```data```  : files present here are hosted by the NGINX server

4. Once the NGINX container is up, replace the contents of ```certs/EB_web.crt``` with the contents of ```full_server.crt``` generated in the first step.

    ```bash
        cat full_server.crt > certs/EB_web.crt
    ```

### Modify auto.ipxe as per setup details

1. Inside ```data/auto.ipxe```, replace the placeholders with real values.

    ```bash
        set loadbalancer <LOADBALANCER>
        set macaddress <MAC_ADDRESS>
        set nginx <NGINX_IP_ADDRESS>
    ```

2. Copy the ```vmlinuz``` and ```initramfs``` files generated in tink-stack inside the ```data``` folder.

3. Copy the signed ```ipxe.efi``` generated as per the [documentation](https://github.com/intel-innersource/documentation.edge.one-edge.maestro/blob/762b2526abd36203f2ee5c20b45ccaea9ebb2140/content/docs/specs/secure-boot.md#download-and-build-ipxe-image) inside the ```data``` folder.

### Upload certificate to BIOS

1. Refer the [documentation](https://github.com/intel-innersource/documentation.edge.one-edge.maestro/blob/762b2526abd36203f2ee5c20b45ccaea9ebb2140/content/docs/specs/secure-boot.md#bios-settings-in-idrac-gui) to upload the HTTP boot URL.<br>
The URL will be of the form ```https://<NGINX_HOST_IP_ADDRESS/ipxe.efi```<br>
The certificate file will be the ```full_server.crt``` file generated earlier.

### HTTP Boot

1. Boot to Tinkerbell interface using HTTP boot option.

### Deploy onboarding and provisioning components

> Note: This setup instructions are meant for On-prem deployment

1. Deploy the Tinkerbell services using tink-stack umbrella helm charts. If RKE2 cluster is not setup then below setup script will bring up RKE2 cluster and deploy the Tinkerbell components.

   ```bash
   cd provisioning
   chmod +x ./setup_tinkerbell_stack_with_intel_network.sh
   ./setup_tinkerbell_stack_with_intel_network.sh
   ```

2. Build custom tinker actions docker images using script
Update config file which holds all the configuration details needed for the setup. Change parameters in config file `pub_inerface_name`, `pd_host_ip` and `load_balancer_ip` and proxy settings.

    ```bash
    cd deployments/scripts/onboarding
    vim config
    ```

    ```bash
    cd setup_scripts
    chmod + ./setup_actions.sh
    ./setup_actions.sh
    ```

3. Deploy the FDO services and provisioning service using helm chart

    ```bash
    cd deployments/scripts/onboarding/setup_scripts
    chmod + ./helm_setup_script.sh
    ./helm_setup_script.sh
    ```

## How to test

>Note: Install earthly

1. Build provisioning CLI tool using Earthfile

   ```bash
   earthly +build-onboardingcli
   ```

2. Update `internal/provisioningservice/test/client/profile_sample.yaml` with OS profile and node details. Update `macid` with edge node mac ID, `sutip` with IP address of the node, `pdip` with provisioning service IP and `loadbalancerip`.

3. Set ENV variables `MGR_HOST` to IP of provisioning system and `ONBMGR_PORT` to 32000 which is node port of onboarding manager service.

4. Run onboardingcli

    ```bash
    cd internal/provisioningservice/test/client/
    ../../../../build/onboardingcli
    ```

5. On edge node complete the configuring the secure boot and HTTPS boot using KVM console/ Dell idRAC. Refer to step

6. Choose boot option to boot from UEFI HTTP

7. After this on the node side operations will happen without intervention. Monitor the logs of onboarding manager service and tinker boots log.
