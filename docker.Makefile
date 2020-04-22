include vars.mk

LINT_IMAGE_NAME := $(BIN_NAME)-lint:$(BUILD_TAG)
DEV_IMAGE_NAME := $(BIN_NAME)-dev:$(BUILD_TAG)
BIN_IMAGE_NAME := $(BIN_NAME)-bin:$(BUILD_TAG)
CROSS_IMAGE_NAME := $(BIN_NAME)-cross:$(BUILD_TAG)
CLI_IMAGE_NAME := $(BIN_NAME)-cli:$(BUILD_TAG)
E2E_CROSS_IMAGE_NAME := $(BIN_NAME)-e2e-cross:$(BUILD_TAG)

BIN_CTNR_NAME := $(BIN_NAME)-bin-$(TAG)
CLI_CNTR_NAME := $(BIN_NAME)-cli-$(TAG)
CROSS_CTNR_NAME := $(BIN_NAME)-cross-$(TAG)
E2E_CROSS_CTNR_NAME := $(BIN_NAME)-e2e-cross-$(TAG)
COV_CTNR_NAME := $(BIN_NAME)-cov-$(TAG)
SCHEMAS_CTNR_NAME := $(BIN_NAME)-schemas-$(TAG)

BUILD_ARGS=--build-arg TAG=$(TAG) --build-arg COMMIT=$(COMMIT) --build-arg ALPINE_VERSION=$(ALPINE_VERSION) --build-arg GOPROXY=$(GOPROXY)

PKG_PATH := /go/src/$(PKG_NAME)


CNAB_BASE_INVOCATION_IMAGE_NAME := docker/cnab-app-base:$(BUILD_TAG)
CNAB_BASE_INVOCATION_IMAGE_PATH := _build/invocation-image

PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME := docker/cnab-app-base:$(TAG)

.DEFAULT: all
all: cross test

create_bin:
	@$(call mkdir,bin)

build_dev_image:
	docker build $(BUILD_ARGS) --target=dev -t $(DEV_IMAGE_NAME) .

shell: build_dev_image ## run a shell in the docker build image
	docker run -ti --rm $(DEV_IMAGE_NAME) bash

cross: create_bin ## cross-compile binaries (linux, darwin, windows)
	docker build $(BUILD_ARGS) --output type=local,dest=./bin/ --target=cross -t $(CROSS_IMAGE_NAME) .
	@$(call chmod,+x,bin/$(BIN_NAME)-linux)
	@$(call chmod,+x,bin/$(BIN_NAME)-linux-arm64)
	@$(call chmod,+x,bin/$(BIN_NAME)-linux-arm)
	@$(call chmod,+x,bin/$(BIN_NAME)-darwin)
	@$(call chmod,+x,bin/$(BIN_NAME)-windows.exe)

cli-cross: create_bin
	docker build $(BUILD_ARGS) --output type=local,dest=./bin/ --target=cli -t $(CLI_IMAGE_NAME) .
	@$(call chmod,+x,bin/docker-linux)
	@$(call chmod,+x,bin/docker-darwin)
	@$(call chmod,+x,bin/docker-windows.exe)

e2e-cross: create_bin
	docker build $(BUILD_ARGS) --output type=local,dest=./bin/ --target=e2e-cross -t $(E2E_CROSS_IMAGE_NAME)  .
	@$(call chmod,+x,bin/$(BIN_NAME)-e2e-linux)
	@$(call chmod,+x,bin/$(BIN_NAME)-e2e-darwin)
	@$(call chmod,+x,bin/gotestsum-linux)
	@$(call chmod,+x,bin/gotestsum-darwin)
	@$(call chmod,+x,bin/test2json-linux)
	@$(call chmod,+x,bin/test2json-darwin)

