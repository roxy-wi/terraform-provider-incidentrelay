BINARY_NAME ?= terraform-provider-incidentrelay
VERSION ?= 0.1.0
HOSTNAME ?= registry.terraform.io
NAMESPACE ?= roxy-wi
TYPE ?= incidentrelay
OS_ARCH := $(shell go env GOOS)_$(shell go env GOARCH)
INSTALL_DIR := $(HOME)/.terraform.d/plugins/$(HOSTNAME)/$(NAMESPACE)/$(TYPE)/$(VERSION)/$(OS_ARCH)

.PHONY: build test test-race fmt fmt-check tidy install-local clean

build:
	go build -o bin/$(BINARY_NAME) .

test:
	go test ./...

test-race:
	go test -race ./...

fmt:
	gofmt -w .

fmt-check:
	test -z "$$(gofmt -l .)"

tidy:
	go mod tidy

install-local: build
	mkdir -p $(INSTALL_DIR)
	cp bin/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)_v$(VERSION)

clean:
	rm -rf bin
