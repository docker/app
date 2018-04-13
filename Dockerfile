ARG ALPINE_VERSION=3.7
ARG GO_VERSION=1.10.1
ARG TAG=unknown

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build
RUN apk add --no-cache \
  build-base \
  git
WORKDIR /go/src/github.com/docker/lunchbox/
COPY . .

FROM build AS bin-build
ARG TAG
RUN make TAG=${TAG} bin-all

FROM build AS test
ARG TAG
RUN make TAG=${TAG} unit-test e2e-test