tars:
	tar --transform='flags=r;s|$(BIN_NAME)-linux|$(BIN_NAME)-plugin-linux|' -czf bin/$(BIN_NAME)-linux.tar.gz -C bin $(BIN_NAME)-linux
	tar czf bin/$(BIN_NAME)-e2e-linux.tar.gz -C bin $(BIN_NAME)-e2e-linux
	tar --transform='flags=r;s|$(BIN_NAME)-linux-arm64|$(BIN_NAME)-plugin-linux-arm64|' -czf bin/$(BIN_NAME)-linux-arm64.tar.gz -C bin $(BIN_NAME)-linux-arm64
	tar --transform='flags=r;s|$(BIN_NAME)-linux-arm|$(BIN_NAME)-plugin-linux-arm|' -czf bin/$(BIN_NAME)-linux-arm.tar.gz -C bin $(BIN_NAME)-linux-arm
	tar --transform='flags=r;s|$(BIN_NAME)-darwin|$(BIN_NAME)-plugin-darwin|' -czf bin/$(BIN_NAME)-darwin.tar.gz -C bin $(BIN_NAME)-darwin
	tar czf bin/$(BIN_NAME)-e2e-darwin.tar.gz -C bin $(BIN_NAME)-e2e-darwin
	tar --transform='flags=r;s|$(BIN_NAME)-windows|$(BIN_NAME)-plugin-windows|' -czf bin/$(BIN_NAME)-windows.tar.gz -C bin $(BIN_NAME)-windows.exe
	tar czf bin/$(BIN_NAME)-e2e-windows.tar.gz -C bin $(BIN_NAME)-e2e-windows.exe

test: test-unit test-e2e ## run all tests

test-unit: build_dev_image ## run unit tests
	@$(call mkdir,_build/test-results)
	docker run --rm -v $(CURDIR)/_build/test-results:/test-results $(DEV_IMAGE_NAME) make TEST_RESULTS_PREFIX=$(TEST_RESULTS_PREFIX) test-unit

test-e2e: build_dev_image invocation-image ## run end-to-end tests
	docker run -v /var/run:/var/run:ro --rm --network="host" $(DEV_IMAGE_NAME) make TEST_RESULTS_PREFIX=$(TEST_RESULTS_PREFIX) bin/$(BIN_NAME) E2E_TESTS=$(E2E_TESTS) test-e2e

COV_LABEL := com.docker.app.cov-run=$(TAG)
coverage-run: build_dev_image ## run tests with coverage
	@$(call mkdir,_build)
	docker run -v /var/run:/var/run:ro --name $(COV_CTNR_NAME) --network="host" -t $(DEV_IMAGE_NAME) make COMMIT=${COMMIT} TAG=${TAG} TEST_RESULTS_PREFIX=$(TEST_RESULTS_PREFIX) coverage
coverage-results:
	docker cp $(COV_CTNR_NAME):$(PKG_PATH)/_build/cov/ ./_build/ci-cov
	docker cp $(COV_CTNR_NAME):$(PKG_PATH)/_build/test-results/ ./_build/test-results
	docker rm $(COV_CTNR_NAME)
# coverage is split in two like this so that CI can extract the results even on failure (which will be detected via the junit) using the individual steps, but for end users running we want the overall failure.
coverage: coverage-run coverage-results

lint: ## run linter(s)
	$(info Linting...)
	docker build -t $(LINT_IMAGE_NAME) -f Dockerfile.lint .
	docker run --rm $(LINT_IMAGE_NAME) make lint

