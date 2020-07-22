#!/bin/sh

set -e

# optional cleanup of history db migrations before run
if [[ $CLEANUP_HISTORY != "" ]]; then
    echo "Cleanning daobet history migrations"
    dbmate down || true
fi

# run daobet history db mirgations
echo "Running daobet history migrations"
dbmate up

# run action monitor db migrations
echo "Running action monitor migrations"
dbmate -e SHARED_DATABASE_URL -d /app/shared-db/migrations up

# run action monitor
exec ./monitor -config /app/configs/config.yml
