# Makefile for goskema

PKG ?= ./...
BENCH_PKG ?= ./benchmarks
BENCH_FILTER ?= .
BENCHTIME ?= 1x
TIMEOUT ?= 10m
COVERFILE ?= coverage.out

# Detect availability of 'go tool covdata' (required for multi-package coverage in newer Go)
HAS_COVDATA := $(shell go tool covdata -h >/dev/null 2>&1 && echo yes || echo no)
TEST_FLAGS := -race -timeout $(TIMEOUT)
ifeq ($(HAS_COVDATA),yes)
TEST_FLAGS += -cover
endif

# compare submodule
COMPARE_DIR := benchmarks/compare
COMPARE_PKG := ./benchmarks/compare

.PHONY: all help tidy fmt fmtcheck vet lint test cover bench bench-cpu bench-huge bench-compare bench-report generate manifests install clean

all: test ## Run tests (default)

help: ## Show help
	@grep -E '^[a-zA-Z0-9_.-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

tidy: ## go mod tidy
	go mod tidy

fmt: ## Format (gofmt -s -w)
	gofmt -s -w .

fmtcheck: ## Check formatting (fail if needed)
	@fmtout=$$(gofmt -s -l .); if [ -n "$$fmtout" ]; then echo "gofmt needed for:"; echo "$$fmtout"; exit 1; fi

vet: ## go vet
	go vet $(PKG)

lint: ## golangci-lint run
	golangci-lint run

test: ## go test with race (and coverage if available)
ifeq ($(HAS_COVDATA),yes)
	go test $(PKG) $(TEST_FLAGS)
else
	@echo "[warn] go tool covdata not found; running tests without -cover"
	go test $(PKG) $(TEST_FLAGS)
endif

cover: ## Generate coverage profile (requires covdata)
ifeq ($(HAS_COVDATA),yes)
	go test $(PKG) -race -covermode=atomic -coverprofile=$(COVERFILE)
else
	@echo "[error] go tool covdata not found; coverage profile not supported on this toolchain" && exit 1
endif

bench: ## Run benchmarks in $(BENCH_PKG) (vars: BENCH_FILTER, BENCHTIME)
	go test -run ^$$ -bench $(BENCH_FILTER) -benchmem -benchtime=$(BENCHTIME) $(BENCH_PKG)

bench-cpu: ## Bench with CPU/mem profiles
	go test -run ^$$ -bench $(BENCH_FILTER) -benchmem -count=1 -cpuprofile cpu.out -memprofile mem.out $(BENCH_PKG)

bench-huge: ## Run huge array benchmarks
	$(MAKE) bench BENCH_FILTER=^Benchmark_ParseFrom_HugeArray

bench-compare: ## Run comparison benchmarks in $(COMPARE_DIR)
	cd $(COMPARE_DIR) && go mod tidy && go test -run ^$$ -bench $(BENCH_FILTER) -benchmem -benchtime=$(BENCHTIME)

bench-report: ## Run benches and generate BENCH_RESULTS.md (vars: BENCH_FILTER, BENCHTIME)
	./scripts/bench_report.sh

generate: ## go generate
	go generate $(PKG)

manifests: ## Generate/apply manifests (noop if not configured)
	@if [ -d config ]; then echo "Building manifests..."; exit 0; else echo "manifests: noop (no ./config)"; fi

install: ## Install manifests into cluster (noop if not configured)
	@if [ -d config ]; then echo "Installing manifests..."; exit 0; else echo "install: noop (no ./config)"; fi

clean: ## Clean artifacts
	rm -f $(COVERFILE) cpu.out mem.out


