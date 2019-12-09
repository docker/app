FROM dockercore/golang-cross:1.12.9@sha256:3ea9dcef4dd2c46d80445c0b22d6177817f4cfce22c523cc12a5a1091cb37705 AS cli-build
ENV DISABLE_WARN_OUTSIDE_CONTAINER=1
ARG CLI_CHANNEL=stable
ARG CLI_VERSION=19.03.5

RUN apt-get install -y -q --no-install-recommends \
  coreutils \
  util-linux \
  uuid-runtime

WORKDIR /go/src/github.com/docker/cli

RUN git clone https://github.com/docker/cli . && git checkout v${CLI_VERSION}
RUN mkdir build
RUN curl -fL https://download.docker.com/linux/static/${CLI_CHANNEL}/x86_64/docker-${CLI_VERSION}.tgz | tar xzO docker/docker > build/docker-linux-amd64 && chmod +x build/docker-linux-amd64
RUN curl -fL https://download.docker.com/linux/static/${CLI_CHANNEL}/aarch64/docker-${CLI_VERSION}.tgz | tar xzO docker/docker > build/docker-linux-arm64 && chmod +x build/docker-linux-arm64
RUN curl -fL https://download.docker.com/linux/static/${CLI_CHANNEL}/armhf/docker-${CLI_VERSION}.tgz | tar xzO docker/docker > build/docker-linux-arm && chmod +x build/docker-linux-arm
RUN curl -fL https://download.docker.com/mac/static/${CLI_CHANNEL}/x86_64/docker-${CLI_VERSION}.tgz | tar xzO docker/docker > build/docker-darwin-amd64

ARG GOPROXY
RUN make binary-windows

# main dev image
FROM golang:1.13.3 AS dev

RUN apt-get update && apt-get install -y -q --no-install-recommends \
  coreutils \
  util-linux \
  uuid-runtime

WORKDIR /go/src/github.com/docker/app/
COPY --from=cli-build /go/src/github.com/docker/cli/build/docker-linux-amd64 /usr/bin/docker

ENV PATH=${PATH}:/go/src/github.com/docker/app/bin/
ARG DEP_VERSION=v0.5.4
RUN curl -o /usr/bin/dep -L https://github.com/golang/dep/releases/download/${DEP_VERSION}/dep-linux-amd64 && \
  chmod +x /usr/bin/dep
ARG GOTESTSUM_VERSION=v0.3.4
ARG GOPROXY
RUN mkdir $GOPATH/src/gotest.tools && \
  git clone -q https://github.com/gotestyourself/gotestsum $GOPATH/src/gotest.tools/gotestsum && \
  cd $GOPATH/src/gotest.tools/gotestsum && \
  git -C $GOPATH/src/gotest.tools/gotestsum checkout -q $GOTESTSUM_VERSION && \
  GO111MODULE=on GOOS=linux   go build -o /usr/local/bin/gotestsum-linux       gotest.tools/gotestsum && \
  GO111MODULE=on GOOS=darwin  go build -o /usr/local/bin/gotestsum-darwin      gotest.tools/gotestsum && \
  GO111MODULE=on GOOS=windows go build -o /usr/local/bin/gotestsum-windows.exe gotest.tools/gotestsum && \
  ln -s gotestsum-linux /usr/local/bin/gotestsum
# Source for cmd/test2json is part of the Go distribution and is
# therefore available in the base image.
RUN GOOS=linux go build -o /usr/local/bin/test2json-linux       cmd/test2json && \
  GOOS=darwin  go build -o /usr/local/bin/test2json-darwin      cmd/test2json && \
  GOOS=windows go build -o /usr/local/bin/test2json-windows.exe cmd/test2json
RUN go get -d gopkg.in/mjibson/esc.v0 && \
  cd /go/src/github.com/mjibson/esc && \
  go build -v -o /usr/bin/esc . && \
  rm -rf /go/src/* /go/pkg/* /go/bin/*
COPY . .

FROM scratch AS cli
COPY --from=cli-build /go/src/github.com/docker/cli/build/docker-linux-amd64 docker-linux
COPY --from=cli-build /go/src/github.com/docker/cli/build/docker-darwin-amd64 docker-darwin
COPY --from=cli-build /go/src/github.com/docker/cli/build/docker-windows-amd64 docker-windows.exe

FROM dev AS cross-build
ARG TAG="unknown"
RUN make TAG=${TAG} cross

FROM scratch AS cross
ARG PROJECT_PATH=/go/src/github.com/docker/app
COPY --from=cross-build ${PROJECT_PATH}/bin/docker-app-linux docker-app-linux
COPY --from=cross-build ${PROJECT_PATH}/bin/docker-app-linux-arm64 docker-app-linux-arm64
COPY --from=cross-build ${PROJECT_PATH}/bin/docker-app-linux-arm docker-app-linux-arm
COPY --from=cross-build ${PROJECT_PATH}/bin/docker-app-darwin docker-app-darwin
COPY --from=cross-build ${PROJECT_PATH}/bin/docker-app-windows.exe docker-app-windows.exe

FROM cross-build AS e2e-cross-build
ARG TAG="unknown"
# Run e2e tests
RUN make TAG=${TAG} e2e-cross

FROM scratch AS e2e-cross
ARG PROJECT_PATH=/go/src/github.com/docker/app
COPY --from=e2e-cross-build ${PROJECT_PATH}/bin/docker-app-e2e-linux docker-app-e2e-linux
COPY --from=e2e-cross-build ${PROJECT_PATH}/bin/docker-app-e2e-darwin docker-app-e2e-darwin
COPY --from=e2e-cross-build ${PROJECT_PATH}/bin/docker-app-e2e-windows.exe docker-app-e2e-windows.exe
COPY --from=e2e-cross-build /usr/local/bin/gotestsum-linux gotestsum-linux
COPY --from=e2e-cross-build /usr/local/bin/gotestsum-darwin gotestsum-darwin
COPY --from=e2e-cross-build /usr/local/bin/gotestsum-windows.exe gotestsum-windows.exe
COPY --from=e2e-cross-build /usr/local/bin/test2json-linux test2json-linux
COPY --from=e2e-cross-build /usr/local/bin/test2json-darwin test2json-darwin
COPY --from=e2e-cross-build /usr/local/bin/test2json-windows.exe test2json-windows.exe
