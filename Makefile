# Alex Mackay 2024

build:
	GO111MODULE=on go build -ldflags "-w -linkmode external -extldflags '-static' -X 'github.com/ATMackay/eth-proxy/service.buildDate=$(shell date +"%Y-%m-%d %H:%M:%S")' -X 'github.com/ATMackay/eth-proxy/service.gitCommit=$(shell git rev-parse --short HEAD)'" ./cmd/eth-proxy
	mv eth-proxy ./build

run: build
	cd build && ./eth-proxy

test: 
	go test -v -cover ./service

test-stack:
	go test -v ./integrationtests

docker:
	cd docker && ./build.sh

.PHONY: build docker test test-stack run