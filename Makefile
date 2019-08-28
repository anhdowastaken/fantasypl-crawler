SHELL := /bin/bash

ARCH := "$(shell uname -m)"
PLATFORM := $(shell go env GOOS)
GOCMD := go
GOPATH := $(shell go env GOPATH)
GOBIN := $(GOPATH)/bin
GOBUILD := $(GOCMD) build
GORUN := $(GOCMD) run
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOLINT := $(GOCMD) run $(GOPATH)/src/golang.org/x/lint/golint/
WORKINGSPACE := $(shell pwd)
WORKINGSPACEBIN := $(WORKINGSPACE)/bin
BINARYNAME := fantasypl-crawler

# Prepare environment variables
export GOROOT:=$(GOROOT)
export GOPATH:=$(GOPATH)

all: help

gomod:
	cd $(WORKINGSPACE); \
	$(GOCMD) mod tidy; \
	$(GOCMD) mod vendor;

clean:
	# Clean output of build
	rm -rf $(WORKINGSPACEBIN)/static; \
	rm -rf $(WORKINGSPACEBIN)/template; \
	rm -rf $(WORKINGSPACEBIN)/$(BINARYNAME).conf; \
	rm -rf $(WORKINGSPACEBIN)/$(BINARYNAME);

prepare:
	if [[ ! -d "$(WORKINGSPACEBIN)/" ]]; then \
		mkdir -p "$(WORKINGSPACEBIN)/"; \
	fi

build: clean gomod prepare
	cd $(WORKINGSPACE); \
	$(GOBUILD) -ldflags="-s -w" -mod vendor -o $(WORKINGSPACEBIN)/$(BINARYNAME); \
	# TODO: Consider to use upx here to reduce binary size

	if [ $$? -eq 0 ]; then \
		cp $(WORKINGSPACE)/$(BINARYNAME).conf $(WORKINGSPACEBIN)/$(BINARYNAME).conf; \
	fi

run: build
	cd $(WORKINGSPACEBIN); \
	./$(BINARYNAME)

help:
	@echo "make <option>"
	@echo "- all | help: Show help"
	@echo "- gomod: Prepare Go modules"
	@echo "- build: Build product"
	@echo "- run: Execute build then run immediately"
	@echo "- clean: Clean all outputs"
