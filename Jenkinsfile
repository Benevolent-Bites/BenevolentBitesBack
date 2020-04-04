pipeline {
    environment {
        registry = "rishabhbector/benevolentbites"
        registryCredential = "dockerhub"
        dockerImage = ''
    }

    agent any

    stages {
        stage('Cloning Repo') {
            steps {
                git 'https://github.com/rishabh-bector/BenevolentBitesBack.git'
            }
        }
        stage('Build Docker Image') {
            when {
                branch 'master' 
            }
            steps {
                echo "STAGE: Deploying..."
                script {
                    dockerImage = docker.build registry + ":$BUILD_NUMBER"
                }
            }
        }
        stage('Deploy Image') {
            steps{
                script {
                    docker.withRegistry('', registryCredential ) {
                        dockerImage.push()
                    }
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