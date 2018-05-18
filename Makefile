include golang.mk
.DEFAULT_GOAL := test # override default goal set in library makefile

SHELL := /bin/bash
PKG = github.com/Clever/go-bench
PKGS := $(shell go list ./... | grep -v /vendor)
EXECUTABLE := $(shell basename $(PKG))
.PHONY: test vendor $(PKGS)

$(eval $(call golang-version-check,1.10))

all: test build

build:
	go build -o bin/$(EXECUTABLE) $(PKG)

test: $(PKGS)

$(PKGS): golang-test-all-deps
	$(call golang-test-all,$@)



install_deps: golang-dep-vendor-deps
	$(call golang-dep-vendor)