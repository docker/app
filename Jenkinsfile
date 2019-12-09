    pipeline {
    agent none

    options {
        skipDefaultCheckout(true)
        buildDiscarder(logRotator(numToKeepStr: '20'))
        timeout(time: 1, unit: 'HOURS')
    }

    environment {
        TAG = "${env.BUILD_TAG}"
        GOPROXY = "direct"
        DOCKER_BUILDKIT = "1"
    }

    stages {
        stage('Build') {
            parallel {
                stage("Validate") {
                    agent { label 'ubuntu-1804 && x86_64 && overlay2' }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            ansiColor('xterm') {
                                sh 'make -f docker.Makefile lint'
                                sh 'make -f docker.Makefile check-vendor'
                            }
                        }
                    }
                    post {
                        always {
                            deleteDir()
                        }
                    }
                }
                stage("Binaries"){
                    agent { label 'ubuntu-1804 && x86_64 && overlay2' }
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
                                } finally {
                                    def clean_images = /docker image ls --format="{{.Repository}}:{{.Tag}}" '*$BUILD_TAG*' | xargs --no-run-if-empty  docker image rm -f/
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
                    agent { label 'ubuntu-1804 && x86_64 && overlay2' }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            ansiColor('xterm') {
                                sh 'make -f docker.Makefile invocation-image save-invocation-image'
                                sh 'make -f docker.Makefile INVOCATION_IMAGE_TAG=$TAG-coverage OUTPUT=coverage-invocation-image.tar save-invocation-image-tag'
                            }
                            dir('_build') {
                                stash name: 'invocation-image', includes: 'invocation-image.tar'
                                stash name: 'coverage-invocation-image', includes: 'coverage-invocation-image.tar'
                            }
                        }
                    }
                    post {
                        always {
                            dir('src/github.com/docker/app') {
                                sh 'docker rmi docker/cnab-app-base:$TAG'
                                sh 'docker rmi docker/cnab-app-base:$TAG-coverage'
                            }
                            deleteDir()
                        }
                    }
                }
            }
        }
        stage('Test') {
            parallel {
                stage("Coverage") {
                    agent { label 'ubuntu-1804 && x86_64 && overlay2' }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            dir('_build') {
                                unstash "coverage-invocation-image"
                                sh 'docker load -i coverage-invocation-image.tar'
                            }
                            ansiColor('xterm') {
                                sh 'make -f docker.Makefile TAG=$TAG-coverage coverage-run'
                                sh 'make -f docker.Makefile TAG=$TAG-coverage coverage-results'
                            }
                            archiveArtifacts '_build/ci-cov/all.out'
                            archiveArtifacts '_build/ci-cov/coverage.html'
                        }
                    }
                    post {
                        always {
                            dir('src/github.com/docker/app/_build/test-results') {
                                sh '[ ! -e unit-coverage.xml ] || sed -i -E -e \'s,"github.com/docker/app","unit/basic",g; s,"github.com/docker/app/([^"]*)","unit/basic/\\1",g\' unit-coverage.xml'
                                sh '[ ! -e e2e-coverage.xml ] || sed -i -E -e \'s,"github.com/docker/app/e2e","e2e/basic",g\' e2e-coverage.xml'
                                archiveArtifacts '*.xml'
                                junit '*.xml'
                            }
                            sh 'docker rmi docker/cnab-app-base:$TAG-coverage'
                            deleteDir()
                        }
                    }
                }
                stage("Test Linux") {
                    agent { label 'ubuntu-1804 && x86_64 && overlay2' }
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
                                sh './gotestsum-linux --format short-verbose --junitfile e2e-linux.xml --raw-command -- ./test2json-linux -t -p e2e/linux ./docker-app-e2e-linux -test.v --e2e-path=e2e'
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
