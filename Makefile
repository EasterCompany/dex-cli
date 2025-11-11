# Makefile for dex-cli

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOLINT=golangci-lint

# Build information
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.0.0")
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_YEAR := $(shell date -u +"%Y")
BUILD_HASH := $(shell cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 6 | head -n 1)
EXECUTABLE_PATH := $(HOME)/Dexter/bin/dex

# Linker flags
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.branch=$(BRANCH) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE) -X main.buildYear=$(BUILD_YEAR) -X main.buildHash=$(BUILD_HASH)"

.PHONY: all build clean test lint format check deps

all: build

# New target to ensure modules are downloaded and go.sum is correct
deps:
	@echo "Ensuring Go modules are tidy..."
	@$(GOCMD) mod tidy

# Check now depends on 'deps'
check: deps format lint test

format:
	@echo "Formatting..."
	@$(GOFMT) ./...

lint:
	@echo "Linting..."
	@$(GOLINT) run

test:
	@echo "Testing..."
	@$(GOTEST) -v ./...

# Build depends on check (which now depends on deps)
build: check
	@echo "Building..."
	@$(GOBUILD) $(LDFLAGS) -o $(EXECUTABLE_PATH) .

build-for-release: check
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION argument is required for build-for-release"; \
		exit 1; \
	fi
	@echo "Building for release with version $(VERSION)..."
	@$(GOBUILD) -ldflags "-X main.version=$(VERSION) -X main.branch=$(BRANCH) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE) -X main.buildYear=$(BUILD_YEAR) -X main.buildHash=$(BUILD_HASH)" -o $(EXECUTABLE_PATH) .

install: build

clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -f $(EXECUTABLE_PATH)

echo-version:
	@echo $(VERSION)
