properties([buildDiscarder(logRotator(numToKeepStr: '20'))])

pipeline {
    agent {
        label 'pipeline'
    }

    options {
        skipDefaultCheckout(true)
    }

    stages {
        stage('Build') {
            agent {
                label 'linux'
            }
            steps  {
                dir('src/github.com/docker/app') {
                    script {
                        try {
                            checkout scm
                            sh 'docker image prune -f'
                            sh 'make -f docker.Makefile lint'
                            sh 'make -f docker.Makefile cross e2e-cross tars'
                            dir('bin') {
                                stash name: 'binaries'
                            }
                            dir('e2e') {
                                stash name: 'e2e'
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
        stage('Test') {
            parallel {
                stage("Coverage") {
                    environment {
                        CODECOV_TOKEN = credentials('jenkins-codecov-token')
                    }
                    agent {
                        label 'linux'
                    }
                    steps {
                        dir('src/github.com/docker/app') {
                            checkout scm
                            sh 'make -f docker.Makefile coverage'
                            archiveArtifacts '_build/ci-cov/all.out'
                            archiveArtifacts '_build/ci-cov/coverage.html'
                            sh 'curl -s https://codecov.io/bash | bash -s - -f _build/ci-cov/all.out -K'
                        }
                    }
                }
                stage("Gradle test") {
                    agent {
                        label 'linux'
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
                        label 'linux'
                    }
                    steps  {
                        dir('src/github.com/docker/app') {
                            unstash 'binaries'
                            unstash 'e2e'
                            sh './docker-app-e2e-linux'
                        }
                    }
                    post {
                        always {
                            deleteDir()
                        }
                    }
                }
                stage("Test Mac") {
                    agent {
                        label "mac"
                    }
                    steps {
                        dir('src/github.com/docker/app') {
                            unstash 'binaries'
                            unstash 'e2e'
                            sh './docker-app-e2e-darwin'
                        }
                    }
                    post {
                        always {
                            deleteDir()
                        }
                    }
                }
                stage("Test Win") {
                    agent {
                        label "windows"
                    }
                    steps {
                        dir('src/github.com/docker/app') {
                            unstash "binaries"
                            unstash 'e2e'
                            bat 'docker-app-e2e-windows.exe'
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
        stage('Release') {
            agent {
                label "linux"
            }
            when {
                buildingTag()
            }
            steps {
                dir('src/github.com/docker/app') {
                    unstash 'artifacts'
                    echo "Releasing $TAG_NAME"
                    dir('bin') {
                        release('docker/app')
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

def release(repo) {
    withCredentials([[$class: 'StringBinding', credentialsId: 'github-release-token', variable: 'GITHUB_TOKEN']]) {
        def data = "{\"tag_name\": \"$TAG_NAME\", \"name\": \"$TAG_NAME\", \"draft\": true, \"prerelease\": true}"
        def url = "https://api.github.com/repos/$repo/releases"
        def reply = sh(returnStdout: true, script: "curl -sSf -H \"Authorization: token $GITHUB_TOKEN\" -H \"Accept: application/json\" -H \"Content-type: application/json\" -X POST -d '$data' $url")
        def release = readJSON text: reply
        url = release.upload_url.replace('{?name,label}', '')
        sh("for f in * ; do curl -sf -H \"Authorization: token $GITHUB_TOKEN\" -H \"Accept: application/json\" -H \"Content-type: application/octet-stream\" -X POST --data-binary \"@\${f}\" $url?name=\${f}; done")
    }
}
