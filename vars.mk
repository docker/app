PKG_NAME := github.com/docker/app
BIN_NAME ?= docker-app
E2E_NAME := $(BIN_NAME)-e2e

# Enable experimental features. "on" or "off"
EXPERIMENTAL := off

# Failing to resolve sh.exe to a full path denotes a windows vanilla shell.
# Although 'simple' commands are still exec'ed, 'complex' ones are batch'ed instead of sh'ed.
ifeq ($(SHELL),sh.exe)
  NULL := nul
  TO_NULL := > $(NULL) 2>&1 || (exit 0)
  mkdir = mkdir $(subst /,\,$(1)) $(TO_NULL)
  rm = del /F /Q $(subst /,\,$(1)) $(TO_NULL)
  rmdir = rmdir /S /Q $(subst /,\,$(1)) $(TO_NULL)
  chmod =
  BUILDTIME ?= unknown
else
  # The no-op redirection forces make to shell out the commands instead of spawning a process as
  # the latter can fail on windows running cmd or powershell while having a unix style shell in the path.
  mkdir = mkdir -p $(1) 1>&1
  rm = rm -rf $(1) 1>&1
  rmdir = rm -rf $(1) 1>&1
  chmod = chmod $(1) $(2) 1>&1
  NULL := /dev/null
  TO_NULL := > $(NULL) 2>&1
endif

ifeq ($(TAG),)
  TAG := $(shell git describe --always --dirty --abbrev=10 2> $(NULL))
endif
ifeq ($(COMMIT),)
  COMMIT := $(shell git rev-parse --short HEAD 2> $(NULL))
endif
BUILD_TAG ?= $(TAG)