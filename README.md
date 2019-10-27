# Redis Proxy


## Features
 - Implements a proxy GET for redis

 - key and value pairs are cached on GET

 - cache is evicted in LRU fashion on reaching capacity

 - cached items have a time to live, on expiry items are evicted from cache

 - client requests has parallel concurrent processing  upto a configured max connections,
    after which the connections waits to be processed


## Build, Test and Run
 - make test
 - make
 - server starts on port 8080

## Usage
 - ex: `curl localhost:8080/GET?key=a`


## Architecture
  ![arhictecture](architecture.png)

## LRU cache
- I have implemented a ttl based lru cache forking the popular lru cache
  from hashicorp: https://github.com/gopheros/golang-lru