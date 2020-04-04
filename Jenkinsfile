pipeline {
    environment {
        registry = "rishabhbector/benevolentbites"
        registryCredential = "dockerhub"
        dockerImage = ''
    }

    agent any

    stages {
        stage('Clone Repo') {
            steps {
                git 'https://github.com/rishabh-bector/BenevolentBitesBack.git'
            }
        }
        stage('Build Image') {
            steps {
                echo "STAGE: Deploying..."
                script {
                    dockerImage = docker.build registry + ":$BUILD_NUMBER"
                }
            }
        }
        stage('Push Image') {
            steps {
                script {
                    docker.withRegistry('', registryCredential ) {
                        dockerImage.push()
                    }
                }
            }
        }
        // Only create automatic release for DEV
        stage('Deploy DEV') {
            when {
                branch 'master'
            }
            steps {
                withCredentials([string(credentialsId: 'OctopusAPIKey', variable: 'APIKey')]) {	                
                    sh 'sudo octo create-release --project "Benevolent Bites" --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}'
                    sh 'sudo octo deploy-release --project "Benevolent Bites" --version latest --deployto DEV --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}'         
                }
            }
        }
        stage('Remove Unused Image') {
            steps{
                sh "docker rmi $registry:$BUILD_NUMBER"
            }
        }
    }
    options { buildDiscarder(logRotator(numToKeepStr: '5')) }
}