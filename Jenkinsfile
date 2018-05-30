properties([buildDiscarder(logRotator(numToKeepStr: '20'))])

pipeline {
    agent {
        label 'linux'
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
                dir('src/github.com/docker/lunchbox') {
                    script {
                        try {
                            checkout scm
                            sh 'docker image prune -f'
                            sh 'make ci-lint'
                            sh 'make ci-test'
                            sh 'make ci-bin-all'
                            sh 'mkdir stash'
                            sh 'ls *.tar.gz | xargs -i tar -xf {} -C stash'
                            sh 'make ci-gradle-test DOCKERAPP_BINARY=$PWD/stash/docker-app-linux'
                            dir('stash') {
                                stash name: 'e2e'
                            }
                            if(!(env.BRANCH_NAME ==~ "PR-\\d+")) {
                                stash name: 'artifacts', includes: '*.tar.gz', excludes: '*-e2e-*'
                                archiveArtifacts '*.tar.gz'
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
                stage("Coverage report") {
                    environment {
                        CODECOV_TOKEN = credentials('jenkins-codecov-token')
                    }
                    agent {
                        label 'linux'
                    }
                    steps {
                        dir('src/github.com/docker/lunchbox') {
                            checkout scm
                            sh 'make ci-coverage'
                            archiveArtifacts '_build/ci-cov/all.out'
                            archiveArtifacts '_build/ci-cov/coverage.html'
                            sh 'curl -s https://codecov.io/bash | bash -s - -f _build/ci-cov/all.out -K'
                        }
                    }
                }
                stage("Test Linux") {
                    agent {
                        label 'linux'
                    }
                    steps  {
                        dir('src/github.com/docker/lunchbox') {
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
                        dir('src/github.com/docker/lunchbox') {
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
                        dir('src/github.com/docker/lunchbox') {
                            unstash "e2e"
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
                dir('src/github.com/docker/lunchbox') {
                    sh 'rm -f *.tar.gz'
                    unstash 'artifacts'
                    echo "Releasing $TAG_NAME"
                    release('docker/lunchbox')
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
        sh("ls *.tar.gz | xargs -i curl -sf -H \"Authorization: token $GITHUB_TOKEN\" -H \"Accept: application/json\" -H \"Content-type: application/gzip\" -X POST --data-binary \"@{}\" $url?name={}")
    }
}
