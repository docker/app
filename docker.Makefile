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
SCHEMAS_CTNR_NAME := $(BIN_NAME)-schemas-$(TAG)

DUFFLE_BIN_NAME := duffle
DUFFLE_PKG := /go/src/github.com/deis/duffle

BUILD_ARGS=--build-arg=EXPERIMENTAL=$(EXPERIMENTAL) --build-arg=TAG=$(TAG) --build-arg=COMMIT=$(COMMIT)

PKG_PATH := /go/src/$(PKG_NAME)

CNAB_BASE_INVOCATION_IMAGE_NAME := docker/cnab-app-base:$(TAG)

CNAB_BASE_INVOCATION_IMAGE_PATH := /tmp/invocation-image-$(TAG).tar
RM := rm
ifeq ($(OS),Windows_NT)
  CNAB_BASE_INVOCATION_IMAGE_PATH := c:\tmp\invocation-image-$(TAG).tar
  RM := del
endif

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
	docker cp $(CROSS_CTNR_NAME):$(DUFFLE_PKG)/bin/$(DUFFLE_BIN_NAME)-linux-amd64 bin/$(DUFFLE_BIN_NAME)-linux
	docker cp $(CROSS_CTNR_NAME):$(DUFFLE_PKG)/bin/$(DUFFLE_BIN_NAME)-darwin-amd64 bin/$(DUFFLE_BIN_NAME)-darwin
	docker cp $(CROSS_CTNR_NAME):$(DUFFLE_PKG)/bin/$(DUFFLE_BIN_NAME)-windows-amd64.exe bin/$(DUFFLE_BIN_NAME)-windows.exe
	docker rm $(CROSS_CTNR_NAME)
	@$(call chmod,+x,bin/$(BIN_NAME)-linux)
	@$(call chmod,+x,bin/$(BIN_NAME)-darwin)
	@$(call chmod,+x,bin/$(BIN_NAME)-windows.exe)
	@$(call chmod,+x,bin/$(DUFFLE_BIN_NAME)-linux)
	@$(call chmod,+x,bin/$(DUFFLE_BIN_NAME)-darwin)
	@$(call chmod,+x,bin/$(DUFFLE_BIN_NAME)-windows.exe)

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
	tar czf bin/$(BIN_NAME)-linux.tar.gz -C bin $(BIN_NAME)-linux $(DUFFLE_BIN_NAME)-linux
	tar czf bin/$(BIN_NAME)-e2e-linux.tar.gz -C bin $(BIN_NAME)-e2e-linux
	tar czf bin/$(BIN_NAME)-darwin.tar.gz -C bin $(BIN_NAME)-darwin $(DUFFLE_BIN_NAME)-darwin
	tar czf bin/$(BIN_NAME)-e2e-darwin.tar.gz -C bin $(BIN_NAME)-e2e-darwin
	tar czf bin/$(BIN_NAME)-windows.tar.gz -C bin $(BIN_NAME)-windows.exe $(DUFFLE_BIN_NAME)-windows.exe
	tar czf bin/$(BIN_NAME)-e2e-windows.tar.gz -C bin $(BIN_NAME)-e2e-windows.exe

test: test-unit test-e2e ## run all tests

test-unit: build_dev_image ## run unit tests
	docker run --rm $(DEV_IMAGE_NAME) make EXPERIMENTAL=$(EXPERIMENTAL) test-unit

test-e2e: build_dev_image invocation-image ## run end-to-end tests
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

specification/bindata.go: specification/schemas/*.json build_dev_image
	docker run --name $(SCHEMAS_CTNR_NAME) $(DEV_IMAGE_NAME) sh -c "make schemas"
	docker cp $(SCHEMAS_CTNR_NAME):$(PKG_PATH)/specification/bindata.go ./specification/
	docker rm $(SCHEMAS_CTNR_NAME)

schemas: specification/bindata.go ## generate specification/bindata.go from json schemas

invocation-image:
	docker build $(BUILD_ARGS) --target=invocation -t $(CNAB_BASE_INVOCATION_IMAGE_NAME) .

save-invocation-image: invocation-image
	docker save $(CNAB_BASE_INVOCATION_IMAGE_NAME) -o $(CNAB_BASE_INVOCATION_IMAGE_PATH)

load-invocation-image:
	docker load -i $(CNAB_BASE_INVOCATION_IMAGE_PATH)
	$(RM) $(CNAB_BASE_INVOCATION_IMAGE_PATH)

clean-invocation-image:
	$(RM) $(CNAB_BASE_INVOCATION_IMAGE_PATH)

push-invocation-image:
	docker push $(CNAB_BASE_INVOCATION_IMAGE_NAME)

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: lint test-e2e test-unit test cross e2e-cross coverage gradle-test shell build_dev_image tars vendor schemas help invocation-image
