pipeline {
    agent any
    parameters {
        choice(name: 'OS', choices: ['linux', 'apple', 'windows'], description: 'Pick OS')
        choice(name: 'ARCH', choices: ['amd64', 'arm64'], description: 'Pick ARCH')
    }

    environment {
        GITHUB_TOKEN=credentials('mrgitmail')
        REPO = 'https://github.com/mrgitmail/kbot.git'
        GHCR_PAT = credentials('github-pat')
        BRANCH = 'main'
    }

    stages {

        stage('clone') {
            steps {
                echo 'Clone Repository'
                git branch: "${BRANCH}", url: "${REPO}"
            }
        }

        stage('test') {
            steps {
                echo 'Testing started'
                sh "make test"
            }
        }

        stage('build') {
            steps {
                echo "Building binary for platform ${params.OS} on ${params.ARCH} started"
                sh "make ${params.OS} ${params.ARCH}"
            }
        }

        stage('image') {
            steps {
                echo "Building image for platform ${params.OS} on ${params.ARCH} started"
                sh "make image-${params.OS} ${params.ARCH}"
            }
        }
        
        stage('login to GHCR') {
            steps {
                withCredentials([string(credentialsId: 'github-pat', variable: 'GHCR_PAT')]) {
                    sh "echo ${GHCR_PAT} | docker login ghcr.io -u ${GHCR_PAT} --password-stdin"
                }
            }
        }

        stage('push image') {
            steps {
                sh "make -n ${params.OS} ${params.ARCH} image push"
            }
        } 
    }
    post {
        always {
            sh 'docker logout'
        }
    }
}