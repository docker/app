ARG ALPINE_VERSION=3.8
ARG GO_VERSION=1.11.0

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build
ARG DOCKERCLI_VERSION=18.03.1-ce
ARG DOCKERCLI_CHANNEL=edge
RUN apk add --no-cache \
  bash \
  make\
  git \
  curl \
  util-linux \
  coreutils \
  build-base
RUN curl -Ls https://download.docker.com/linux/static/$DOCKERCLI_CHANNEL/x86_64/docker-$DOCKERCLI_VERSION.tgz | \
  tar -xz docker/docker && \
  mv docker/docker /usr/bin/docker

WORKDIR /go/src/github.com/docker/app/

# main dev image
FROM build AS dev
ENV PATH=${PATH}:/go/src/github.com/docker/app/bin/
ARG DEP_VERSION=v0.5.0
RUN curl -o /usr/bin/dep -L https://github.com/golang/dep/releases/download/${DEP_VERSION}/dep-linux-amd64 && \
    chmod +x /usr/bin/dep
RUN go get -d gopkg.in/mjibson/esc.v0 && \
    cd /go/src/github.com/mjibson/esc && \
    go build -v -o /usr/bin/esc . && \
    rm -rf /go/src/* /go/pkg/* /go/bin/*
COPY vendor/github.com/deis/duffle /go/src/github.com/deis/duffle
# Build duffle and init
RUN (cd /go/src/github.com/deis/duffle && \
  make bootstrap build-release && \
  ./bin/duffle-linux-amd64 init)
COPY . .

# FIXME(vdemeester) change from docker-app to dev once buildkit is merged in moby/docker
FROM dev AS cross
ARG EXPERIMENTAL="off"
ARG TAG
ARG COMMIT
RUN make EXPERIMENTAL=${EXPERIMENTAL} TAG=${TAG} COMMIT=${COMMIT} cross

# FIXME(vdemeester) change from docker-app to dev once buildkit is merged in moby/docker
FROM cross AS e2e-cross
ARG EXPERIMENTAL="off"
# Run e2e tests
ARG TAG
ARG COMMIT
RUN make EXPERIMENTAL=${EXPERIMENTAL} TAG=${TAG} COMMIT=${COMMIT} e2e-cross

# builder of invocation image entrypoint
FROM build AS invocation-build
COPY . .
ARG EXPERIMENTAL="off"
ARG TAG
ARG COMMIT
RUN make EXPERIMENTAL=${EXPERIMENTAL} TAG=${TAG} COMMIT=${COMMIT} bin/run

# cnab invocation image
FROM alpine:${ALPINE_VERSION} AS invocation
RUN apk add --no-cache ca-certificates
COPY --from=invocation-build /go/src/github.com/docker/app/bin/run /cnab/app/run
WORKDIR /cnab/app
CMD /cnab/app/run
