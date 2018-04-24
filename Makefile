PKG_NAME := github.com/docker/lunchbox
BIN_NAME := docker-app

TAG ?= $(shell git describe --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)

IMAGE_NAME := docker-app

ALPINE_VERSION := 3.7
GO_VERSION := 1.10.1

IMAGE_BUILD_ARGS := \
    --build-arg ALPINE_VERSION=$(ALPINE_VERSION) \
    --build-arg GO_VERSION=$(GO_VERSION) \
    --build-arg BIN_NAME=$(BIN_NAME) \
    --build-arg COMMIT=$(COMMIT) \
    --build-arg TAG=$(TAG)

LDFLAGS := "-s -w \
	-X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
	-X $(PKG_NAME)/internal.Version=$(TAG)"

#####################
# Local Development #
#####################

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
    EXEC_EXT := .exe
endif

all: bin test

CHECK_GO_ENV:
	@test $$(go list) = "$(PKG_NAME)" || \
		(echo "Invalid Go environment" && false)

bin: CHECK_GO_ENV
	@echo "Building _build/$(BIN_NAME)$(EXEC_EXT)..."
	go build -ldflags=$(LDFLAGS) -i -o _build/$(BIN_NAME)$(EXEC_EXT)

OS_LIST ?= darwin linux windows
bin-all: CHECK_GO_ENV
	@echo "Building for all platforms..."
	$(foreach OS, $(OS_LIST), GOOS=$(OS) go build -ldflags=$(LDFLAGS) -i -o _build/$(TAG)/$(BIN_NAME)-$(OS)$(if $(filter windows, $(OS)),.exe,) || exit 1;)

release:
	gsutil cp -r _build/$(TAG) gs://docker_app

test check: lint unit-test e2e-test

lint:
	@echo "Linting..."
	@tar -c Dockerfile.lint gometalinter.json | docker build -t $(IMAGE_NAME)-lint $(IMAGE_BUILD_ARGS) -f Dockerfile.lint - --target=lint-volume > /dev/null
	@docker run --rm -v $(dir $(realpath $(lastword $(MAKEFILE_LIST)))):/go/src/$(PKG_NAME):ro,cached $(IMAGE_NAME)-lint

e2e-test:
	@echo "Running e2e tests..."
	go test ./e2e/

unit-test:
	@echo "Running unit tests..."
	go test $(shell go list ./... | grep -vE '/vendor/|/e2e')

clean:
	rm -Rf ./_build docker-app-*.tar.gz

######
# CI #
######

ci-lint:
	@echo "Linting..."
	docker build -t $(IMAGE_NAME)-lint:$(TAG) $(IMAGE_BUILD_ARGS) -f Dockerfile.lint . --target=lint-image
	docker run --rm $(IMAGE_NAME)-lint:$(TAG)

ci-test:
	@echo "Testing..."
	docker build -t $(IMAGE_NAME)-test:$(TAG) $(IMAGE_BUILD_ARGS) . --target=test

ci-bin-%:
	@echo "Building tarball for $*..."
	docker build -t $(IMAGE_NAME)-bin-all:$(TAG) $(IMAGE_BUILD_ARGS) . --target=bin-build
	docker run --rm $(IMAGE_NAME)-bin-all:$(TAG) tar -cz $(BIN_NAME)-$*$(if $(filter windows, $*),.exe,) -C /go/src/$(PKG_NAME)/_build/$(TAG)/ > $(BIN_NAME)-$*-$(TAG).tar.gz

.PHONY: bin bin-all release test check lint e2e-test unit-test clean ci-lint ci-test
.DEFAULT: all
