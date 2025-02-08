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

