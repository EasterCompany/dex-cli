# Makefile for dex-cli

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOLINT=golangci-lint

# Get the version from the command line, default to 0.0.0
version ?= 0.0.0
v_arg = $(filter v=%,$(.VARIABLES))
ifneq ($(v_arg),)
    version = $(patsubst v=%,%,$(v_arg))
endif

# Build information
BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
BUILD_YEAR := $(shell date -u +"%Y")
EXECUTABLE_PATH := $(HOME)/Dexter/bin/dex

# Linker flags
LDFLAGS = -ldflags "-X main.version=$(version) -X main.branch=$(BRANCH) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD_DATE) -X main.buildYear=$(BUILD_YEAR)"

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

install: build

clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -f $(EXECUTABLE_PATH)
