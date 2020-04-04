pipeline {
    environment {
        registry = "rishabhbector/benevolentbites"
        registryCredential = ‘dockerhub’
    }

    agent { dockerfile true }

    stages {
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