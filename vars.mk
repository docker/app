PKG_NAME := github.com/docker/app
BIN_NAME := docker-app
E2E_NAME := $(BIN_NAME)-e2e

# Enable experimental features. "on" or "off"
EXPERIMENTAL := off

# Comma-separated list of renderers
RENDERERS := "none"

TAG ?= $(shell git describe --always --dirty 2>/dev/null)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null)

ifeq ($(OS),Windows_NT)
  PLATFORM := windows
  CHMOD =
  EXEC_EXT := .exe
else
  PLATFORM := $(shell uname -s | tr '[:upper:]' '[:lower:]')
  CHMOD = chmod
  EXEC_EXT :=
endif
STUPID := ./bin/stupid-$(PLATFORM)$(EXEC_EXT)

BUILDTIME := $(shell $(STUPID) date)

LDFLAGS := "-s -w \
	-X $(PKG_NAME)/internal.GitCommit=$(COMMIT) \
	-X $(PKG_NAME)/internal.Version=$(TAG)      \
	-X $(PKG_NAME)/internal.Experimental=$(EXPERIMENTAL) \
	-X $(PKG_NAME)/internal.Renderers=$(RENDERERS) \
	-X $(PKG_NAME)/internal.BuildTime=$(BUILDTIME)"

ifeq ($(WINDOWS),yes)
  mkdir = mkdir $(subst /,\,$(1)) > nul 2>&1 || (exit 0)
else
  mkdir = mkdir -p $(1)
endif
