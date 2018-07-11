include vars.mk

LINT_IMAGE_NAME := $(BIN_NAME)-lint:$(TAG)
DEV_IMAGE_NAME := $(BIN_NAME)-dev:$(TAG)
BIN_IMAGE_NAME := $(BIN_NAME)-bin:$(TAG)
CROSS_IMAGE_NAME := $(BIN_NAME)-cross:$(TAG)
E2E_CROSS_IMAGE_NAME := $(BIN_NAME)-e2e-cross:$(TAG)
GRADLE_IMAGE_NAME := $(BIN_NAME)-gradle:$(TAG)

BIN_CTNR_NAME := $(BIN_NAME)-bin-$(TAG)
CROSS_CTNR_NAME := $(BIN_NAME)-cross-$(TAG)
E2E_CROSS_CTNR_NAME := $(BIN_NAME)-e2e-cross-$(TAG)
COV_CTNR_NAME := $(BIN_NAME)-cov-$(TAG)

BUILD_ARGS="--build-arg=EXPERIMENTAL=$(EXPERIMENTAL)"

PKG_PATH := /go/src/$(PKG_NAME)

.DEFAULT: all
all: cross test

create_bin:
	@$(call mkdir,bin)

build_dev_image:
	docker build $(BUILD_ARGS) --target=dev -t $(DEV_IMAGE_NAME) .

shell: build_dev_image ## run a shell in the docker build image
	docker run -ti --rm $(DEV_IMAGE_NAME) bash

cross: create_bin ## cross-compile binaries (linux, darwin, windows)
	docker build $(BUILD_ARGS) --target=cross -t $(CROSS_IMAGE_NAME)  .
	docker create --name $(CROSS_CTNR_NAME) $(CROSS_IMAGE_NAME) noop
	docker cp $(CROSS_CTNR_NAME):$(PKG_PATH)/bin/$(BIN_NAME)-linux bin/$(BIN_NAME)-linux
	docker cp $(CROSS_CTNR_NAME):$(PKG_PATH)/bin/$(BIN_NAME)-darwin bin/$(BIN_NAME)-darwin
	docker cp $(CROSS_CTNR_NAME):$(PKG_PATH)/bin/$(BIN_NAME)-windows.exe bin/$(BIN_NAME)-windows.exe
	docker rm $(CROSS_CTNR_NAME)
	@$(call chmod,+x,bin/$(BIN_NAME)-linux)
	@$(call chmod,+x,bin/$(BIN_NAME)-darwin)
	@$(call chmod,+x,bin/$(BIN_NAME)-windows.exe)

e2e-cross: create_bin
	docker build $(BUILD_ARGS) --target=e2e-cross -t $(E2E_CROSS_IMAGE_NAME)  .
	docker create --name $(E2E_CROSS_CTNR_NAME) $(E2E_CROSS_IMAGE_NAME) noop
	docker cp $(E2E_CROSS_CTNR_NAME):$(PKG_PATH)/bin/$(BIN_NAME)-e2e-linux bin/$(BIN_NAME)-e2e-linux
	docker cp $(E2E_CROSS_CTNR_NAME):$(PKG_PATH)/bin/$(BIN_NAME)-e2e-darwin bin/$(BIN_NAME)-e2e-darwin
	docker cp $(E2E_CROSS_CTNR_NAME):$(PKG_PATH)/bin/$(BIN_NAME)-e2e-windows.exe bin/$(BIN_NAME)-e2e-windows.exe
	docker rm $(E2E_CROSS_CTNR_NAME)
	@$(call chmod,+x,bin/$(BIN_NAME)-e2e-linux)
	@$(call chmod,+x,bin/$(BIN_NAME)-e2e-darwin)
	@$(call chmod,+x,bin/$(BIN_NAME)-e2e-windows.exe)

tars:
	tar czf bin/$(BIN_NAME)-linux.tar.gz -C bin $(BIN_NAME)-linux
	tar czf bin/$(BIN_NAME)-e2e-linux.tar.gz -C bin $(BIN_NAME)-e2e-linux
	tar czf bin/$(BIN_NAME)-darwin.tar.gz -C bin $(BIN_NAME)-darwin
	tar czf bin/$(BIN_NAME)-e2e-darwin.tar.gz -C bin $(BIN_NAME)-e2e-darwin
	tar czf bin/$(BIN_NAME)-windows.tar.gz -C bin $(BIN_NAME)-windows.exe
	tar czf bin/$(BIN_NAME)-e2e-windows.tar.gz -C bin $(BIN_NAME)-e2e-windows.exe

test: test-unit test-e2e ## run all tests

test-unit: build_dev_image ## run unit tests
	docker run --rm $(DEV_IMAGE_NAME) make EXPERIMENTAL=$(EXPERIMENTAL) test-unit

test-e2e: build_dev_image ## run end-to-end tests
	docker run -v /var/run:/var/run:ro --rm --network="host" $(DEV_IMAGE_NAME) make EXPERIMENTAL=$(EXPERIMENTAL) bin/$(BIN_NAME) test-e2e

COV_LABEL := com.docker.app.cov-run=$(TAG)
coverage: build_dev_image ## run tests with coverage
	@$(call mkdir,_build)
	docker run -v /var/run:/var/run:ro --name $(COV_CTNR_NAME) --network="host" -tid $(DEV_IMAGE_NAME) make COMMIT=${COMMIT} TAG=${TAG} EXPERIMENTAL=$(EXPERIMENTAL) coverage
	docker logs -f $(COV_CTNR_NAME)
	docker cp $(COV_CTNR_NAME):$(PKG_PATH)/_build/cov/ ./_build/ci-cov
	docker rm $(COV_CTNR_NAME)

gradle-test:
	tar cf - Dockerfile.gradle bin/docker-app-linux integrations/gradle | docker build -t $(GRADLE_IMAGE_NAME) -f Dockerfile.gradle -
	docker run --rm $(GRADLE_IMAGE_NAME) bash -c "./gradlew --stacktrace build && cd example && gradle renderIt"

lint: ## run linter(s)
	$(info Linting...)
	docker build -t $(LINT_IMAGE_NAME) -f Dockerfile.lint .
	docker run --rm $(LINT_IMAGE_NAME) make lint

vendor: build_dev_image
	$(info Vendoring...)
	docker run --rm $(DEV_IMAGE_NAME) sh -c "make vendor && hack/check-git-diff vendor"

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: lint test-e2e test-unit test cross e2e-cross coverage gradle-test shell build_dev_image tars vendor help
