SHELL := /usr/bin/env bash

BINFILE=build/_output/bin/$(OPERATOR_NAME)
MAINPACKAGE=./cmd/manager
GOENV=GOOS=linux GOARCH=amd64 CGO_ENABLED=0
GOFLAGS=-gcflags="all=-trimpath=${GOPATH}" -asmflags="all=-trimpath=${GOPATH}"

TESTTARGETS := $(shell go list -e ./... | egrep -v "/(vendor)/")
# ex, -v
TESTOPTS :=

include project.mk

default: generate build

.PHONY: clean
clean:
	rm -rf build/_output/bin/

.PHONY: test
test:
	go test $(TESTOPTS) $(TESTTARGETS)

.PHONY: generate
generate:
	go generate pkg/gcpclient/client.go

.PHONY: build
build: clean ## Build binary
	${GOENV} go build ${GOFLAGS} -o ${BINFILE} ${MAINPACKAGE}

.PHONY: clean
clean:
	rm -rf ${BINFILE}

run:
	go run cmd/manager/main.go

image:
	buildah build-using-dockerfile --network=host -f build/Dockerfile -t quay.io/${USER}/gcp-project-operator
