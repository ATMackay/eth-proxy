# Alex Mackay 2024

# Build folder
BUILD_FOLDER = build

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

test: 
	go test -v -cover ./proxy ./client

test-integration:
	go test -v -cover ./integrationtests

test-benchmarks:
	go test -benchmem -bench BenchmarkConcurrentRequests ./integrationtests

docker:
	cd docker && ./build.sh
	@echo  "To run the application execute 'docker run -p 8080:8080 -e ETH_PROXY_URLS=<your_ethereum_api> eth-proxy'"

.PHONY: build run docker test test-stack test-benchmarks