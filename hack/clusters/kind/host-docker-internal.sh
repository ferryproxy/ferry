#!/usr/bin/env bash

# Need Docker 18.03
hostip=$(docker run --rm docker.io/library/alpine sh -c "nslookup host.docker.internal | grep 'Address' | grep -v '#' | grep -v ':53' | awk '{print \$2}' | head -n 1")

if [[ ${hostip} == "" ]]; then
  # For Docker running on Linux used 172.17.0.1 which is the Docker-host in Docker’s default-network.
  hostip="172.17.0.1"
fi

echo ${hostip}
