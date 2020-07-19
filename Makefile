PROJECT_NAME := keepalived-exporter
PKG := "github.com/cafebazaar/$(PROJECT_NAME)"
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)
GO_FILES := $(shell find . -name '*.go' | grep -v /vendor/ | grep -v _test.go)
LINTER = golangci-lint
LINTER_VERSION = v1.28.3
COMMIT := $(shell git rev-parse HEAD)
VERSION := $(shell git describe --tags ${COMMIT} | cut -c2-)
ARCH := $(shell dpkg --print-architecture)
RELEASE_FILENAME := $(PROJECT_NAME)-$(VERSION).linux-$(ARCH)

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

test:
	@go test -v -cover -race ./...

clean: ## Remove previous build and release files
	@rm -f $(PROJECT_NAME)
	@rm -f $(RELEASE_FILENAME).zip
	@rm -f $(RELEASE_FILENAME).tar.gz

release: build
	@mkdir $(RELEASE_FILENAME)
	@cp $(PROJECT_NAME) $(RELEASE_FILENAME)/
	@cp LICENSE $(RELEASE_FILENAME)/
	@zip -r $(RELEASE_FILENAME).zip $(RELEASE_FILENAME)
	@tar -czvf $(RELEASE_FILENAME).tar.gz $(RELEASE_FILENAME)
	@rm -rf $(RELEASE_FILENAME)
