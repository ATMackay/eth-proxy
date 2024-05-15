# Alex Mackay 2024

build:
	GO111MODULE=on go build -ldflags "-w -linkmode external -extldflags '-static' -X 'github.com/ATMackay/eth-proxy/service.buildDate=$(shell date +"%Y-%m-%d %H:%M:%S")' -X 'github.com/ATMackay/eth-proxy/service.gitCommit=$(shell git rev-parse --short HEAD)'" ./cmd/eth-proxy
	mv eth-proxy ./build
	@echo  "To run the application execute ./build/eth-proxy --config config.yml"

run: build
	cd build && ./eth-proxy --config ../config.yml

test: 
	go test -v -cover ./service ./client

test-stack:
	go test -v -cover ./integrationtests

test-benchmarks:
	go test -benchmem -bench BenchmarkConcurrentRequests ./integrationtests

docker:
	cd docker && ./build.sh
	@echo  "To run the application execute 'docker run -p 8080:8080 -e ETH_PROXY_URLS=<your_ethereum_api> eth-proxy'"

.PHONY: build run docker test test-stack test-benchmarks