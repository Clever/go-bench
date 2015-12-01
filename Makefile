SHELL := /bin/bash
PKG = github.com/Clever/go-bench
SUBPKGS = slowhttp
PKGS = $(PKG) $(SUBPKGS)

.PHONY: test $(PKGS)

GOVERSION := $(shell go version | grep 1.5)
ifeq "$(GOVERSION)" ""
  $(error must be running Go version 1.5)
endif

export GO15VENDOREXPERIMENT = 1

test: $(PKG)

$(PKG):
ifeq ($(LINT),1)
	golint $(GOPATH)/src/$@*/**.go
endif
	go get -d -t $@
ifeq ($(COVERAGE),1)
	go test -cover -coverprofile=$(GOPATH)/src/$@/c.out $@ -test.v
	go tool cover -html=$(GOPATH)/src/$@/c.out
else
	go test $@ -test.v
endif


SHELL := /bin/bash
PKGS := $(shell go list ./... | grep -v /vendor)
GODEP := $(GOPATH)/bin/godep

$(GODEP):
	go get -u github.com/tools/godep

vendor: $(GODEP)
	$(GODEP) save $(PKGS)
	find vendor/ -path '*/vendor' -type d | xargs -IX rm -r X # remove any nested vendor directories
