SHELL := /usr/bin/env bash

BINFILE=build/_output/bin/$(OPERATOR_NAME)
MAINPACKAGE=./cmd/manager
GOENV=GOOS=linux GOARCH=amd64 CGO_ENABLED=0
GOFLAGS=-gcflags="all=-trimpath=${GOPATH}" -asmflags="all=-trimpath=${GOPATH}"

TESTTARGETS := $(shell go list -e ./... | egrep -v "/(vendor)/")
# ex, -v
TESTOPTS :=

include project.mk

default: gobuild

.PHONY: clean
clean:
	rm -rf ./build/_output

.PHONY: gotest
gotest:
	go test $(TESTOPTS) $(TESTTARGETS)

.PHONY: gobuild
gobuild: gotest ## Build binary
	${GOENV} go build ${GOFLAGS} -o ${BINFILE} ${MAINPACKAGE}