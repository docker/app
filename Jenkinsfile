node('gcp-linux-worker-0') {
    stage('Build') {
        dir('src/github.com/docker/lunchbox') {
            checkout scm
            sh 'rm -f *.tar.gz'
            sh 'docker image prune -f'
            sh 'make ci-lint'
            sh 'make ci-test'
            sh 'make ci-bin-linux'
            sh 'make ci-bin-darwin'
            sh 'make ci-bin-windows'
            archiveArtifacts '*.tar.gz'
        }
    }
}
