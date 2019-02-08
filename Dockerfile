FROM dockercore/golang-cross:1.11.5@sha256:17a7e0f158521c50316a0d0c1ab1f6a75350b4d82e7ef03c98bcfbdf04feb4f3 AS build
ENV     DISABLE_WARN_OUTSIDE_CONTAINER=1

RUN apt-get install -y -q --no-install-recommends \
    coreutils \
    util-linux \
    uuid-runtime

WORKDIR /go/src/github.com/docker/cli

RUN git clone https://github.com/docker/cli.git .
RUN make cross binary && \
 cp build/docker-linux-amd64 /usr/bin/docker

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
COPY . .

# FIXME(vdemeester) change from docker-app to dev once buildkit is merged in moby/docker
FROM dev AS cross
ARG EXPERIMENTAL="off"
ARG TAG="unknown"
RUN make EXPERIMENTAL=${EXPERIMENTAL} TAG=${TAG} cross

# FIXME(vdemeester) change from docker-app to dev once buildkit is merged in moby/docker
FROM cross AS e2e-cross
ARG EXPERIMENTAL="off"
ARG TAG="unknown"
# Run e2e tests
RUN make EXPERIMENTAL=${EXPERIMENTAL} TAG=${TAG} e2e-cross
