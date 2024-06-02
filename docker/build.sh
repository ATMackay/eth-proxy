#!/usr/bin/env bash

set -e

commit_hash=$(git rev-parse HEAD)
commit_hash_short=$(git rev-parse --short HEAD)
commit_timestamp=$(git show -s --format="%ci" ${commit_hash})

docker build \
       --build-arg SERVICE=eth-proxy \
       --build-arg GIT_SHA="$commit_hash" \
       --build-arg GIT_TIMESTAMP="$commit_timestamp" \
       -t eth-proxy:latest  \
       -t eth-proxy:"$commit_hash_short"  \
       -f Dockerfile ..