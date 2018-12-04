# Building `docker-app` from source

This guide is useful if you intend to contribute on `docker/app`. Thanks for your
effort. Every contribution is very appreciated.

This doc includes:
* [Build requirements](#build-requirements)
* [Using Go](#build-using-go)
* [Using Docker](#build-using-docker)
* [Testing](#testing-docker-app)

## Build requirements

To build the `docker-app`, at least one of the following build system
dependencies are required:

* Docker (17.12 or above)
* Go (1.10.x or above)

You will also need the following tools:

* GNU Make
* [`dep`](https://github.com/golang/dep)

## Build using Go

First you need to setup your Go development environment. You can follow this
guideline [How to write go code](https://golang.org/doc/code.html) and at the
end you need to have `GOPATH` set in your environment.

At this point you can use `go` to checkout `docker-app` in your `GOPATH`:

```sh
go get github.com/docker/app
```

You are ready to build `docker-app` yourself!

`docker-app` uses `make` to create a repeatable build flow. It means that you
can run:

```sh
make
```

This is going to build all the project binaries in the `./bin/`
directory, run tests (unit and end-to-end).

```sh
make bin/docker-app             # builds the docker-app binary
make bin/docker-app-darwin      # builds the docker-app binary for darwin
make bin/docker-app-windows.exe # builds the docker-app binary for windows

make lint                       # run the linter on the sources
make test-unit                  # run the unit tests
make test-e2e                   # run the end-to-end tests
```

Vendoring of external imports uses the [`dep`](https://github.com/golang/dep) tool.
Please refer to its documentation if you need to update a dependency.

## Build using Docker

If you don't have Go installed but Docker is present, you can also use
`docker.Makefile` to build `docker-app` and run tests. This
`docker.Makefile` is used by our continuous integration too.

```sh
make -f docker.Makefile           # builds cross binaries build and tests
make -f docker.Makefile cross     # builds cross binaries (linux, darwin, windows)
make -f docker.Makefile schemas   # update the embedded schemas

make -f docker.Makefile lint      # run the linter on the sources
make -f docker.Makefile test-unit # run the unit tests
make -f docker.Makefile test-e2e  # run the end-to-end tests
```

## Testing docker-app

During the automated CI, the unit tests and end-to-end tests are run as
part of the PR validation. As a developer you can run these tests
locally by using any of the following `Makefile` targets:

- `make test`: run all non-end-to-end tests
- `make test-e2e`: run all end-to-end tests

To execute a specific test or set of tests you can use the `go test`
capabilities without using the `Makefile` targets. The following
examples show how to specify a test name and also how to use the flag
directly against `go test` to run root-requiring tests.

```sh
# run the test <TEST_NAME>:
go test -v -run "<TEST_NAME>" .
```
