# Alex Mackay 2024

build:
	GO111MODULE=on go build ./cmd/eth-proxy
	mv eth-proxy ./build
	@echo  "To run the application execute ./build/eth-proxy --config config.yml"

run: build
	cd build && ./eth-proxy --config ../config.yml

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