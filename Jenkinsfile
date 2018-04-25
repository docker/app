properties([buildDiscarder(logRotator(numToKeepStr: '20'))])

pipeline {
    agent {
        label 'gcp-linux-worker-0'
    }

    options {
        checkoutToSubdirectory('src/github.com/docker/lunchbox')
    }

    stages {
        stage('Build') {
            agent {
                label 'gcp-linux-worker-0'
            }
            steps  {
                dir('src/github.com/docker/lunchbox') {
                    script {
                        try {
                            checkout scm
                            sh 'rm -f *.tar.gz'
                            sh 'docker image prune -f'
                            sh 'make ci-lint'
                            sh 'make ci-test'
                            sh 'make ci-bin-all'
                            sh 'ls *.tar.gz | xargs -i tar xf {}'
                            stash name: "binaries", includes: "docker-app-*", excludes: "*.tar.gz"
                            archiveArtifacts '*.tar.gz'
                        } finally {
                            def clean_images = /docker image ls --format "{{.ID}}\t{{.Tag}}" | grep $(git describe --always --dirty) | awk '{print $1}' | xargs docker image rm/
                            sh clean_images
                        }
                    }
                }
            }
        }
        stage('Test') {
            parallel {
                stage("Test Linux") {
                    agent {
                        label 'gcp-linux-worker-0'
                    }
                    steps  {
                        dir('src/github.com/docker/lunchbox') {
                            unstash "binaries"
                            sh './docker-app-linux version'
                        }
                    }
                }
                stage("Test Mac") {
                    agent {
                        label "macstadium13"
                    }
                    steps {
                        dir('src/github.com/docker/lunchbox') {
                            unstash "binaries"
                            sh './docker-app-darwin version'
                        }
                    }
                }
                stage("Test Win") {
                    agent {
                        label "gcp-windows-worker-2"
                    }
                    steps {
                        dir('src/github.com/docker/lunchbox') {
                            unstash "binaries"
                            bat 'docker-app-windows.exe version'
                        }
                    }
                }
            }
        }
    }
}
