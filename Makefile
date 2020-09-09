# Include boilerplate's generated Makefile libraries
include boilerplate/generated-includes.mk

.PHONY: update-boilerplate
update-boilerplate:
	@boilerplate/update

gopherbadger:
	go get github.com/jpoles1/gopherbadger
	gopherbadger
