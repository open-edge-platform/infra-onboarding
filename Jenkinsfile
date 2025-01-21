// SPDX-FileCopyrightText: (C) 2024 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel
// Imported groovy libraries:
// 1cicd: ["intel-innersource/applications.devops.jenkins.jenkins-common-pipelines"]
// maestro-i-cicd: ["intel-innersource/frameworks.edge.one-intel-edge.maestro-infra.jenkins-common-pipelines"]

def branchPattern = /(main|release-*)/

def getEnvFromBranchSource(branch, pattern) {
    if (branch ==~ pattern) {
        return 'checkmarx,protex'
    }
    else {
        return 'virus,trivy,hadolint'
    }
}

def getDockerTags(prefix) {
    // Extract git tags
    git_tags = sh(script: 'git tag -l --points-at HEAD', returnStdout: true).trim().split('\n') as List
    // Sanitize in case of no tags
    git_tags = git_tags.findAll { it != '' && it != null }
    git_tags = git_tags.findAll { it.contains(prefix) }
    git_tags = git_tags.collect { it.replaceAll(prefix, '') }
    git_tags = git_tags.collect { it.replaceAll('v', '') }
    // Build the target branch
    git_tags << env.GIT_BRANCH
    return git_tags
}

// Perform only build -> docker.build()
def dockerBuild(prefix) {
    getDockerTags(prefix).each { sh """IMG_VERSION="$it" make docker-build""" }
}

// Perform only push -> docker.push()
def dockerPush(prefix) {
    dockerCommon.login('amr-registry.caas.intel.com', 'sys_oie_devops_amr_harbour')
    getDockerTags(prefix).each { sh """IMG_VERSION="$it" make docker-push""" }
}

def dockerDevPush(prefix) {
    dockerCommon.login('amr-registry.caas.intel.com', 'sys_oie_devops_amr_harbour')
    getDockerTags(prefix).each { sh """IMG_VERSION="$it" make docker-dev-push""" }
}

def envVarsMap = [:]

