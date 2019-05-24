ARG ALPINE_VERSION=3.9.4

FROM dockercore/golang-cross:1.12.5@sha256:15b5f9805c0395d3ad80f9354ee358312e1abe3a683ce80371ad0551199ff253 AS build

RUN apt-get install -y -q --no-install-recommends \
    coreutils \
    util-linux \
    uuid-runtime

WORKDIR /go/src/github.com/docker/app/

COPY . .
ARG EXPERIMENTAL="off"
ARG TAG="unknown"
RUN make EXPERIMENTAL=${EXPERIMENTAL} BUILD_TAG=${BUILD_TAG} TAG=${TAG} bin/cnab-run

 # local cnab invocation image
FROM alpine:${ALPINE_VERSION} as invocation
RUN apk add --no-cache ca-certificates && adduser -S cnab
USER cnab
COPY --from=build /go/src/github.com/docker/app/bin/cnab-run /cnab/app/run
WORKDIR /cnab/app
CMD /cnab/app/run
