PKG_NAME := github.com/docker/lunchbox
BIN_NAME := docker-app

TAG := ${shell git describe --always --dirty}

IMAGE_NAME := docker-app

GO_VERSION := 1.10
RUN_BASE_TAG := 3.7

IMAGE_BUILD_ARGS := \
    --build-arg GO_VERSION=$(GO_VERSION) \
    --build-arg RUN_BASE_TAG=$(RUN_BASE_TAG)

LDFLAGS :=

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
    EXEC_EXT := .exe
endif

all: bin test

CHECK_GO_ENV:
	@test $$(go list) = "$(PKG_NAME)" || \
		(echo "Invalid Go environment" && false)

bin: CHECK_GO_ENV
	@echo "Building _build/bin/$(BIN_NAME)$(EXEC_EXT)..."
	@go build -ldflags=$(LDFLAGS) -i -o _build/bin/$(BIN_NAME)$(EXEC_EXT) ./

bin-all: CHECK_GO_ENV
	@echo "Building for all archs in _build/$(TAG)"
	GOOS=windows go build -ldflags=$(LDFLAGS) -i -o _build/$(TAG)/$(BIN_NAME)-windows.exe ./
	GOOS=linux go build -ldflags=$(LDFLAGS) -i -o _build/$(TAG)/$(BIN_NAME)-linux ./
	GOOS=darwin go build -ldflags=$(LDFLAGS) -i -o _build/$(TAG)/$(BIN_NAME)-macos ./

release:
	gsutil cp -r _build/$(TAG) gs://docker_app

image:
	@docker build -t $(IMAGE_NAME) $(IMAGE_BUILD_ARGS) . --target run

test: lint unit-test e2e-test

lint:
	@echo "Linting..."
	@tar -c Dockerfile.lint gometalinter.json | docker build -t $(IMAGE_NAME)-lint $(IMAGE_BUILD_ARGS) -f Dockerfile.lint - > /dev/null
	@docker run --rm -v $(dir $(realpath $(lastword $(MAKEFILE_LIST)))):/go/src/$(PKG_NAME) $(IMAGE_NAME)-lint

e2e-test:
	@echo "Running e2e tests..."
	@go test ./e2e/

unit-test:
	@echo "Running unit tests..."
	@go test $(shell go list ./... | grep -vE '/vendor/|/e2e')

clean:
	rm -Rf ./_build

.PHONY: bin image test lint e2e-test unit-test clean
.DEFAULT: all
