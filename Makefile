# Alex Mackay 2024

# Build folder
BUILD_FOLDER = build

# Test coverage variables
COVERAGE_BUILD_FOLDER = $(BUILD_FOLDER)/coverage
UNIT_COVERAGE_OUT  = $(COVERAGE_BUILD_FOLDER)/ut_cov.out

# Git based version
VERSION_TAG ?= $(shell git describe --tags)
GIT_COMMIT ?= $(shell git rev-parse HEAD)

COMMIT_DATE ?= $(shell git show -s --format="%ci" $(shell git rev-parse HEAD))

build:
	GO111MODULE=on go build -o $(BUILD_FOLDER)/eth-proxy -v \
	-ldflags=" -X 'github.com/ATMackay/eth-proxy/proxy.Version=$(VERSION_TAG)' -X 'github.com/ATMackay/eth-proxy/proxy.CommitDate=$(COMMIT_DATE)' -X 'github.com/ATMackay/eth-proxy/proxy.GitCommit=$(GIT_COMMIT)'" \
	./cmd/eth-proxy
	@echo  "To run the application execute ./build/eth-proxy --config config.yml"

run: build
	cd build && $(BUILD_FOLDER)/eth-proxy --config ../config.yml

build/coverage:
	mkdir -p $(COVERAGE_BUILD_FOLDER)

test: build/coverage
	go test -cover -coverprofile $(UNIT_COVERAGE_OUT) -v ./proxy ./client

test-coverage: test
	go tool cover -html=$(UNIT_COVERAGE_OUT)


test-integration:
	go test -v -cover ./integrationtests

test-benchmarks:
	go test -benchmem -bench BenchmarkConcurrentRequests ./integrationtests

docker:
	cd docker && ./build.sh
	@echo  "To run the application execute 'docker run -p 8080:8080 -e ETH_PROXY_URLS=<your_ethereum_api> eth-proxy'"

.PHONY: build run docker test test-stack test-benchmarks