# Go Ethereum Execution Client Proxy

## Components

* Go HTTP microservice built with [httprouter](https://github.com/julienschmidt/httprouter) exposing a [Prometheus](https://prometheus.io/)  metrics server endpoint.
* Lighteight Ethereum client interface with [go-ethereum](https://github.com/ethereum/go-ethereum/tree/master/ethclient)'s ethclient library
* Integration testing with go-ethereum's [simulation package](https://github.com/ethereum/go-ethereum/tree/master/ethclient/simulated)

## Getting started


Start service
```
~/go/src/github.com/ATMackay/eth-proxy$ make run
```

Use a new terminal to interact with the application. Use the `/status` endpoint to probe for liveness 

```
~$ curl localhost:8080/status
{"version":"v0.1.0-17379d11","service":"eth-proxy","failures":[]}
```

Use the `/health` endpoint to probe for readiness (an empty failures list indicates that the service is healthy and ready to take requests).
```
~$ curl localhost:8080/health
{"version":"v0.1.0-17379d11","service":"eth-proxy","failures":[]}
```

Use the /eth/balance to query the ether balance of your choice.
```
~$ curl localhost:8080/eth/balance/0xfe3b557e8fb62b89f4916b721be55ceb828dbd73
{"balance":"3074023230436339576"}
```

## Testing

Execute unit tests (providing coverage metrics)
```
~/go/src/github.com/ATMackay/eth-proxy$ make test
```

Stack tests (mocking eth-proxy Ethereum nodes interactions)
```
~/go/src/github.com/ATMackay/eth-proxy$ make test-stack
```

## Docker

Build Docker image
```
~/go/src/github.com/ATMackay/eth-proxy$ make docker
```