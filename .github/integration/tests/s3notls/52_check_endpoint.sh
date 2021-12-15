#!/bin/bash

cd dev_utils || exit 1

# Test Health Endpoint
check_health=$(curl -o /dev/null -s -w "%{http_code}\n" http://localhost:8080/health)

if [ "$check_health" != "200" ]; then
    echo "Health endpoint does not respond properly"
    echo "got: ${check_health}"
    exit 1
fi

echo "Health endpoint is ok"

# Test empty token

check_401=$(curl -o /dev/null -s -w "%{http_code}\n" http://localhost:8080/metadata/datasets)

if [ "$check_401" != "401" ]; then
    echo "no token provided should give 401"
    echo "got: ${check_401}"
    exit 1
fi

echo "got correct response when no token provided"
