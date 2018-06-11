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
  util-linux
RUN curl -Ls https://download.docker.com/linux/static/$DOCKERCLI_CHANNEL/x86_64/docker-$DOCKERCLI_VERSION.tgz | \
  tar -xz docker/docker && \
  ls -l . && \
  mv docker/docker /usr/bin/docker

WORKDIR /go/src/github.com/docker/app/

FROM build AS dev
ARG DEP_VERSION=v0.4.1
RUN curl -o /usr/bin/dep -L https://github.com/golang/dep/releases/download/${DEP_VERSION}/dep-linux-amd64 && \
    chmod +x /usr/bin/dep
COPY . .

FROM dev AS docker-app
ARG COMMIT=unknown
ARG TAG=unknown
ARG BUILDTIME=unknown
RUN make COMMIT=${COMMIT} TAG=${TAG} BUILDTIME=${BUILDTIME} bin/docker-app

# FIXME(vdemeester) change from docker-app to dev once buildkit is merged in moby/docker
FROM docker-app AS cross
ARG COMMIT=unknown
ARG TAG=unknown
ARG BUILDTIME=unknown
RUN make COMMIT=${COMMIT} TAG=${TAG} BUILDTIME=${BUILDTIME} cross

# FIXME(vdemeester) change from docker-app to dev once buildkit is merged in moby/docker
FROM cross AS e2e-cross
ARG COMMIT=unknown
ARG TAG=unknown
ARG BUILDTIME=unknown
RUN make COMMIT=${COMMIT} TAG=${TAG} BUILDTIME=${BUILDTIME} e2e-cross
