#!/usr/bin/env make

VERSION=$(shell git show-ref --head --hash head)

GO_LDFLAGS=-ldflags "-X `go list ./version`.Version $(VERSION)"

.DEFAULT: build
.PHONY: build

all: build

deps:
	go get -t ./...

build: deps
	go build -o overlord ${GO_LDFLAGS} ./cmd/main.go

build_linux: deps
	env GOOS=linux go build -o overlord ${GO_LDFLAGS} ./cmd/main.go
