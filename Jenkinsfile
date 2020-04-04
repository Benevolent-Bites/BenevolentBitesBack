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
                echo "STAGE: Cloning repo..."
                git 'https://github.com/rishabh-bector/BenevolentBitesBack.git'
            }
        }
        
        stage('Build Image') {
            steps {
                echo "STAGE: Building image..."
                script {
                    dockerImage = docker.build registry + ":$BUILD_NUMBER"
                }
            }
        }

        stage('Push Image') {
            steps {
                script {
                    echo "STAGE: Pushing image..."
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
                echo "STAGE: Deploying to DEV..."
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