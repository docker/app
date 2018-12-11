include vars.mk

ifeq ($(BUILDTIME),)
  BUILDTIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" 2> /dev/null)
endif
ifeq ($(BUILDTIME),)
  BUILDTIME := unknown
  $(warning unable to set BUILDTIME. Set the value manually)
endif

BUILDTAGS=""
ifeq ($(EXPERIMENTAL),on)
  BUILDTAGS="experimental"
endif

LDFLAGS := "-s -w \
  -X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
  -X $(PKG_NAME)/internal.Version=$(TAG)      \
  -X $(PKG_NAME)/internal.Experimental=$(EXPERIMENTAL) \
  -X $(PKG_NAME)/internal.BuildTime=$(BUILDTIME)"

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
  EXEC_EXT := .exe
endif

GO_BUILD := CGO_ENABLED=0 go build -tags=$(BUILDTAGS) -ldflags=$(LDFLAGS)
GO_TEST := CGO_ENABLED=0 go test -tags=$(BUILDTAGS) -ldflags=$(LDFLAGS)

ADVERTISE_ADDRESS_OPTION :=
# When on Linux get the default network
ifneq ($(OS),Windows_NT)
  UNAME_S := $(shell uname -s)
  ifeq ($(UNAME_S),Linux)
    DEFAULT_NETWORK_INTERFACE := $(shell netstat -r | grep default | awk '{print $$NF}')
    ADVERTISE_ADDRESS_OPTION := --advertise-addr $(DEFAULT_NETWORK_INTERFACE)
  endif
endif

INITIAL_SWARM_MODE := "off"
$(shell docker node list $(TO_NULL))
ifeq ($(.SHELLSTATUS),0)
	INITIAL_SWARM_MODE="on"
endif

all: bin/$(BIN_NAME) test

check_go_env:
	@test $$(go list) = "$(PKG_NAME)" || \
		(echo "Invalid Go environment - The local directory structure must match:  $(PKG_NAME)" && false)

cross: bin/$(BIN_NAME)-linux bin/$(BIN_NAME)-darwin bin/$(BIN_NAME)-windows.exe ## cross-compile binaries (linux, darwin, windows)

e2e-cross: bin/$(BIN_NAME)-e2e-linux bin/$(BIN_NAME)-e2e-darwin bin/$(BIN_NAME)-e2e-windows.exe

.PHONY: bin/$(BIN_NAME)-e2e-windows
bin/$(BIN_NAME)-e2e-%.exe bin/$(BIN_NAME)-e2e-%: e2e bin/$(BIN_NAME)-%
	GOOS=$* $(GO_TEST) -c -o $@ ./e2e/

.PHONY: bin/$(BIN_NAME)-windows
bin/$(BIN_NAME)-%.exe bin/$(BIN_NAME)-%: cmd/$(BIN_NAME) check_go_env
	GOOS=$* $(GO_BUILD) -o $@ ./$<

bin/%: cmd/% check_go_env
	$(GO_BUILD) -o $@$(EXEC_EXT) ./$<

check: lint test

test: test-unit test-e2e ## run all tests

lint: ## run linter(s)
	@echo "Linting..."
	@gometalinter --config=gometalinter.json ./...

set-credentials:
	# Set credentials
	@bin/$(BIN_NAME) add-credentialset --force creds-test testTargetContext
	@bin/$(BIN_NAME) context rm testTargetContext # Ideal would be having a "--force" on create
	@bin/$(BIN_NAME) context create testTargetContext

swarm-init:
	@if [ $(INITIAL_SWARM_MODE) = "off" ]; then\
		docker swarm init $(ADVERTISE_ADDRESS_OPTION) $(TO_NULL); # Required by "docker-app install" \
	fi

--run-test-e2e: bin/$(BIN_NAME) set-credentials swarm-init ## Private target that actually runs end-to-end tests, please call `test-e2e` instead
	@echo "Running e2e tests..."
	$(GO_TEST) -v ./e2e/

test-e2e: --run-test-e2e
	@if [ $(INITIAL_SWARM_MODE) = "off" ]; then\
		docker swarm leave --force 2>&1 >/dev/null; # Unique manager requires "--force" to leave \
	fi

test-unit: ## run unit tests
	@echo "Running unit tests..."
	$(GO_TEST) $(shell go list ./... | grep -vE '/e2e')

coverage-bin:
	CGO_ENABLED=0 go test -tags="$(BUILDTAGS) testrunmain" -ldflags=$(LDFLAGS) -coverpkg="./..." -c -o _build/$(BIN_NAME).cov ./cmd/docker-app

coverage-test-unit:
	@echo "Running unit tests (coverage)..."
	@$(call mkdir,_build/cov)
	$(GO_TEST) -cover -test.coverprofile=_build/cov/unit.out $(shell go list ./... | grep -vE '/e2e')

coverage-test-e2e: coverage-bin
	@echo "Running e2e tests (coverage)..."
	@$(call mkdir,_build/cov)
	DOCKERAPP_BINARY=../e2e/coverage-bin $(GO_TEST) -v ./e2e

coverage: coverage-test-unit coverage-test-e2e ## run tests with coverage
	go install ./vendor/github.com/wadey/gocovmerge/
	gocovmerge _build/cov/*.out > _build/cov/all.out
	go tool cover -func _build/cov/all.out
	go tool cover -html _build/cov/all.out -o _build/cov/coverage.html

clean: ## clean build artifacts
	$(call rmdir,bin)
	$(call rmdir,_build)
	$(call rm,docker-app-*.tar.gz)

vendor: ## update vendoring
	$(call rmdir,vendor)
	dep ensure -v

specification/bindata.go: specification/schemas/*.json
	go generate github.com/docker/app/specification

schemas: specification/bindata.go ## generate specification/bindata.go from json schemas

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: cross e2e-cross test check lint test-unit test-e2e coverage coverage-bin coverage-test-unit coverage-test-e2e clean vendor schemas help
.DEFAULT: all
