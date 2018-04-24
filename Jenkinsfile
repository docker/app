node('gcp-linux-worker-0') {
    stage('Build') {
        dir('src/github.com/docker/lunchbox') {
            try {
                checkout scm
                sh 'rm -f *.tar.gz'
                sh 'docker image prune -f'
                sh 'make ci-lint'
                sh 'make ci-test'
                sh 'make ci-bin-linux'
                sh 'make ci-bin-darwin'
                sh 'make ci-bin-windows'
                archiveArtifacts '*.tar.gz'
            } finally {
                def clean_images = /docker image ls --format "{{.ID}}\t{{.Tag}}" | grep $(git describe --always --dirty) | awk '{print $1}' | xargs docker image rm/
                sh clean_images
            }
        }
    }
}
