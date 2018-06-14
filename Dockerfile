ARG ALPINE_VERSION=3.7
ARG GO_VERSION=1.10.2

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build
ARG DOCKERCLI_VERSION=18.03.1-ce
ARG DOCKERCLI_CHANNEL=edge
RUN apk add --no-cache \
  bash \
  make\
  git \
  curl \
  util-linux \
  coreutils
RUN curl -Ls https://download.docker.com/linux/static/$DOCKERCLI_CHANNEL/x86_64/docker-$DOCKERCLI_VERSION.tgz | \
  tar -xz docker/docker && \
  mv docker/docker /usr/bin/docker

WORKDIR /go/src/github.com/docker/app/

FROM build AS dev
ARG DEP_VERSION=v0.4.1
RUN curl -o /usr/bin/dep -L https://github.com/golang/dep/releases/download/${DEP_VERSION}/dep-linux-amd64 && \
    chmod +x /usr/bin/dep
ARG STUPID_VERSION=0.0.3
RUN mkdir bin && \
    curl -o bin/stupid-darwin -L https://github.com/jeanlaurent/stupid/releases/download/${STUPID_VERSION}/stupid-darwin && \
    curl -o bin/stupid-linux -L https://github.com/jeanlaurent/stupid/releases/download/${STUPID_VERSION}/stupid-linux && \
    curl -o bin/stupid-windows.exe -L https://github.com/jeanlaurent/stupid/releases/download/${STUPID_VERSION}/stupid-windows.exe && \
    chmod +x bin/stupid-linux
COPY . .

FROM dev AS cross
RUN make cross

FROM cross AS e2e-cross
RUN make e2e-cross
