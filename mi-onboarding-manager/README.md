Your new repo has been pre-propulated with this Readme and a minimal Jenkinsfile. The steps for the Jenkinsfile should be adapted to suit the build/test commands of this repo contents.

## Scans
Scans have been limited to the minimal required number. They can be extended with Bandit(for python code) or Snyk(for go code). Protex will only run on "main" branch and not on PRs because of tool limitations (parelel jobs cannot be executed) and also to shorten the PR check time. 

## Triggers
Please adapt the time at which the main branch is being executed according to your night time

## Containers
amr-registry.caas.intel.com/one-intel-edge/rrp-devops/oie_ci_testing:latest is used currently. This container has the following tools: 
```
Git 
Make and standard build tooling 
Docker CLI (to start containers) 
Go 1.19.x 
Python 3.9 or later 
NodeJS 18 
Mermaid CLI (used in documentation generation): https://github.com/mermaid-js/mermaid-cli 
```

## Coverage above 70% threshold. The following tools 
```
Python - Coverage
Java - Bazel, Jacoco
Go - Built-in Coverage
JS - c8
```

## Linters. The following tools 
```
Python - Flake8 (formerly pep8)
Java - Sonallint 
Go - GoLint
JS - prittier, Karma
Ansible - Ansible Lint
```

## Artifacts
The source will be packed in a archive named after the repo. The archive will then be uploaded to artifactory following a path simillar to:
https://ubit-artifactory-or.intel.com/artifactory/one-intel-edge-or-local/<project_name>/<jenkins_controller>/<jenkins_team>/<jenkins_job>/<repo_name>/<branch>/

## Create Custom HTTPS supported NGINX server

1. Refer to this section [Server Certificates for HTTPS boot](https://github.com/intel-innersource/documentation.edge.one-edge.maestro/blob/762b2526abd36203f2ee5c20b45ccaea9ebb2140/content/docs/specs/secure-boot.md#server-certificates-for-htts-boot) for creating certificates. The file ```full_server.crt``` will be required in the next steps.
1. Clone the [repository](https://github.com/intel-sandbox/nginx/tree/main)
1. Go inside the repository and build and run the nginx container as per the [README](https://github.com/intel-sandbox/nginx/blob/main/README.md). 
As seen in the docker run command example we are mounting two folders to the container, which are refered to in the next steps.
    ```
        -v ./certs:/etc/ssl/cert/ \
        -v ./data:/usr/share/nginx/html \
    ```
    ```certs``` : server certificates are present here
    ```data```  : files present here are hosted by the NGINX server
    <br>
    
1. Once the NGINX container is up, replace the contents of ```certs/EB_web.crt``` with the contents of ```full_server.crt``` generated in the first step.
```
    $ cat full_server.crt > certs/EB_web.crt
```

## Modify auto.ipxe as per setup details
1. Inside ```data/auto.ipxe```, replace the placeholders with real values.
```
    set loadbalancer <LOADBALANCER>
    set macaddress <MAC_ADDRESS>
    set nginx <NGINX_IP_ADDRESS>
```
2. Copy the ```vmlinuz``` and ```initramfs``` files generated in tink-stack inside the ```data``` folder.
3. Copy the signed ```ipxe.efi``` generated as per the [documentation](https://github.com/intel-innersource/documentation.edge.one-edge.maestro/blob/762b2526abd36203f2ee5c20b45ccaea9ebb2140/content/docs/specs/secure-boot.md#download-and-build-ipxe-image) inside the ```data``` folder.

## Upload certificate to BIOS

1. Refer the [documentation](https://github.com/intel-innersource/documentation.edge.one-edge.maestro/blob/762b2526abd36203f2ee5c20b45ccaea9ebb2140/content/docs/specs/secure-boot.md#bios-settings-in-idrac-gui) to upload the HTTP boot URL.<br>
The URL will be of the form ```https://<NGINX_HOST_IP_ADDRESS/ipxe.efi```<br>
The certificate file will be the ```full_server.crt``` file generated earlier.

## HTTP Boot

1. Boot to Tinkerbell interface using HTTP boot option.
