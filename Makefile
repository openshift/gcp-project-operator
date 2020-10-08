SHELL := /usr/bin/env bash

OPERATOR_DOCKERFILE = ./build/Dockerfile

# Include shared Makefiles
include project.mk
include standard.mk

default: gobuild

# Extend Makefile after here

.PHONY: docker-build
docker-build: build

generate:
	go get github.com/golang/mock/mockgen
	go get golang.org/x/tools/cmd/goimports
	go generate ${TESTTARGETS}
coverage:
	go get github.com/jpoles1/gopherbadger
	gopherbadger
