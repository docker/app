include vars.mk

IMAGE_NAME := docker-app

IMAGE_BUILD_ARGS := \
    --build-arg COMMIT=$(COMMIT) \
    --build-arg TAG=$(TAG)       \
    --build-arg BUILDTIME=$(BUILDTIME)

PKG_PATH := /go/src/$(PKG_NAME)

.DEFAULT: all
all: cross test

create_bin:
	@$(call mkdir,bin)

build_dev_image:
	docker build --target=dev -t $(IMAGE_NAME)-dev $(IMAGE_BUILD_ARGS) .

shell: build_dev_image
	docker run -ti --rm $(IMAGE_NAME)-dev bash

bin/$(BIN_NAME)-linux: create_bin
	docker build --target=$(BIN_NAME) -t $(IMAGE_NAME)-bin $(IMAGE_BUILD_ARGS) .
	$(eval containerID=$(shell docker create $(IMAGE_NAME)-bin noop))
	docker cp $(containerID):$(PKG_PATH)/bin/$(BIN_NAME) $@
	docker rm $(containerID)
	@chmod +x $@

cross: create_bin
	docker build --target=$* -t $(IMAGE_NAME)-cross $(IMAGE_BUILD_ARGS) .
	$(eval containerID=$(shell docker create $(IMAGE_NAME)-cross noop))
	docker cp $(containerID):$(PKG_PATH)/bin/$(BIN_NAME)-linux bin/$(BIN_NAME)-linux
	docker cp $(containerID):$(PKG_PATH)/bin/$(BIN_NAME)-darwin bin/$(BIN_NAME)-darwin
	docker cp $(containerID):$(PKG_PATH)/bin/$(BIN_NAME)-windows.exe bin/$(BIN_NAME)-windows.exe
	docker rm $(containerID)
	@$(call chmod,+x,bin/$(BIN_NAME)-linux)
	@$(call chmod,+x,bin/$(BIN_NAME)-darwin)
	@$(call chmod,+x,bin/$(BIN_NAME)-windows.exe)

e2e-cross: create_bin
	docker build --target=e2e-cross -t $(IMAGE_NAME)-e2e-cross $(IMAGE_BUILD_ARGS) .
	$(eval containerID=$(shell docker create $(IMAGE_NAME)-e2e-cross noop))
	docker cp $(containerID):$(PKG_PATH)/bin/$(BIN_NAME)-e2e-linux bin/$(BIN_NAME)-e2e-linux
	docker cp $(containerID):$(PKG_PATH)/bin/$(BIN_NAME)-e2e-darwin bin/$(BIN_NAME)-e2e-darwin
	docker cp $(containerID):$(PKG_PATH)/bin/$(BIN_NAME)-e2e-windows.exe bin/$(BIN_NAME)-e2e-windows.exe
	docker rm $(containerID)
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

test: test-unit test-e2e

test-unit: build_dev_image
	docker run --rm $(IMAGE_NAME)-dev make COMMIT=${COMMIT} TAG=${TAG} BUILDTIME=${BUILDTIME} test-unit

test-e2e: build_dev_image
	docker run -v /var/run:/var/run:ro --rm $(IMAGE_NAME)-dev make COMMIT=${COMMIT} TAG=${TAG} BUILDTIME=${BUILDTIME} bin/$(BIN_NAME) test-e2e

COV_LABEL := com.docker.app.cov-run=$(TAG)
coverage: build_dev_image
	@$(call mkdir,_build)
	$(eval containerID=$(shell docker run -v /var/run:/var/run:ro -tid $(IMAGE_NAME)-dev make COMMIT=${COMMIT} TAG=${TAG} BUILDTIME=${BUILDTIME} coverage))
	docker logs -f $(containerID)
	docker cp $(containerID):$(PKG_PATH)/_build/cov/ ./_build/ci-cov
	docker rm $(containerID)

gradle-test: bin/$(BIN_NAME)-linux
	docker build -t $(IMAGE_NAME)-bin -f Dockerfile.gradle .
	docker run --rm $(IMAGE_NAME)-bin bash -c "./gradlew --stacktrace build && cd example && gradle renderIt"

lint:
	@echo "Linting..."
	docker build -t $(IMAGE_NAME)-lint:$(TAG) $(IMAGE_BUILD_ARGS) -f Dockerfile.lint .
	docker run --rm $(IMAGE_NAME)-lint:$(TAG) make lint

.PHONY: lint test-e2e test-unit test cross e2e-cross coverage gradle-test shell build_dev_image tars
