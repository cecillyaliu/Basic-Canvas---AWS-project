#!/bin/bash

cd `dirname $0`
pwd

./demo \
    --db_endpoint="${DB_ENDPOINT}" \
    --db_username="${DB_USERNAME}" \
    --db_password="${DB_PASSWORD}"