include vars.mk

LDFLAGS := "-s -w \
  -X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
  -X $(PKG_NAME)/internal.Version=$(TAG)"

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
  EXEC_EXT := .exe
endif

INCLUDE_E2E :=
ifneq ($(E2E_TESTS),)
  INCLUDE_E2E := -run $(E2E_TESTS)
endif

TEST_RESULTS_DIR = _build/test-results
STATIC_FLAGS= CGO_ENABLED=0
GO_BUILD = $(STATIC_FLAGS) go build -tags=$(BUILDTAGS) -ldflags=$(LDFLAGS)
GO_TEST = $(STATIC_FLAGS) go test -tags=$(BUILDTAGS) -ldflags=$(LDFLAGS)
GO_TESTSUM = $(STATIC_FLAGS) gotestsum --junitfile $(TEST_RESULTS_DIR)/$(TEST_RESULTS_PREFIX)$(1) -- -tags=$(BUILDTAGS) -ldflags=$(LDFLAGS)

all: bin/$(BIN_NAME) build-invocation-image test

check_go_env:
	@test $$(go list) = "$(PKG_NAME)" || \
		(echo "Invalid Go environment - The local directory structure must match:  $(PKG_NAME)" && false)

cross: cross-plugin ## cross-compile binaries (linux, darwin, windows)

cross-plugin: bin/$(BIN_NAME)-linux bin/$(BIN_NAME)-darwin bin/$(BIN_NAME)-windows.exe bin/$(BIN_NAME)-linux-arm64 bin/$(BIN_NAME)-linux-arm

e2e-cross: bin/$(BIN_NAME)-e2e-linux bin/$(BIN_NAME)-e2e-darwin bin/$(BIN_NAME)-e2e-windows.exe

.PHONY: dynamic
dynamic: STATIC_FLAGS :=
dynamic: bin/$(BIN_NAME)

.PHONY: bin/$(BIN_NAME)-e2e-windows
bin/$(BIN_NAME)-e2e-%.exe bin/$(BIN_NAME)-e2e-%: e2e bin/$(BIN_NAME)-%
	GOOS=$* $(GO_TEST) -c -o $@ ./e2e/

.PHONY: bin/$(BIN_NAME)-linux-arm64
bin/$(BIN_NAME)-linux-arm64: cmd/$(BIN_NAME) check_go_env
	GOOS=linux GOARCH=arm64 $(GO_BUILD) -o $@ ./$<

.PHONY: bin/$(BIN_NAME)-linux-arm
bin/$(BIN_NAME)-linux-arm: cmd/$(BIN_NAME) check_go_env
	GOOS=linux GOARCH=arm $(GO_BUILD) -o $@ ./$<

.PHONY: bin/$(BIN_NAME)-windows
bin/$(BIN_NAME)-%.exe bin/$(BIN_NAME)-%: cmd/$(BIN_NAME) check_go_env
	GOOS=$* $(GO_BUILD) -o $@ ./$<

bin/%: cmd/% check_go_env
	$(GO_BUILD) -o $@$(EXEC_EXT) ./$<

build-invocation-image: ## build invocation image if not present (internal usage, not diplayed in help)
	@echo "Build invocation image if needed"
	$(if $(shell docker images -q docker/cnab-app-base:$(TAG)),, \
		$(MAKE) -f ./docker.Makefile invocation-image \
	)

check: lint test

test: test-unit test-e2e ## run all tests

lint: ## run linter(s)
	@echo "Linting..."
	golangci-lint run --verbose --print-resources-usage --timeout 10m0s ./...

test-e2e: bin/$(BIN_NAME) ## run end-to-end tests
	@echo "Running e2e tests..."
	@$(call mkdir,$(TEST_RESULTS_DIR))
	$(call GO_TESTSUM,e2e.xml) -v ./e2e/ $(INCLUDE_E2E)

test-unit: ## run unit tests
	@echo "Running unit tests..."
	@$(call mkdir,$(TEST_RESULTS_DIR))
	$(call GO_TESTSUM,unit.xml) $(shell go list ./... | grep -vE '/e2e')

coverage-bin:
	CGO_ENABLED=0 go test -tags="$(BUILDTAGS) testrunmain" -ldflags=$(LDFLAGS) -coverpkg="./..." -c -o _build/$(BIN_NAME).cov ./cmd/docker-app

coverage-test-unit:
	@echo "Running unit tests (coverage)..."
	@$(call mkdir,_build/cov)
	@$(call mkdir,$(TEST_RESULTS_DIR))
	$(call GO_TESTSUM,unit-coverage.xml) -cover -test.coverprofile=_build/cov/unit.out $(shell go list ./... | grep -vE '/e2e')

coverage-test-e2e: coverage-bin
	@echo "Running e2e tests (coverage)..."
	@$(call mkdir,_build/cov)
	@$(call mkdir,$(TEST_RESULTS_DIR))
	DOCKERAPP_BINARY=../e2e/coverage-bin $(call GO_TESTSUM,e2e-coverage.xml) -v ./e2e $(INCLUDE_E2E)

coverage: coverage-test-unit coverage-test-e2e ## run tests with coverage
	@echo "Fixing coverage files..."
	find _build/cov/ -type f -name "*.out" -print0 | xargs -0 sed -i '/^coverage/d'
	grep coverage _build/cov/*.out || true
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
	dep ensure -v $(DEP_ARGS)

specification/bindata.go: specification/schemas/*.json
	go generate github.com/docker/app/specification

schemas: specification/bindata.go ## generate specification/bindata.go from json schemas

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: cross e2e-cross test check lint test-unit test-e2e coverage coverage-bin coverage-test-unit coverage-test-e2e clean vendor schemas help fix-coverage
.DEFAULT: all


.PHONY: yamldocs
yamldocs: ## generate documentation YAML files consumed by docs repo
	mkdir -p ./_build/docs
	docker build --output type=local,dest=./_build/ -f docs/yaml/Dockerfile .
