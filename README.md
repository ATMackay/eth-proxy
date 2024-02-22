# Go Ethereum Execution Client Proxy

## Components

* Go HTTP microservice built with [httprouter](https://github.com/julienschmidt/httprouter) exposing a [Prometheus](https://prometheus.io/)  metrics server endpoint.
* Lightweight Ethereum client interface using [go-ethereum](https://github.com/ethereum/go-ethereum/tree/master/ethclient)'s ethclient library.
* Integration testing with go-ethereum's [simulation package](https://github.com/ethereum/go-ethereum/tree/master/ethclient/simulated).

## Getting started


Start the application
```
~/go/src/github.com/ATMackay/eth-proxy$ make run
```

Open a new terminal to interact with the application. Use the `/status` endpoint to probe for liveness (it will always return OK)

```
~$ curl localhost:8080/status
{"message":"OK","version":"0.1.0-992d0028","service":"eth-proxy"}
```

Use the `/health` endpoint to probe for readiness (an empty failures list indicates that the service is healthy and ready to take requests)
```
~$ curl localhost:8080/health
{"version":"v0.1.0-992d0028","service":"eth-proxy","failures":[]}
```

Use the `/eth/balance/<addr>` to query the ether balance for an address of your choice. For example
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

To run with docker first build the `eth-proxy` Docker image
```
~/go/src/github.com/ATMackay/eth-proxy$ make docker
```

```
~/go/src/github.com/ATMackay/eth-proxy$ docker images
REPOSITORY                                                                      TAG                 IMAGE ID       CREATED          SIZE
eth-proxy                                                                       f38d1fc             d50fcb2ad302   2 minutes ago   13.8MB
eth-proxy                                                                       latest              d50fcb2ad302   2 minutes ago   13.8MB
```

Run a container
```
~$ docker run -p 8080:8080 -e ETH_PROXY_URLS=https://mainnet.infura.io/v3/4c664372f60943f690c615f182d50c63 eth-proxy
```