pipeline {
     triggers {
        // nightly build between 07:00 a.m. - 23:59 a.m.(Etc/UTC), Monday - Friday:
        cron(env.BRANCH_NAME =~ /(main|release-*)/ ? 'H 07 * * 1-5' : '')
    }
    agent {
        label 'oie_spot_executor'
    }
    environment {
        DEBIAN_FRONTEND="noninteractive"
        authorEmail = sh (script: 'git --no-pager show -s --format=\'%ae\'',returnStdout: true).trim()
    }
    stages {
        stage('Verify branch name') {
            when {
                changeRequest()
            }
            steps {
                script {
                    def currentBranch = env.CHANGE_BRANCH
                    if (currentBranch ==~ branchPattern) {
                        error "Not allowed branch name!"
                    }
                }
            }
        }
        stage('Discover Changed Subfolders') {
            steps {
                script {
                    // Find all subfolders
                    def projects = sh(script: "ls -d */", returnStdout: true).trim().split("\n")
                    def changedProjects = []
                    projects.each { project ->
                        def isChanged = ''
                        if (env.BRANCH_NAME ==~ branchPattern) {
                            // Diff in case of post-merge is done with respect to last-1 commit in the current branch
                            // This assumes that PRs are merged with squash and merge.
                            isChanged = sh(
                                script: "git diff --quiet origin/${env.BRANCH_NAME}~1 HEAD -- ${project} || echo 'CHANGED'",
                                returnStdout: true
                            ).trim()
                        } else {
                            // Logic for PR build
                            isChanged = sh(
                                script: "git diff --quiet origin/${env.CHANGE_TARGET} HEAD -- ${project} || echo 'CHANGED'",
                                returnStdout: true
                            ).trim()
                        }
                        if (isChanged == 'CHANGED') {
                            changedProjects << project.replaceAll('/', '')
                        }
                    }
                    // Store the changed projects as a comma-separated list
                    env.CHANGED_PROJECTS = changedProjects.join(",")
                    echo "Changed projects: ${env.CHANGED_PROJECTS}"
                }
            }
        }
        stage('Verify branch name for release') {
            when {
                changeRequest()
            }
            steps {
                script {
                    def currentBranch = env.CHANGE_TARGET
                    // We want to ensure that the changes for release branches are only for the target folder for that release branch.
                    if (currentBranch ==~ /release-.*/ && (env.CHANGED_PROJECTS.split(",").size() > 1 || !(currentBranch ==~ /release-${env.CHANGED_PROJECTS.split(',')[0].replaceAll('\\/$', '')}-\d+\.\d+/))) {
                        error "Not allowed branch PR, you can change only the release branch target folder!"
                    }
                }
            }
        }
        stage('Run Sub-Pipelines') {
            matrix{
                axes {
                    axis {
                        name 'PROJECT_FOLDER'
                        values "onboarding-manager", "dkam", "tinker-actions"
                    }
                }
                stages {
                    stage('Sub-Pipeline') {
                        when {
                            // Run sub-pipeline only if the target project is changed, run beforeAgent to avoid spinning up the agent.
                            beforeAgent true
                            expression {
                                env.CHANGED_PROJECTS.split(',').contains("${PROJECT_FOLDER}")
                            }
                        }
                        agent {
                            docker {
                              label 'oie_spot_executor'
                              image 'amr-registry.caas.intel.com/one-intel-edge/rrp-devops/oie_ci_testing:2.13.9'
                              alwaysPull true
                            }
                        }
                        environment {
                            // Some scripts require the PROJECT_FOLDER to be set as BASEDIR.
                            BASEDIR="${PROJECT_FOLDER}"
                        }
                        stages {
                            stage('Load Environment Variables') {
                                steps {
                                    script {
                                        def envFile = "${PROJECT_FOLDER}/.env.jenkins"
                                        echo "Loading environment variables from ${envFile}"
                                        if (fileExists(envFile)) {
                                            def envVars = readFile(envFile).split('\n').collect { it.trim() }.findAll { it && !it.startsWith('#') }
                                            def envVarsMapForProject = envVars.collectEntries { line ->
                                                def (key, value) = line.split('=')
                                                [(key): value]
                                            }
                                            envVarsMap[PROJECT_FOLDER] = envVarsMapForProject
                                        } else {
                                            echo "No .env.jenkins file found in ${PROJECT_FOLDER}"
                                        }
                                    }
                                }
                            }
                            stage('Print Loaded Environment Variables') {
                                steps {
                                    script {
                                        echo "Loaded environment variables for ${PROJECT_FOLDER}:"
                                        envVarsMap[PROJECT_FOLDER].each { key, value ->
                                            echo "${key}=${value}"
                                        }
                                    }
                                }
                            }
                            stage('Setup') {
                                steps {
                                    withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'sys_oie_devops_github_api',usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD']])
                                    {
                                        netrcPatch()
                                    }
                                }
                            }
                            stage('Print env') {
                                steps {
                                    script {
                                        def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                        withEnv(envVars) {
                                            sh 'printenv'
                                        }
                                    }
                                }
                            }
                            stage('Scan Virus, Checkmarx, Protex, Trivy') {
                                environment {
                                    SCANNERS = getEnvFromBranchSource(env.BRANCH_NAME, branchPattern)
                                    PROJECT_SRC_DIR = "${PROJECT_FOLDER}"
                                    VIRUS_SCAN_DIR = "${PROJECT_FOLDER}"
                                    VIRUS_SCAN_REPORT_NAME = "McAfee Virus Scan Report ${PROJECT_FOLDER}"
                                }
                                when {
                                    anyOf { branch 'main'; branch 'release-*'; changeRequest(); }
                                }
                                steps {
                                    script {
                                        dir("${PROJECT_FOLDER}") {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE') {
                                                    rbheStaticCodeScan()
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Version Check') {
                                steps {
                                    dir("${PROJECT_FOLDER}") {
                                        script {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                echo "Check if its a valid code version"
                                                sh '''
                                                /opt/ci/version-check.sh "${PROJECT_FOLDER}-"
                                                '''
                                            }
                                        }
                                    }
                                }
                            }
                            stage('[DKAM] install system dependencies') {
                                when {
                                    expression { PROJECT_FOLDER == 'dkam' }
                                }
                                steps {
                                    script {
                                        sh '''
                                            apt-get update && apt-get install -y pigz qemu-utils
                                        '''
                                    }
                                }
                            }
                            stage('Dep Version Check') {
                                when {
                                    expression { PROJECT_FOLDER != 'tinker-actions' }
                                }
                                steps {
                                    dir("${PROJECT_FOLDER}") {
                                        script {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                sh '''
                                                echo "Verifying dependencies version"
                                                make dependency-check
                                                '''
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Build Code') {
                                when {
                                    expression { PROJECT_FOLDER != 'tinker-actions' }
                                }
                                steps {
                                    dir("${PROJECT_FOLDER}") {
                                        script {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                // TODO: add checksec
                                                sh '''
                                                echo "Building the code"
                                                make go-build
                                                '''
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Lint Code') {
                                when {
                                    changeRequest()
                                }
                                steps {
                                    dir("${PROJECT_FOLDER}") {
                                        script {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                sh '''
                                                echo "Linting the code"
                                                make lint BASE_BRANCH="${CHANGE_TARGET:-${GIT_BRANCH}}"
                                                '''
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Test Code') {
                                when {
                                    allOf {
                                        changeRequest()
                                        expression { PROJECT_FOLDER != 'tinker-actions' }
                                    }
                                }
                                steps {
                                    dir("${PROJECT_FOLDER}") {
                                        script {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                sh '''
                                                    echo "Testing the code"
                                                    make test
                                                    ls build/
                                                '''
                                                artifactUpload()
                                            }
                                        }
                                    }
                                }
                                post {
                                    success {
                                        sh '''
                                        mv "${PROJECT_FOLDER}/build/coverage.xml" "${PROJECT_FOLDER}/build/coverage-${PROJECT_FOLDER}.xml"
                                        mv "${PROJECT_FOLDER}/build/report.xml" "${PROJECT_FOLDER}/build/report-${PROJECT_FOLDER}.xml"
                                        '''
                                        // TODO: apparently reports are overriding between different subfolders
                                        coverageReport("${PROJECT_FOLDER}/build/coverage-${PROJECT_FOLDER}.xml")
                                        junit "${PROJECT_FOLDER}/build/report-${PROJECT_FOLDER}.xml"
                                    }
                                }
                            }
                            stage('[Tinker Actions] Fuzz Test') {
                                when {
                                    expression { PROJECT_FOLDER == 'tinker-actions' }
                                }
                                steps {
                                    dir("${PROJECT_FOLDER}") {
                                        sh '''
                                        make fuzztest
                                        '''
                                    }
                                }
                                post {
                                    always {
                                        script {
                                            sh '''
                                            mkdir -p fuzz_artifacts
                                            find . -name "fuzz_*.log" -exec cp {} fuzz_artifacts/ \\;

                                            for log_file in fuzz_artifacts/fuzz_*.log; do
                                                if [ -f "$log_file" ]; then
                                                    echo "---- $log_file ----"
                                                    cat "$log_file"
                                                    echo "---- End of $log_file ----"
                                                else
                                                    echo "No log files found."
                                                fi
                                            done
                                            '''

                                            archiveArtifacts artifacts: "fuzz_artifacts/**", allowEmptyArchive: true
                                        }
                                    }
                                }
                            }
                            stage('Validate clean folder') {
                                when {
                                    changeRequest()
                                }
                                steps {
                                    dir("${PROJECT_FOLDER}") {
                                        script {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                sh '''
                                                bash -c "diff -u <(echo -n) <(git diff .)"
                                                '''
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Version Tag') {
                                when {
                                    anyOf { branch 'main'; branch 'release-*'; }
                                }
                                steps {
                                    dir("${PROJECT_FOLDER}") {
                                        withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'sys_oie_devops_github_api', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD']]) {
                                            script {
                                                def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                                withEnv(envVars) {
                                                    sh '''
                                                    echo "Generate tag if SemVer"
                                                    /opt/ci/version-tag-param.sh "${PROJECT_FOLDER}-"
                                                    '''
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Build Docker image') {
                                when {
                                    anyOf { branch 'main'; branch 'release-*'; changeRequest(); }
                                }
                                steps {
                                    script {
                                        dir("${PROJECT_FOLDER}") {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                dockerBuild("${PROJECT_FOLDER}-")
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Scan Containers') {
                                when {
                                    anyOf { branch 'main'; branch 'release-*'; changeRequest(); }
                                }
                                environment {
                                    SCANNERS = 'trivy'
                                }
                                steps {
                                    script {
                                        def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                        withEnv(envVars) {
                                            scanContainers()
                                        }
                                    }
                                }
                            }
                            stage('Push Docker image') {
                                when {
                                    anyOf { branch 'main'; branch 'release-*'; expression { common.isMatchingCommit(/.*\[push-docker-image\]*/) }; }
                                }
                                steps {
                                    script {
                                        dir("${PROJECT_FOLDER}") {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                dockerPush("${PROJECT_FOLDER}-")
                                            }
                                        }
                                    }
                                }
                            }
                            stage('[Tinker Actions] Push PR tagged Docker image') {
                                when {
                                    expression { PROJECT_FOLDER == 'tinker-actions' && changeRequest() && env.CHANGE_ID != null }
                                }
                                steps {
                                    script {
                                        dir("${PROJECT_FOLDER}") {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                dockerDevPush("${PROJECT_FOLDER}-")
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Scan BDBA') {
                                environment {
                                    SCANNERS = 'bdba'
                                }
                                when {
                                    anyOf { branch 'main'; branch 'release-*'; }
                                }
                                steps {
                                    script {
                                        dir("${PROJECT_FOLDER}") {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                sh """
                                                [ -d "../BDBA" ] || mkdir ../BDBA
                                                tar -zcvf ../BDBA/${env.GIT_SHORT_URL}.tar.gz build
                                                """
                                                script {
                                                    catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE') {
                                                        rbheStaticCodeScan()
                                                    }
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Scan Binary SSCB') {
                                when {
                                    changeRequest()
                                }
                                steps {
                                    script {
                                        def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                        withEnv(envVars) {
                                            scanBinarySSCB()
                                        }
                                    }
                                }
                            }
                            stage('Artifact') {
                                steps {
                                    script {
                                        dir("${PROJECT_FOLDER}") {
                                            def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                            withEnv(envVars) {
                                                artifactUpload()
                                            }
                                        }
                                    }
                                }
                            }
                            stage('Version dev') {
                                when {
                                    anyOf { branch 'main'; branch 'iaas-*-*'; branch 'release-*'; }
                                }
                                steps {
                                    dir("${PROJECT_FOLDER}") {
                                        withCredentials([[$class: 'UsernamePasswordMultiBinding', credentialsId: 'sys_oie_devops_github_api', usernameVariable: 'USERNAME', passwordVariable: 'PASSWORD']]) {
                                            script {
                                                def envVars = envVarsMap[PROJECT_FOLDER].collect { key, value -> "${key}=${value}" }
                                                withEnv(envVars) {
                                                    // Override GIT_SHORT_URL for versionDev to point to the actual repo.
                                                    env.GIT_SHORT_URL = "frameworks.edge.one-intel-edge.maestro-infra.eim-managers"
                                                    versionDev("pierventre", "daniele-moro")
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
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
