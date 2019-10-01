FROM dockercore/golang-cross:1.12.9@sha256:3ea9dcef4dd2c46d80445c0b22d6177817f4cfce22c523cc12a5a1091cb37705 AS build
ENV     DISABLE_WARN_OUTSIDE_CONTAINER=1

RUN apt-get install -y -q --no-install-recommends \
    coreutils \
    util-linux \
    uuid-runtime

WORKDIR /go/src/github.com/docker/cli

RUN git clone https://github.com/docker/cli . && git checkout a1b83ffd2cbeefc0752e5aa7a543d49c1ddfd2cb

ARG GOPROXY
RUN make binary-osx binary-windows binary && \
 cp build/docker-linux-amd64 /usr/bin/docker

WORKDIR /go/src/github.com/docker/app/

# main dev image
FROM build AS dev
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
RUN GOOS=linux   go build -o /usr/local/bin/test2json-linux       cmd/test2json && \
    GOOS=darwin  go build -o /usr/local/bin/test2json-darwin      cmd/test2json && \
    GOOS=windows go build -o /usr/local/bin/test2json-windows.exe cmd/test2json
RUN go get -d gopkg.in/mjibson/esc.v0 && \
    cd /go/src/github.com/mjibson/esc && \
    go build -v -o /usr/bin/esc . && \
    rm -rf /go/src/* /go/pkg/* /go/bin/*
COPY . .

FROM dev AS cross
ARG EXPERIMENTAL="off"
ARG TAG="unknown"
RUN make EXPERIMENTAL=${EXPERIMENTAL} TAG=${TAG} cross

FROM cross AS e2e-cross
ARG EXPERIMENTAL="off"
ARG TAG="unknown"
# Run e2e tests
RUN make EXPERIMENTAL=${EXPERIMENTAL} TAG=${TAG} e2e-cross
