PROJECT_NAME := keepalived-exporter
PKG := "github.com/cafebazaar/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
LINTER = golangci-lint
LINTER_VERSION = v1.24.0

.PHONY: all dep lint build clean

all: build

dep: ## Get the dependencies
	@go mod tidy

lintdeps: ## golangci-lint dependencies
	curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(GOPATH)/bin $(LINTER_VERSION)

lint: lintdeps ## to lint the files
	$(LINTER) run --config=.golangci-lint.yml ./...

build: dep ## Build the binary file
	@go build -i -v $(PKG)/cmd/$(PROJECT_NAME)

clean: ## Remove previous build
	@rm -f $(PROJECT_NAME)
