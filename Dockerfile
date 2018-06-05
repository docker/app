ARG ALPINE_VERSION=3.7
ARG GO_VERSION=1.10.1
ARG COMMIT=unknown
ARG TAG=unknown
ARG BUILDTIME=unknown

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build
RUN apk add --no-cache \
  bash \
  build-base \
  docker \
  git \
  util-linux
WORKDIR /go/src/github.com/docker/app/

FROM build AS bin-build
COPY . .
ARG COMMIT
ARG TAG
ARG BUILDTIME=unknown
RUN make COMMIT=${COMMIT} TAG=${TAG} BUILDTIME=${BUILDTIME} bin

FROM build AS bin-all
COPY . .
ARG COMMIT
ARG TAG
ARG BUILDTIME=unknown
RUN make COMMIT=${COMMIT} TAG=${TAG} BUILDTIME=${BUILDTIME} bin-all e2e-all

FROM build AS test
COPY . .
ARG COMMIT
ARG TAG
ARG BUILDTIME=unknown
RUN make COMMIT=${COMMIT} TAG=${TAG} BUILDTIME=${BUILDTIME} unit-test

FROM java:8-jdk AS gradle_test
WORKDIR /app
COPY integrations/gradle .
COPY --from=bin-build /go/src/github.com/docker/app/_build/docker-app /usr/local/bin
RUN ./gradlew --stacktrace build && \
    cd example && ./gradlew renderIt
