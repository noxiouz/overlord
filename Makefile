#!/usr/bin/env make

VERSION=$(shell git show-ref --head --hash head)

GO_LDFLAGS=-ldflags "-X `go list ./version`.Version $(VERSION)"

.DEFAULT: build
.PHONY: build

build:
	go build -o overlord ${GO_LDFLAGS} ./cmd/main.go
