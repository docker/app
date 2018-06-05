PKG_NAME := github.com/docker/app
BIN_NAME := docker-app
E2E_NAME := $(BIN_NAME)-e2e

# Enable experimental features. "on" or "off"
EXPERIMENTAL := off

# Comma-separated list of renderers
RENDERERS := "none"

TAG ?= $(shell git describe --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
CWD = $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

# Used by ci-gradle-test target
DOCKERAPP_BINARY ?= $(CWD)/_build/$(BIN_NAME)-linux

IMAGE_NAME := docker-app

ifeq ($(BUILDTIME),)
  BUILDTIME := ${shell date --utc --rfc-3339 ns 2> /dev/null | sed -e 's/ /T/'}
endif
ifeq ($(BUILDTIME),)
  BUILDTIME := ${shell gdate --utc --rfc-3339 ns 2> /dev/null | sed -e 's/ /T/'}
endif
ifeq ($(BUILDTIME),)
  $(error unable to set BUILDTIME, ensure that you have GNU date installed or set manually)
endif

IMAGE_BUILD_ARGS := \
    --build-arg COMMIT=$(COMMIT) \
    --build-arg TAG=$(TAG)       \
    --build-arg BUILDTIME=$(BUILDTIME)


LDFLAGS := "-s -w \
	-X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
	-X $(PKG_NAME)/internal.Version=$(TAG)      \
	-X $(PKG_NAME)/internal.Experimental=$(EXPERIMENTAL) \
	-X $(PKG_NAME)/internal.Renderers=$(RENDERERS) \
	-X $(PKG_NAME)/internal.BuildTime=$(BUILDTIME)"

GO_BUILD := CGO_ENABLED=0 go build
GO_TEST := CGO_ENABLED=0 go test

#####################
# Local Development #
#####################

OS_LIST ?= darwin linux windows

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
    EXEC_EXT := .exe
endif

PKG_PATH := /go/src/$(PKG_NAME)

all: bin test

check_go_env:
	@test $$(go list) = "$(PKG_NAME)" || \
		(echo "Invalid Go environment" && false)

bin: check_go_env
	@echo "Building _build/$(BIN_NAME)$(EXEC_EXT)..."
	$(GO_BUILD) -ldflags=$(LDFLAGS) -o _build/$(BIN_NAME)$(EXEC_EXT) ./cmd/docker-app

bin-all: check_go_env
	@echo "Building for all platforms..."
	$(foreach OS, $(OS_LIST), GOOS=$(OS) $(GO_BUILD) -ldflags=$(LDFLAGS) -o _build/$(BIN_NAME)-$(OS)$(if $(filter windows, $(OS)),.exe,) ./cmd/docker-app || exit 1;)

e2e-all: check_go_env
	@echo "Building for all platforms..."
	$(foreach OS, $(OS_LIST), GOOS=$(OS) $(GO_TEST) -c -o _build/$(E2E_NAME)-$(OS)$(if $(filter windows, $(OS)),.exe,) ./e2e || exit 1;)

test check: lint unit-test e2e-test

lint:
	@echo "Linting..."
	@tar -c Dockerfile.lint gometalinter.json | docker build -t $(IMAGE_NAME)-lint $(IMAGE_BUILD_ARGS) -f Dockerfile.lint - > /dev/null
	@docker run --rm -v $(CWD):$(PKG_PATH):ro,cached $(IMAGE_NAME)-lint

e2e-test: bin
	@echo "Running e2e tests..."
	$(GO_TEST) ./e2e/

unit-test:
	@echo "Running unit tests..."
	$(GO_TEST) $(shell go list ./... | grep -vE '/e2e')

coverage-bin:
	$(GO_TEST) -coverpkg="./..." -c -ldflags=$(LDFLAGS) -tags testrunmain -o _build/$(BIN_NAME).cov ./cmd/docker-app
	go install ./vendor/github.com/wadey/gocovmerge/

coverage: coverage-bin
	mkdir -p _build/cov
	@echo "Running e2e tests (coverage)..."
	DOCKERAPP_BINARY=../e2e/coverage-bin $(GO_TEST) -v ./e2e
	@echo "Running unit tests (coverage)..."
	$(GO_TEST) -cover -test.coverprofile=_build/cov/unit.out $(shell go list ./... | grep -vE '/e2e')
	gocovmerge _build/cov/*.out > _build/cov/all.out
	go tool cover -func _build/cov/all.out
	go tool cover -html _build/cov/all.out -o _build/cov/coverage.html

clean:
	rm -Rf ./_build docker-app-*.tar.gz

##########################
# Continuous Integration #
##########################

COV_LABEL := com.docker.app.cov-run=$(TAG)

ci-lint:
	@echo "Linting..."
	docker build -t $(IMAGE_NAME)-lint:$(TAG) $(IMAGE_BUILD_ARGS) -f Dockerfile.lint .
	docker run --rm $(IMAGE_NAME)-lint:$(TAG)

ci-test:
	@echo "Testing..."
	docker build -t $(IMAGE_NAME)-test:$(TAG) $(IMAGE_BUILD_ARGS) . --target=test

ci-coverage:
	docker build --target=build -t $(IMAGE_NAME)-cov:$(TAG) $(IMAGE_BUILD_ARGS) .
	docker run -v /var/run/docker.sock:/var/run/docker.sock --label $(COV_LABEL) $(IMAGE_NAME)-cov:$(TAG) make COMMIT=$(TAG) TAG=$(COMMIT) BUILDTIME=$(BUILDTIME) coverage
	mkdir -p ./_build && docker cp $$(docker ps -aql --filter label=$(COV_LABEL)):$(PKG_PATH)/_build/cov/ ./_build/ci-cov

ci-bin-all:
	docker build -t $(IMAGE_NAME)-bin-all:$(TAG) $(IMAGE_BUILD_ARGS) . --target=bin-build
	$(foreach OS, $(OS_LIST), docker run --rm $(IMAGE_NAME)-bin-all:$(TAG) tar -cz -C $(PKG_PATH)/_build $(BIN_NAME)-$(OS)$(if $(filter windows, $(OS)),.exe,) > $(BIN_NAME)-$(OS)-$(TAG).tar.gz || exit 1;)
	$(foreach OS, $(OS_LIST), docker run --rm $(IMAGE_NAME)-bin-all:$(TAG) /bin/sh -c "cp $(PKG_PATH)/_build/*-$(OS)* $(PKG_PATH)/e2e && cd $(PKG_PATH)/e2e && tar -cz * --exclude=*.go" > $(E2E_NAME)-$(OS)-$(TAG).tar.gz || exit 1;)

ci-gradle-test:
	docker run --user $(shell id -u) --rm -v $(CWD)/integrations/gradle:/gradle -v $(DOCKERAPP_BINARY):/usr/local/bin/docker-app \
	  -e GRADLE_USER_HOME=/tmp/gradle \
	  gradle:jdk8 bash -c "cd /gradle && ./gradlew --stacktrace build && cd example && gradle renderIt"

.PHONY: bin bin-all release test check lint test-cov e2e-test e2e-all unit-test coverage coverage-bin clean ci-lint ci-test ci-coverage ci-bin-all ci-e2e-all ci-gradle-test
.DEFAULT: all
