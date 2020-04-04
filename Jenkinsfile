pipeline {
    agent { dockerfile true }
    stages {
        stage('Deploy') {
            when {
                branch 'master' 
            }
            steps {
                echo "STAGE: Deploying..."
                withCredentials([string(credentialsId: 'OctopusAPIKey', variable: 'APIKey')]) {
                    echo "Loaded Credentials.."
                    sh 'octo push --package /dist/benevolent-back.1.0.0.zip --replace-existing --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}'
                    sh 'octo create-release --project "Benevolent Bites" --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}'
                    sh 'octo deploy-release --project "Benevolent Bites" --version latest --deployto Integration --server https://benevolentbites.octopus.app/ --apiKey ${APIKey}'
                }
            }
        }
    }
    options { buildDiscarder(logRotator(numToKeepStr: '5')) }
}