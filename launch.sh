#!/bin/sh
docker-compose -f environment/docker-compose.yml up -d
nodeos -e -p eosio --config-dir `pwd`environment/ --delete-all-blocks --disable-replay-opts