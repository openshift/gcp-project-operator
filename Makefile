include boilerplate/generated-includes.mk

SHELL := /usr/bin/env bash

OPERATOR_DOCKERFILE = ./build/Dockerfile

default:

# Extend Makefile after here

.PHONY: docker-build
docker-build: build

.PHONY: boilerplate-update
boilerplate-update:
	@boilerplate/update