vendor: build_dev_image
	$(info Update Vendoring...)
	docker rm -f docker-app-vendoring || true
	# git bash, mingw and msys by default rewrite args that seems to be linux paths and try to expand that to a meaningful windows path
	# we don't want that to happen when mounting paths referring to files located in the container. Thus we use the double "//" prefix that works
	# both on windows, linux and macos
	docker run -it --name docker-app-vendoring -v docker-app-vendor-cache://dep-cache -e DEPCACHEDIR=//dep-cache $(DEV_IMAGE_NAME) sh -c "rm -rf ./vendor && make vendor DEP_ARGS=\"$(DEP_ARGS)\""
	rm -rf ./vendor
	docker cp docker-app-vendoring:/go/src/github.com/docker/app/vendor .
	docker cp docker-app-vendoring:/go/src/github.com/docker/app/Gopkg.lock .
	docker rm -f docker-app-vendoring
	$(warning You may need to reset permissions on vendor/*)

clean-vendor-cache:
	docker rm -f docker-app-vendoring || true
	docker volume rm -f docker-app-vendor-cache

check-vendor: build_dev_image
	$(info Check Vendoring...)
	docker run --rm $(DEV_IMAGE_NAME) sh -c "make vendor && hack/check-git-diff vendor"

specification/bindata.go: specification/schemas/*.json build_dev_image
	docker run --name $(SCHEMAS_CTNR_NAME) $(DEV_IMAGE_NAME) sh -c "make schemas"
	docker cp $(SCHEMAS_CTNR_NAME):$(PKG_PATH)/specification/bindata.go ./specification/
	docker rm $(SCHEMAS_CTNR_NAME)

schemas: specification/bindata.go ## generate specification/bindata.go from json schemas

invocation-image:
	docker build -f Dockerfile.invocation-image $(BUILD_ARGS) --target=invocation -t $(CNAB_BASE_INVOCATION_IMAGE_NAME) -t $(CNAB_BASE_INVOCATION_IMAGE_NAME)-amd64 --platform=amd64 .

invocation-image-arm64:
	docker build -f Dockerfile.invocation-image $(BUILD_ARGS) --target=invocation -t $(CNAB_BASE_INVOCATION_IMAGE_NAME)-arm64 --platform=arm64 .

invocation-image-arm:
	docker build -f Dockerfile.invocation-image $(BUILD_ARGS) --target=invocation -t $(CNAB_BASE_INVOCATION_IMAGE_NAME)-arm --platform=arm .

invocation-image-cross: invocation-image invocation-image-arm64 invocation-image-arm

save-invocation-image-tag:
	docker tag $(CNAB_BASE_INVOCATION_IMAGE_NAME) docker/cnab-app-base:$(INVOCATION_IMAGE_TAG)
	docker save docker/cnab-app-base:$(INVOCATION_IMAGE_TAG) -o _build/$(OUTPUT)

save-invocation-image:
	@$(call mkdir,_build)
	docker save $(CNAB_BASE_INVOCATION_IMAGE_NAME) -o $(CNAB_BASE_INVOCATION_IMAGE_PATH).tar

save-invocation-image-cross: save-invocation-image
	docker save $(CNAB_BASE_INVOCATION_IMAGE_NAME)-arm64 -o $(CNAB_BASE_INVOCATION_IMAGE_PATH)-arm64.tar
	docker save $(CNAB_BASE_INVOCATION_IMAGE_NAME)-arm -o $(CNAB_BASE_INVOCATION_IMAGE_PATH)-arm.tar

push-invocation-image:
	# tag and push linux/amd64
	docker tag $(CNAB_BASE_INVOCATION_IMAGE_NAME) $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME)
	docker push $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME)
	# tag and push linux/arm64
	docker tag $(CNAB_BASE_INVOCATION_IMAGE_NAME)-arm64 $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME)-arm64
	docker push $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME)-arm64
	# tag and push linux/armhf
	docker tag $(CNAB_BASE_INVOCATION_IMAGE_NAME)-arm $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME)-arm
	docker push $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME)-arm
	# create and push manifest list
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest create $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME) $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME) $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME)-arm64 $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME)-arm
	DOCKER_CLI_EXPERIMENTAL=enabled docker manifest push $(PUSH_CNAB_BASE_INVOCATION_IMAGE_NAME)

help: ## this help
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

.PHONY: lint test-e2e test-unit test cli-cross cross e2e-cross coverage coverage-run coverage-results shell build_dev_image tars vendor check-vendor schemas help invocation-image invocation-image-arm invocation-image-arm64 invocation-image-cross save-invocation-image save-invocation-image-tag push-invocation-image
