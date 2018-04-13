
node('macstadium13') {
    stage('macOS') {
        def ws = pwd()
        withEnv(["GOPATH=${ws}"]) {
            dir('src/github.com/docker/lunchbox') {
                env.PATH="${GOPATH}/bin:/usr/local/go/bin:$PATH"
                checkout scm
                sh 'make bin'
                archiveArtifacts '_build/bin/*'
            }
        }
    }
}
