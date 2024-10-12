#!/usr/bin/env bash

set -e

commit_hash=$(git rev-parse HEAD)
commit_hash_short=$(git rev-parse --short HEAD)
commit_timestamp=$(git show -s --format="%ci" ${commit_hash})
version_tag=$(git describe --tags)

docker build \
       --build-arg SERVICE=eth-proxy \
       --build-arg GIT_COMMIT="$commit_hash" \
       --build-arg COMMIT_DATE="$commit_timestamp" \
       --build-arg VERSION_TAG="$version_tag" \
       -t eth-proxy:latest  \
       -t eth-proxy:"$commit_hash_short"  \
       -f Dockerfile ..
       
# Remove intermediate Docker layers
docker image prune -f