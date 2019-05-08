properties([buildDiscarder(logRotator(numToKeepStr: '20'))])

pipeline {
    agent {
        label 'linux && x86_64'
    }

    options {
        skipDefaultCheckout(true)
    }

    environment {
        TAG = "${env.BUILD_TAG}"
    }

    stages {
        stage('Build') {
            parallel {
                stage("Binaries"){
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    steps  {
                        dir('src/github.com/docker/app') {
                            script {
                                try {
                                    checkout scm
                                    ansiColor('xterm') {
                                        sh 'make -f docker.Makefile cli-cross cross e2e-cross tars'
                                    }
                                    dir('bin') {
                                        stash name: 'binaries'
                                    }
                                    dir('e2e') {
                                        stash name: 'e2e'
                                    }
                                    dir('examples') {
                                        stash name: 'examples'
                                    }
                                    if(!(env.BRANCH_NAME ==~ "PR-\\d+")) {
                                        stash name: 'artifacts', includes: 'bin/*.tar.gz', excludes: 'bin/*-e2e-*'
                                        archiveArtifacts 'bin/*.tar.gz'
                                    }
                                    archiveArtifacts 'bin/*-linux'
                                } finally {
                                    def clean_images = /docker image ls --format="{{.Repository}}:{{.Tag}}" '*$BUILD_TAG*' | xargs docker image rm -f/
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
                stage('Build Invocation image'){
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            ansiColor('xterm') {
                                sh 'make -f docker.Makefile save-invocation-image'
                                sh 'make -f docker.Makefile INVOCATION_IMAGE_TAG=$TAG-coverage OUTPUT=coverage-invocation-image.tar save-invocation-image-tag'
                                sh 'make -f docker.Makefile INVOCATION_IMAGE_TAG=$TAG-coverage-experimental OUTPUT=coverage-experimental-invocation-image.tar save-invocation-image-tag'
                            }
                            dir('_build') {
                                stash name: 'invocation-image', includes: 'invocation-image.tar'
                                stash name: 'coverage-invocation-image', includes: 'coverage-invocation-image.tar'
                                stash name: 'coverage-experimental-invocation-image', includes: 'coverage-experimental-invocation-image.tar'
                                archiveArtifacts 'invocation-image.tar'
                            }
                        }
                    }
                    post {
                        always {
                            dir('src/github.com/docker/app') {
                                sh 'docker rmi docker/cnab-app-base:$TAG'
                                sh 'docker rmi docker/cnab-app-base:$TAG-coverage'
                                sh 'docker rmi docker/cnab-app-base:$TAG-coverage-experimental'
                            }
                            deleteDir()
                        }
                    }
                }
            }
        }
        stage('Test') {
            parallel {
                stage("Test Linux") {
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    environment {
                        DOCKERAPP_BINARY = '../docker-app-linux'
                        DOCKERCLI_BINARY = '../docker-linux'
                    }
                    steps  {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            dir('_build') {
                                unstash "invocation-image"
                                sh 'docker load -i invocation-image.tar'
                            }
                            unstash "binaries"
                            dir('examples') {
                                unstash "examples"
                            }
                            dir('e2e'){
                                unstash "e2e"
                            }
                            ansiColor('xterm') {
                                sh './gotestsum-linux --format standard-verbose --junitfile e2e-linux.xml --raw-command -- ./test2json-linux -t -p e2e/linux ./docker-app-e2e-linux -test.v --e2e-path=e2e'
                            }
                        }
                    }
                    post {
                        always {
                            archiveArtifacts 'src/github.com/docker/app/e2e-linux.xml'
                            junit 'src/github.com/docker/app/e2e-linux.xml'
                            sh 'docker rmi docker/cnab-app-base:$TAG'
                            deleteDir()
                        }
                    }
                }
            }
        }
    }
}
