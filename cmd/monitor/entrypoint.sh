#!/bin/sh
dbmate up
dbmate -e SHARED_DATABASE_URL -d /app/shared-db/migrations up
./monitor -config /app/configs/config.yml