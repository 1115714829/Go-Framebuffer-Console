BINARY_NAME=framebuffer-console
GO_FILES=$(shell find . -type f -name '*.go')

.PHONY: all build clean run install deps

all: build

build: deps
	CGO_ENABLED=1 GOOS=linux go build -o $(BINARY_NAME) ./cmd/main

build-static: deps
	CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o $(BINARY_NAME) ./cmd/main

deps:
	go mod tidy
	go mod download

clean:
	rm -f $(BINARY_NAME)

run: build
	sudo ./$(BINARY_NAME)

install: build
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)

test:
	go test -v ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

dev: fmt vet build

.DEFAULT_GOAL := all