PKG_NAME := github.com/docker/lunchbox
BIN_NAME := docker-app

# Enable experimental features. "on" or "off"
EXPERIMENTAL := off

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
	-X $(PKG_NAME)/internal.Version=$(TAG)      \
	-X $(PKG_NAME)/internal.Experimental=$(EXPERIMENTAL)"

GO_BUILD := CGO_ENABLED=0 go build
GO_TEST := go test

#####################
# Local Development #
#####################

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
    EXEC_EXT := .exe
endif

all: bin test

check_go_env:
	@test $$(go list) = "$(PKG_NAME)" || \
		(echo "Invalid Go environment" && false)

bin: check_go_env
	@echo "Building _build/$(BIN_NAME)$(EXEC_EXT)..."
	$(GO_BUILD) -ldflags=$(LDFLAGS) -i -o _build/$(BIN_NAME)$(EXEC_EXT)

OS_LIST ?= darwin linux windows
bin-all: check_go_env
	@echo "Building for all platforms..."
	$(foreach OS, $(OS_LIST), GOOS=$(OS) $(GO_BUILD) -ldflags=$(LDFLAGS) -i -o _build/$(TAG)/$(BIN_NAME)-$(OS)$(if $(filter windows, $(OS)),.exe,) || exit 1;)

release:
	gsutil cp -r _build/$(TAG) gs://docker_app

test check: lint unit-test e2e-test

lint:
	@echo "Linting..."
	@tar -c Dockerfile.lint gometalinter.json | docker build -t $(IMAGE_NAME)-lint $(IMAGE_BUILD_ARGS) -f Dockerfile.lint - --target=lint-volume > /dev/null
	@docker run --rm -v $(dir $(realpath $(lastword $(MAKEFILE_LIST)))):/go/src/$(PKG_NAME):ro,cached $(IMAGE_NAME)-lint

e2e-test: bin
	@echo "Running e2e tests..."
	$(GO_TEST) ./e2e/

unit-test:
	@echo "Running unit tests..."
	$(GO_TEST) $(shell go list ./... | grep -vE '/vendor/|/e2e')

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

ci-bin-all:
	docker build -t $(IMAGE_NAME)-bin-all:$(TAG) $(IMAGE_BUILD_ARGS) . --target=bin-build
	$(foreach OS, $(OS_LIST), docker run --rm $(IMAGE_NAME)-bin-all:$(TAG) tar -cz $(BIN_NAME)-$(OS)$(if $(filter windows, $(OS)),.exe,) -C /go/src/$(PKG_NAME)/_build/$(TAG)/ > $(BIN_NAME)-$(OS)-$(TAG).tar.gz || exit 1;)

.PHONY: bin bin-all release test check lint e2e-test unit-test clean ci-lint ci-test ci-bin-all
.DEFAULT: all
