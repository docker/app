include vars.mk

GO_BUILD := CGO_ENABLED=0 go build
GO_TEST := CGO_ENABLED=0 go test

all: bin/$(BIN_NAME) test

check_go_env:
	@test $$(go list) = "$(PKG_NAME)" || \
		(echo "Invalid Go environment" && false)

bin/%: cmd/% check_go_env
	$(GO_BUILD) -ldflags=$(LDFLAGS) -o $@$(EXEC_EXT) ./$<

.PHONY: bin/$(BIN_NAME)-windows
bin/$(BIN_NAME)-windows:: bin/$(BIN_NAME)-windows.exe

bin/$(BIN_NAME)-% bin/$(BIN_NAME)-%.exe: cmd/$(BIN_NAME) check_go_env
	GOOS=$* $(GO_BUILD) -ldflags=$(LDFLAGS) -o $@ ./$<

cross: bin/$(BIN_NAME)-linux bin/$(BIN_NAME)-darwin bin/$(BIN_NAME)-windows.exe

.PHONY: bin/$(BIN_NAME)-e2e-windows
bin/$(BIN_NAME)-e2e-windows:: bin/$(BIN_NAME)-e2e-windows.exe

bin/$(BIN_NAME)-e2e-% bin/$(BIN_NAME)-e2e-%.exe: e2e bin/$(BIN_NAME)-%
	GOOS=$* $(GO_TEST) -c -o $@ ./$<

e2e-cross: bin/$(BIN_NAME)-e2e-linux bin/$(BIN_NAME)-e2e-darwin bin/$(BIN_NAME)-e2e-windows.exe

check: lint test

test: test-unit test-e2e

lint:
	@echo "Linting..."
	@gometalinter --config=gometalinter.json

test-e2e: bin/$(BIN_NAME)
	@echo "Running e2e tests..."
	$(GO_TEST) ./e2e/

test-unit:
	@echo "Running unit tests..."
	$(GO_TEST) $(shell go list ./... | grep -vE '/e2e')

coverage-bin:
	$(GO_TEST) -coverpkg="./..." -c -ldflags=$(LDFLAGS) -tags testrunmain -o _build/$(BIN_NAME).cov ./cmd/docker-app

coverage-test-unit:
	@echo "Running unit tests (coverage)..."
	mkdir -p _build/cov
	$(GO_TEST) -cover -test.coverprofile=_build/cov/unit.out $(shell go list ./... | grep -vE '/e2e')

coverage-test-e2e: coverage-bin
	@echo "Running e2e tests (coverage)..."
	mkdir -p _build/cov
	DOCKERAPP_BINARY=../e2e/coverage-bin $(GO_TEST) -v ./e2e

coverage: coverage-test-unit coverage-test-e2e
	go install ./vendor/github.com/wadey/gocovmerge/
	gocovmerge _build/cov/*.out > _build/cov/all.out
	go tool cover -func _build/cov/all.out
	go tool cover -html _build/cov/all.out -o _build/cov/coverage.html

clean:
	rm -Rf ./bin ./_build docker-app-*.tar.gz

.PHONY: cross e2e-cross test check lint test-unit test-e2e coverage coverage-bin coverage-test-unit coverage-test-e2e clean
.DEFAULT: all
