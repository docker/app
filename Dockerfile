ARG GO_VERSION=1.10
ARG RUN_BASE_TAG=3.7
ARG BUILD_BASE_TAG=${GO_VERSION}-alpine${RUN_BASE_TAG}

FROM golang:${BUILD_BASE_TAG} AS build
RUN apk add --no-cache \
  make
WORKDIR /go/src/github.com/docker/lunchbox/
COPY . .
RUN make bin

FROM alpine:${RUN_BASE_TAG} AS run
COPY --from=build /go/src/github.com/docker/lunchbox/_build/bin/docker-app /
ENTRYPOINT ["/docker-app"]
