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


## Build CDN NGINX container
```
git clone https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.infrastructure.provisioning-cdn-nginx.git cdn-nginx
cd cdn-nginx
docker build --rm --build-arg http_proxy='http://proxy-dmz.intel.com:911' --build-arg https_proxy='http://proxy-dmz.intel.com:912' --build-arg HTTP_PROXY='http://proxy-dmz.intel.com:911' --build-arg HTTPS_PROXY='http://proxy-dmz.intel.com:912' --build-arg NO_PROXY='localhost,*.intel.com,intel.com,127.0.0.1' --build-arg no_proxy='localhost,*.intel.com,intel.com,127.0.0.1' -t amr-registry.caas.intel.com/one-intel-edge/maestro-i/frameworks.edge.one-intel-edge.maestro-infra.services.infrastructure.provisioning-cdn-nginx:1.0.1-dev -f cdn-nginx/Dockerfile cdn-nginx
```

# UNIT TESTING
# Ensure ports are available
By running ```sudo lsof -i -P -n | grep LISTEN```, make sure that ports ```69, 80, 8080``` are free
## Run CDN NGINX container
```
docker run --rm -d -p 8080:8080 --network host --env BOOTS_SERVICE_URL=localhost amr-registry.caas.intel.com/one-intel-edge/maestro-i/frameworks.edge.one-intel-edge.maestro-infra.services.infrastructure.provisioning-cdn-nginx:1.0.1-dev
```
## Run stub CDN Boots container
```
docker run --rm -d --pull always --network host gar-registry.caas.intel.com/star-fw/tinkerbell/cdn-boots:stub
```

## Test
Run the following command
```
curl "http://localhost:8080/chain.php?mac=02:00:00:00:00:ff&&boot_url=http://localhost:8080" --noproxy '*'
```
Expected output
```
#!ipxe

echo Tinkerbell Boots iPXE
set iface  || shell
set tinkerbell http://127.0.0.1
set syslog_host 127.0.0.1
set ipxe_cloud_config packet

params
param body Device connected to DHCP system
param type provisioning.104.01
imgfetch http://localhost/phone-home##params
imgfree

set action workflow
set state 
set arch x86_64
set bootdevmac 02:00:00:00:00:ff
set base-url http://localhost/kernels?
kernel ${base-url}/5.0.1-fake-version ip=dhcp modules=loop,squashfs,sd-mod,usb-storage alpine_repo=${base-url}/repo-${arch}/main modloop=${base-url}/modloop-${arch} tinkerbell=${tinkerbell} syslog_host=${syslog_host} packet_action=${action} packet_state=${state} osie_vendors_url= grpc_authority=127.0.0.1 packet_base_url=http://install.ewr1.packet.net/workflow instance_id=f9f56dff-098a-4c5f-a51c-19ad35de85d1 worker_id=f9f56dff-098a-4c5f-a51c-19ad35de85d1 packet_bootdev_mac=${bootdevmac} facility= intel_iommu=on iommu=pt plan= manufacturer=fake systems incorporated slug=FakeOS 1.0:c7f0d804222d91e0ad2edd00158dbbeb initrd=initramfs.cpio.gz console=tty0 console=ttyS1,115200
initrd ${base-url}/initramfs.cpio.gz
boot
```

