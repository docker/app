PKG_NAME := github.com/docker/app
BIN_NAME := docker-app
E2E_NAME := $(BIN_NAME)-e2e

# Enable experimental features. "on" or "off"
EXPERIMENTAL := off

# Comma-separated list of renderers
RENDERERS := "none"

TAG ?= $(shell git describe --always --dirty 2>/dev/null)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null)
CWD = $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

# Used by ci-gradle-test target
DOCKERAPP_BINARY ?= $(CWD)/bin/$(BIN_NAME)-linux

ifeq ($(BUILDTIME),)
  BUILDTIME := ${shell date --utc --rfc-3339 ns 2> /dev/null | sed -e 's/ /T/'}
endif
ifeq ($(BUILDTIME),)
  BUILDTIME := ${shell gdate --utc --rfc-3339 ns 2> /dev/null | sed -e 's/ /T/'}
endif
ifeq ($(BUILDTIME),)
  $(error unable to set BUILDTIME, ensure that you have GNU date installed or set manually)
endif

LDFLAGS := "-s -w \
	-X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
	-X $(PKG_NAME)/internal.Version=$(TAG)      \
	-X $(PKG_NAME)/internal.Experimental=$(EXPERIMENTAL) \
	-X $(PKG_NAME)/internal.Renderers=$(RENDERERS) \
	-X $(PKG_NAME)/internal.BuildTime=$(BUILDTIME)"

EXEC_EXT :=
ifeq ($(OS),Windows_NT)
    EXEC_EXT := .exe
endif
