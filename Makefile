# Makefile for dex-cli

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOLINT=golangci-lint

# Build information
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0-dev")
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_YEAR := $(shell date -u +"%Y")
EXECUTABLE_PATH := $(HOME)/Dexter/bin/dex

# Linker flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.branch=$(BRANCH) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE) -X main.buildYear=$(BUILD_YEAR)"

.PHONY: all build clean test lint format check

all: build

check: format lint test

format:
	@echo "Formatting..."
	@$(GOFMT) ./...

lint:
	@echo "Linting..."
	@$(GOLINT) run

test:
	@echo "Testing..."
	@$(GOTEST) -v ./...

build: check
	@echo "Building..."
	@$(GOBUILD) $(LDFLAGS) -o $(EXECUTABLE_PATH) .

build-for-release: check
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION argument is required for build-for-release"; \
		exit 1; \
	fi
	@echo "Building for release with version $(VERSION)..."
	@$(GOBUILD) -ldflags "-X main.version=$(VERSION) -X main.branch=$(BRANCH) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE) -X main.buildYear=$(BUILD_YEAR)" -o $(EXECUTABLE_PATH) .

install: build

clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -f $(EXECUTABLE_PATH)
