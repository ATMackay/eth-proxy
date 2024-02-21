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

Use a new terminal to interact with the application. Use the `/status` endpoint to probe for liveness (it will always return OK).

```
~$ curl localhost:8080/status
{"message":"OK","version":"0.1.0-992d0028","service":"eth-proxy"}
```

Use the `/health` endpoint to probe for readiness (an empty failures list indicates that the service is healthy and ready to take requests).
```
~$ curl localhost:8080/health
{"version":"v0.1.0-992d0028","service":"eth-proxy","failures":[]}
```

Use the /eth/balance to query the ether balance of your choice.
```
~$ curl localhost:8080/eth/balance/0xfe3b557e8fb62b89f4916b721be55ceb828dbd73
{"balance":"14058"}
```

Check metrics using the Prometheus server `/metrics` endpoint
```
~$ curl localhost:8080/metrics
# HELP go_cgo_go_to_c_calls_calls_total Count of calls made from Go to C by the current process.
# TYPE go_cgo_go_to_c_calls_calls_total counter
go_cgo_go_to_c_calls_calls_total 4
...
# HELP promhttp_metric_handler_requests_total Total number of scrapes by HTTP status code.
# TYPE promhttp_metric_handler_requests_total counter
promhttp_metric_handler_requests_total{code="200"} 0
promhttp_metric_handler_requests_total{code="500"} 0
promhttp_metric_handler_requests_total{code="503"} 0
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