// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

def getEnvFromBranch(branch) {
    if (branch ==~ /main/) {
        return 'protex'
    }
    else {
        return 'virus'
        // PR checks can be extended with checkmarx, bandit, snyk but source code has to available for them to pass. Protex should be kept only at branch level scanning.
    }
}

pipeline {
    agent {
        docker {
            label 'docker'
            image 'amr-registry.caas.intel.com/one-intel-edge/rrp-devops/oie_ci_testing:latest'
            alwaysPull true
        }
    }
    environment {
        GIT_SHORT_URL=env.GIT_URL.split('/')[4].toString().replaceAll('.git','')
        PROJECT_NAME = "${GIT_SHORT_URL}"
        authorEmail = sh (script: 'git --no-pager show -s --format=\'%ae\'',returnStdout: true).trim()
        SDLE_UPLOAD_PROJECT_ID = ' ' //add your SDL project
    }
    stages {
        stage('Scan Sources'){
            environment {
                SCANNERS            = getEnvFromBranch(env.BRANCH_NAME)
                PROTEX_PROJECT_NAME = "${GIT_SHORT_URL}"
            }
            when {
                anyOf {
                    branch 'main';
                    changeRequest();
                }
            }
            steps {
                rbheStaticCodeScan()
            }
        }
        // This stage is required for service/agent repos only
        // Please remove it for chart repos
        stage('Version Check') {
            steps {
                echo "Check if its a valid code version"
                sh '''
                /opt/ci/version-check.sh
                '''
            }        
        }
        stage('Build') {
            steps {
                echo "Hi, I'm a pipeline, doing build step"
                sh '''
                make build
                '''
            }
        }
        stage('License Check') {
            steps {
                sh '''
                echo "License checking the code"
                make license
                '''
            }
        }
        stage('Lint') {
            steps {
                echo "Hi, I'm a pipeline, doing lint step"
                sh '''
                make lint
                '''
            }
        }
        stage('Test') {
            when {
                changeRequest()
            }
            steps {
                echo "Hi, I'm a pipeline, doing test step"
                sh '''
                make test
                make coverage
                '''
                 /*post {
                    success {
                        coverageReport('cobertura-coverage.xml')
                    }
                }*/
            }
        }
        // This stage is required for service/agent repos only
        // Please remove it for chart repos
        stage('Version Tag') {
            when {
                anyOf { branch 'main'; branch 'feature*'; branch 'release*' }
            }
            steps {
                withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'sys_oie_devops_github_api',usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD']])
                    {
                        netrcPatch()
                        echo "Generate tag if SemVer"
                        sh '''
                        # Tag the version
                        /opt/ci/version-tag.sh
                        '''
                }
            }
        }
        stage('Version dev') {
            when {
                anyOf { branch 'main'; branch 'iaas-*-*'; branch 'release-*'; }
            }
            steps {
                withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'sys_oie_devops_github_api',usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD']])
                {
                    versionDev()
                }
            }
        }
        stage('Auto approve') {
            when {
                changeRequest()
            }
            steps {
                withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'sys_devops_approve_github_api', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD']]) {
                    script {
                        autoApproveAndMergePR()
                    }
                }
            }
        }
        stage('Artifact') {
            steps {
                artifactUpload()
            }
        }
    }
    post {
        always {
            jcpSummaryReport()
            intelLogstashSend failBuild: false, verbose: true
            cleanWs()
        }
        failure {
            script {
                emailFailure()
            }
        }
    }
}
