# Go Ethereum Execution Client Proxy


## Getting started


Start service
```
~/go/src/github.com/ATMackay/eth-proxy$ make run
```

Use a new terminal to interact with the application. Use the Status check to probe for liveness 

```
~$ curl localhost:8080/status
{"version":"v0.1.0-17379d11","service":"eth-proxy","failures":[]}
```

Use the Healthcheck probe to check for readiness (an empty failures list indicates that the service is healthy and ready to take requests).
```
~$ curl localhost:8080/health
{"version":"v0.1.0-17379d11","service":"eth-proxy","failures":[]}
```

Use the /eth/balance to query the ether balance of your choice.
```
~$ curl localhost:8080/eth/balance/0xfe3b557e8fb62b89f4916b721be55ceb828dbd73
{"balance":"3074023230436339576"}
```