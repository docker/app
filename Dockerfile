ARG ALPINE_VERSION=3.7
ARG GO_VERSION=1.10.3

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
ENV PATH=${PATH}:/go/src/github.com/docker/app/bin/
ARG DEP_VERSION=v0.4.1
RUN curl -o /usr/bin/dep -L https://github.com/golang/dep/releases/download/${DEP_VERSION}/dep-linux-amd64 && \
    chmod +x /usr/bin/dep
COPY . .

# FIXME(vdemeester) change from docker-app to dev once buildkit is merged in moby/docker
FROM dev AS cross
ARG EXPERIMENTAL="off"
RUN make EXPERIMENTAL=${EXPERIMENTAL} cross

# FIXME(vdemeester) change from docker-app to dev once buildkit is merged in moby/docker
FROM cross AS e2e-cross
ARG EXPERIMENTAL="off"
RUN make EXPERIMENTAL=${EXPERIMENTAL} e2e-cross
