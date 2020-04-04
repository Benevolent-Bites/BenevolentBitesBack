pipeline {
    agent { dockerfile true }
    stages {
        stage('Build') {
            steps {
                echo "STAGE: Building..."
            }
        }
        stage('Test') {
            steps {
                // Add actual testing eventually
                echo "STAGE: Testing..."
            }
        }
        stage('Deploy') {
            steps {
                echo "STAGE: Deploying..."
                withCredentials([string(credentialsId: 'OctopusAPIKey', variable: 'APIKey')]) {
                    sh """
                        ${tool('Octo CLI')}/Octo push --package benevolent-back.1.0.0.zip --replace-existing --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}
                        ${tool('Octo CLI')}/Octo create-release --project "Benevolent Bites" --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}
                        ${tool('Octo CLI')}/Octo deploy-release --project "Benevolent Bites" --version latest --deployto Integration --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}
                    """
                }
            }
        }
    }
}