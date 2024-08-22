#!/bin/bash
mkdir -p data/pgdata
mkdir -p data/pgsocket
docker rm -f /playdough_test_pg
docker run \
    --name playdough_test_pg \
    -e POSTGRES_DB=playdough_test \
    -e POSTGRES_USER=playdough_test \
    -e POSTGRES_PASSWORD=hunter2 \
    --mount type=bind,source="$(pwd)"/data/pgdata,target=/var/lib/postgresql/data \
    --mount type=bind,source="$(pwd)"/data/pgsocket,target=/var/run/postgresql \
    postgres:16.4 \
    -c "unix_socket_directories=/var/run/postgresql" \
    -c "listen_addresses="
# Connect with:
# psql -h $(pwd)/data/pgsocket -U playdough_test -d playdough_test
