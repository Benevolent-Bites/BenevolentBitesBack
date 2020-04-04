pipeline {
    environment {
        registry = "rishabhbector/benevolentbites"
        registryCredential = "dockerhub"
    }

    agent any

    stages {
        stage('Cloning Repo') {
            steps {
                git 'https://github.com/rishabh-bector/BenevolentBitesBack.git'
            }
        }
        stage('Deploy') {
            when {
                branch 'master' 
            }
            steps {
                echo "STAGE: Deploying..."
                script {
                    docker.build registry + ":$BUILD_NUMBER"
                }
            }
        }
    }
    options { buildDiscarder(logRotator(numToKeepStr: '5')) }
}