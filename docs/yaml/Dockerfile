FROM golang:1.13-alpine AS base

ARG PROJECT=github.com/docker/app
ARG PROJECT_PATH=/go/src/${PROJECT}
ENV CGO_ENABLED=0
ENV PATH=${PATH}:${PROJECT_PATH}/build
WORKDIR $PROJECT_PATH

RUN mkdir -p docs/yaml/gen

COPY . .
RUN go build -o build/yaml-docs-generator ${PROJECT}/docs/yaml
RUN build/yaml-docs-generator \
  --root   "${PROJECT_PATH}" \
  --target "${PROJECT_PATH}/docs/yaml/gen"


FROM scratch
ARG PROJECT=github.com/docker/app
ARG PROJECT_PATH=/go/src/${PROJECT}
# CMD cannot be nil so we set it to empty string
CMD  [""]
COPY --from=base ${PROJECT_PATH}/docs/reference /docs/reference
COPY --from=base ${PROJECT_PATH}/docs/yaml/gen /docs/yaml
