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
git clone https://github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.services.infrastructure.provisioning-cdn-boots.git cdn-boots
cd cdn-boots
docker build --rm --build-arg http_proxy='http://proxy-dmz.intel.com:911' --build-arg https_proxy='http://proxy-dmz.intel.com:912' --build-arg HTTP_PROXY='http://proxy-dmz.intel.com:911' --build-arg HTTPS_PROXY='http://proxy-dmz.intel.com:912' --build-arg NO_PROXY='localhost,*.intel.com,intel.com,127.0.0.1' --build-arg no_proxy='localhost,*.intel.com,intel.com,127.0.0.1' -t intel/nginx-php-debian:v1 -f cdn-nginx/Dockerfile cdn-nginx
```

## Run CDN NGINX container
```
docker run -p 80:80 --network host intel/nginx-php-debian:v1
```
## STUB to test
The  following command will create a new folder named ```temp``` in the current directory and mount inside the container
```
docker run -p 8095:8095 --network host -v $PWD/temp:/usr/local/cdn gar-registry.caas.intel.com/star-fw/tinkerbell/cdn-boots:v1

```


For testing , you may run the following command
```
curl -X POST "http://localhost/write.php?mac=98:4f:ee:18:55:7c&&uuid=50415a46-3131-3051-b030-4a4345a4150&&serial_id=FZAP111000JC&&en_ip=172.22.28.35&&boot_url=localhost"
"
```
Expected output

curl response
```
Write successful
```

check ```temp/setup.json``` in current directory (mounted in previous step) for data validation

expected file content

```
{"mac":"98:4f:ee:18:55:7c","uuid":"50415a46-3131-3051-b030-4a4345a4150","serial_id":"FZAP111000JC","ip":"172.22.28.35"}
```
