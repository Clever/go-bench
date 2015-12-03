SHELL := /bin/bash
PKG = github.com/Clever/go-bench
PKGS := $(shell go list ./... | grep -v /vendor)
EXECUTABLE := go-bench
.PHONY: test vendor $(PKGS)

GOVERSION := $(shell go version | grep 1.5)
ifeq "$(GOVERSION)" ""
  $(error must be running Go version 1.5)
endif
export GO15VENDOREXPERIMENT = 1

GOLINT := $(GOPATH)/bin/golint
$(GOLINT):
	go get github.com/golang/lint/golint

GODEP := $(GOPATH)/bin/godep
$(GODEP):
	go get -u github.com/tools/godep

build:
	go build -o bin/$(EXECUTABLE) $(PKG)

test: $(PKG)

$(PKG): $(GOLINT)
ifeq ($(LINT),1)
	$(GOLINT) $(GOPATH)/src/$@*/**.go
endif
	go get -d -t $@
ifeq ($(COVERAGE),1)
	go test -cover -coverprofile=$(GOPATH)/src/$@/c.out $@ -test.v
	go tool cover -html=$(GOPATH)/src/$@/c.out
else
	go test $@ -test.v
endif

vendor: $(GODEP)
	$(GODEP) save $(PKGS)
	find vendor/ -path '*/vendor' -type d | xargs -IX rm -r X # remove any nested vendor directories
