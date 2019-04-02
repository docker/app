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
                                sh 'make -f docker.Makefile INVOCATION_IMAGE_TAG=$BUILD_TAG-coverage OUTPUT=coverage-invocation-image.tar save-invocation-image-tag'
                                sh 'make -f docker.Makefile INVOCATION_IMAGE_TAG=$BUILD_TAG-coverage-experimental OUTPUT=coverage-experimental-invocation-image.tar save-invocation-image-tag'
                            }
                            dir('_build') {
                                stash name: 'invocation-image', includes: 'invocation-image.tar'
                                stash name: 'coverage-invocation-image', includes: 'coverage-invocation-image.tar'
                                stash name: 'coverage-experimental-invocation-image', includes: 'coverage-experimental-invocation-image.tar'
                            }
                        }
                    }
                    post {
                        always {
                            dir('src/github.com/docker/app') {
                                sh 'docker rmi docker/cnab-app-base:$BUILD_TAG'
                                sh 'docker rmi docker/cnab-app-base:$BUILD_TAG-coverage'
                                sh 'docker rmi docker/cnab-app-base:$BUILD_TAG-coverage-experimental'
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
                    agent {
                        label 'ubuntu-1604-aufs-edge'
                    }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            dir('_build') {
                                unstash "coverage-invocation-image"
                                sh 'docker load -i coverage-invocation-image.tar'
                            }
                            ansiColor('xterm') {
                                sh 'make -f docker.Makefile BUILD_TAG=$BUILD_TAG-coverage coverage'
                            }
                            archiveArtifacts '_build/ci-cov/all.out'
                            archiveArtifacts '_build/ci-cov/coverage.html'
                        }
                    }
                    post {
                        always {
                            sh 'sed -i -E -e \'s,"github.com/docker/app","unit/basic",g; s,"github.com/docker/app/([^"]*)","unit/basic/\\1",g\' src/github.com/docker/app/_build/test-results/unit-coverage.xml'
                            sh 'sed -i -E -e \'s,"github.com/docker/app/e2e","e2e/basic",g\' src/github.com/docker/app/_build/test-results/e2e-coverage.xml'
                            archiveArtifacts 'src/github.com/docker/app/_build/test-results/*.xml'
                            junit 'src/github.com/docker/app/_build/test-results/*.xml'
                            sh 'docker rmi docker/cnab-app-base:$BUILD_TAG-coverage'
                            deleteDir()
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
                            dir('_build') {
                                unstash "coverage-experimental-invocation-image"
                                sh 'docker load -i coverage-experimental-invocation-image.tar'
                            }
                            ansiColor('xterm') {
                                sh 'make EXPERIMENTAL=on TEST_RESULTS_PREFIX="experimental-" -f docker.Makefile BUILD_TAG=$BUILD_TAG-coverage-experimental coverage'
                            }
                        }
                    }
                    post {
                        always {
                            sh 'sed -i -E -e \'s,"github.com/docker/app","unit/experimental",g; s,"github.com/docker/app/([^"]*)","unit/experimental/\\1",g\' src/github.com/docker/app/_build/test-results/experimental-unit-coverage.xml'
                            sh 'sed -i -E -e \'s,"github.com/docker/app/e2e","e2e/experimental",g\' src/github.com/docker/app/_build/test-results/experimental-e2e-coverage.xml'
                            archiveArtifacts 'src/github.com/docker/app/_build/test-results/*.xml'
                            junit 'src/github.com/docker/app/_build/test-results/*.xml'
                            sh 'docker rmi docker/cnab-app-base:$BUILD_TAG-coverage-experimental'
                            deleteDir()
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
                            ansiColor('xterm') {
                                sh 'make -f docker.Makefile gradle-test'
                            }
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
                            sh 'docker rmi docker/cnab-app-base:$BUILD_TAG'
                            deleteDir()
                        }
                    }
                }
            }
        }
    }
}
