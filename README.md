# Platform Action Monitor
The service is responsible for processing events from the blockchain and their delivery to various participants in the system. Provides interfaces for subscribing to events.
### Stack
Golang, eos-go, pgx, gorilla/websocket
### How it works
https://daocasino.atlassian.net/wiki/spaces/DPM/pages/262438920/Event+broker
## How to use
### EOS
```BASH
brew tap eosio/eosio
brew install eosio
brew tap eosio/eosio.cdt
brew install eosio.cdt
```
### History tools
```BASH
git clone --recursive https://github.com/EOSIO/history-tools.git environment/history-tools
docker-compose -f environment/docker-compose.yml build
```
### Launch environments
The environment is configured so that every time the database and node is started, it is cleared and recreated.
#### History tools
Use `ifconfig` command to get the ip of the local interface `en0` and paste it in docker-compose.yml
```
... --fill-connect-to 192.168.1.75:8080
```
```BASH
docker-compose -f environment/docker-compose.yml up -d
```
Use `docker ps`, `docker logs` commands to check if history tools are working.
The database starts for a long time and the history tools may fall with an error. Just repeat `docker-compose up -d`
#### EOS node
```BASH
nodeos -e -p eosio --config-dir `pwd`environment/ --delete-all-blocks --disable-replay-opts
```
Option `--disable-replay-opts` is needed for `state-history-plugin`

### Database Migrations
Install tool [dbmate](https://github.com/amacneil/dbmate#installation) and run:
```
GO111MODULE=on go get -u github.com/amacneil/dbmate
dbmate up
```

### Launch service
```BASH
GO111MODULE=on go run cmd/monitor/main.go -config configs/config.yml
```
#### Dockerize
```BASH
$ docker-compose build
```
## Load testing
Use Artillery https://artillery.io/
```
config:
    target: "ws://localhost:8888/"
    phases:
        - duration: 60
          arrivalRate: 5
        # - duration: 120
        #   arrivalRate: 5
        #   rampTo: 50
        # - duration: 600
        #   arrivalRate: 50
scenarios:
    - engine: "ws"
      name: "Subscribe & Unsubscribe topic"
      flow:
        - send: '{"method":"subscribe","params":{"topic":"test","offset":0},"id":"0"}'
        - think: 0.5
        - send: '{"method":"unbscribe","params":{"topic":"test"},"id":"1"}'
```
```BASH
$ artillery run loadtest.yml
```

## Links
[]byte and JSON https://stackoverflow.com/questions/34089750/go-marshal-byte-to-json-giving-a-strange-string
