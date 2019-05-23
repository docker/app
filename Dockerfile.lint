ARG ALPINE_VERSION=3.9
ARG GO_VERSION=1.12.5

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}
RUN apk add --no-cache \
    curl \
    git \
    make \
    coreutils

RUN GO111MODULE=on go get \
    github.com/golangci/golangci-lint/cmd/golangci-lint@v1.16.0 \
    && rm -rf /go/src/* /go/pkg/*

WORKDIR /go/src/github.com/docker/app
ENV CGO_ENABLED=0

COPY . .
