properties([buildDiscarder(logRotator(numToKeepStr: '20'))])

pipeline {
    agent {
        label 'linux && x86_64'
    }

    options {
        skipDefaultCheckout(true)
    }

    stages {
        stage('Build') {
            parallel {
                stage("Validate") {
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            sh 'make -f docker.Makefile lint'
                            sh 'make -f docker.Makefile vendor'
                        }
                    }
                }
                stage("Binaries"){
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    steps  {
                        dir('src/github.com/docker/app') {
                            script {
                                try {
                                    checkout scm
                                    sh 'make -f docker.Makefile cross e2e-cross tars'
                                    sh 'BIN_NAME=yamlschema make -f docker.Makefile cross'
                                    dir('bin') {
                                        stash name: 'binaries'
                                    }
                                    dir('e2e') {
                                        stash name: 'e2e'
                                    }
                                    dir('vendor') {
                                        stash name: 'vendor'
                                    }
                                    dir('specification') {
                                        stash name: 'specification'
                                    }
                                    dir('examples') {
                                        stash name: 'examples'
                                    }
                                    if(!(env.BRANCH_NAME ==~ "PR-\\d+")) {
                                        stash name: 'artifacts', includes: 'bin/*.tar.gz', excludes: 'bin/*-e2e-*'
                                        archiveArtifacts 'bin/*.tar.gz'
                                    }
                                } finally {
                                    def clean_images = /docker image ls --format "{{.ID}}\t{{.Tag}}" | grep $(git describe --always --dirty) | awk '{print $1}' | xargs docker image rm -f/
                                    sh clean_images
                                }
                            }
                        }
                    }
                    post {
                        always {
                            deleteDir()
                        }
                    }
                }
            }
        }
        stage('Test') {
            parallel {
                stage("Coverage") {
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            sh 'make -f docker.Makefile coverage'
                            archiveArtifacts '_build/ci-cov/all.out'
                            archiveArtifacts '_build/ci-cov/coverage.html'
                        }
                    }
                }
                stage("Coverage (experimental)") {
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            sh 'make EXPERIMENTAL=on -f docker.Makefile coverage'
                        }
                    }
                }
                stage("Gradle test") {
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            dir("bin") {
                                unstash "binaries"
                            }
                            sh 'make -f docker.Makefile gradle-test'
                        }
                    }
                    post {
                        always {
                            deleteDir()
                        }
                    }
                }
                stage("Test Linux") {
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    steps  {
                        dir('src/github.com/docker/app') {
                            unstash "binaries"
                            dir('vendor') {
                                unstash "vendor"
                            }
                            dir('specification') {
                                unstash "specification"
                            }
                            dir('examples') {
                                unstash "examples"
                            }
                            dir('e2e'){
                                unstash "e2e"
                            }
                            sh './docker-app-e2e-linux --e2e-path=e2e'
                        }
                    }
                    post {
                        always {
                            deleteDir()
                        }
                    }
                }
            }
        }
    }
}
