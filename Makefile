# Directory containing the Makefile.
PROJECT_ROOT = $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

export PATH := $(GOBIN):$(PATH)

BENCH_FLAGS ?= -cpuprofile=cpu.pprof -memprofile=mem.pprof -benchmem

# Directories containing independent Go modules.
MODULE_DIRS = . pkg/identifier

GO_INSTALLABLE_DIRS = cmd/cti

# Directories that we want to test and track coverage for.
TEST_DIRS = . ./metadata

.PHONY: all
all: lint cover

.PHONY: lint
lint: 
	@$(foreach mod,$(MODULE_DIRS), \
		(cd $(mod) && \
		echo "[lint] golangci-lint: $(mod)" && \
		golangci-lint run --path-prefix $(mod)) &&) true

.PHONY: tidy
tidy:
	@$(foreach dir,$(MODULE_DIRS), \
		(cd $(dir) && go mod tidy) &&) true

.PHONY: test
test:
	@$(foreach dir,$(TEST_DIRS),(cd $(dir) && go test -race ./...) &&) true

.PHONY: cover
cover:
	@$(foreach dir,$(TEST_DIRS), ( \
		cd $(dir) && \
		go test -race -coverprofile=cover.out -coverpkg=./... ./... \
		&& go tool cover -html=cover.out -o cover.html) &&) true

.PHONY: install
install: go-install

.PHONY: go-install
go-install:
	@$(foreach dir,$(GO_INSTALLABLE_DIRS), ( \
	    cd $(dir) && \
		go install -v ./... \
		&& echo `go list -f '{{.Module.Path}}'` has been installed to `go list -f '{{.Target}}'`) &&) true
