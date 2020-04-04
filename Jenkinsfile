pipeline {
    agent { docker { image 'golang:buster' } }
    environment {
        CGO_ENABLED = '0'
        GOOS = 'linux'
        GOARCH = 'amd64'
    }
    stages {
        stage('Build') {
            steps {
                echo "STAGE: Building..."
                sh 'pwd && ls'
                sh 'go mod download'
            }
        }
        stage('Test') {
            steps {
                // Add actual testing eventually
                echo "STAGE: Testing..."
            }
        }
        stage('Deploy') {
            when {
                branch 'master' 
            }
            steps {
                echo "STAGE: Deploying..."
                withCredentials([string(credentialsId: 'OctopusAPIKey', variable: 'APIKey')]) {
                    sh """
                        octo push --package benevolent-back.1.0.0.zip --replace-existing --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}
                        octo create-release --project "Benevolent Bites" --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}
                        octo deploy-release --project "Benevolent Bites" --version latest --deployto Integration --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}
                    """
                }
            }
        }
    }
    options { buildDiscarder(logRotator(numToKeepStr: '5')) }

}