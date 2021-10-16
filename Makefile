#
# DSQL - Data Structured Query Language Engine
#
BINARY         = dsql
PACKAGE        = github.com/patrickglass/dsql
VER_PREFIX     = $(PACKAGE)/version
DOCKER_IMAGE   ?= ghcr.io/patrickglass/$(BINARY):$(DOCKER_TAG)
DOCKER_PREFIX  ?= ghcr.io/patrickglass/
DOCKER_TAG     ?= latest
DOCKER_PUSH    ?= false
DEVEL_ENV      = DEBUG=true
GIT_TAG        = $(shell if [ -z "`git status --porcelain`" ]; then git describe --exact-match --tags HEAD 2>/dev/null; fi)
GIT_TREE_STATE = $(shell if [ -z "`git status --porcelain`" ]; then echo "clean" ; else echo "dirty"; fi)
VERSIONREL     = $(shell if [ -z "`git status --porcelain`" ]; then git rev-parse --short HEAD 2>/dev/null ; else echo "dirty"; fi)
PKGS           = $(shell go list ./... | grep -v /vendor)
BIN_DIR        = $(shell go env GOPATH)/bin
GOLANGCI-LINT  = $(BIN_DIR)/golangci-lint
GORELEASER     = $(BIN_DIR)/goreleaser
GOLICENSES     = $(BIN_DIR)/go-licenses

ifneq (${GIT_TAG},)
override DOCKER_TAG = ${GIT_TAG}
override VERSIONREL = ${GIT_TAG}
endif


.PHONY: help
help:  ## Show this help.
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {sub("\\\\n",sprintf("\n%22c"," "), $$2);printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: all
all: release  ## Build binary with production settings.

.PHONY: fmt
fmt:  ## Format the go code.
	go fmt $(PKGS)

.PHONY: lint
lint: $(GOLANGCI-LINT) $(GOLICENSES)  ## Lint the go code to ensure code sanity.
	$(GOLANGCI-LINT) run
	$(GOLICENSES) check .

.PHONY: test
test:  ## Run the go unit tests.
	@echo "INFO: Running all go unit tests."
	go test $(PKGS)

.PHONY: coverage
coverage:  ## Report the unit test code coverage.
	@echo "INFO: Generating unit test coverage report."
	go test $(PKGS) -coverprofile=coverage.out
	go tool cover -func=coverage.out

.PHONY: bench
bench:  ## Run the go benchmark tests.
	@echo "INFO: Running all go benchmark tests, Skipping all unit tests."
	go test -run=XXX -bench=. $(PKGS)

.PHONY: run
run:  ## Run the development server.
	@echo "INFO: Starting development instance of dsql engine."
	$(DEVEL_ENV) go run main.go

.PHONY: install
install:
	go install

.PHONY: build
build: $(GORELEASER)  ## Build the multi-arch binaries
	$(GORELEASER) build --single-target

.PHONY: docker-image
docker-image: linux  ## Build the docker image
	@if [ "$(DOCKER_PUSH)" = "true" ] ; then docker push $(DOCKER_PREFIX)$(BINARY):$(DOCKER_TAG) ; fi

.PHONY: precheckin
precheckin: test coverage bench lint $(GORELEASER)  ## Run all the tests and linters which must pass before checking in code
	$(GORELEASER) release --snapshot --rm-dist

.PHONY: release-precheck
release-precheck:
	@if [ "$(GIT_TREE_STATE)" != "clean" ]; then echo 'ERROR: git tree state is $(GIT_TREE_STATE)' ; exit 1; fi
	@if [ -z "$(GIT_TAG)" ]; then echo 'ERROR: commit must be tagged to perform release' ; exit 1; fi

.PHONY: release
release: precheckin release-precheck  ## Run goreleaser to publish binaries

# Tooling and Support Targets
$(GOLANGCI-LINT):
	@echo "INFO: Download golangci-lint binary"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

$(GORELEASER):
	@echo "INFO: Download goreleaser binary"
	go install github.com/goreleaser/goreleaser@latest

$(GOLICENSES):
	@echo "INFO: Download go-licenses binary"
	go install github.com/google/go-licenses@latest

.PHONY: clean
clean:
	rm -rf $(BINARY) dist/ release/ bin/
