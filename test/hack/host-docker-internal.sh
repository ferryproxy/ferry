#!/usr/bin/env bash

# Need Docker 18.03
HOST_IP=$(docker run --rm docker.io/library/alpine sh -c "nslookup host.docker.internal | grep 'Address' | grep -v '#' | grep -v ':53' | awk '{print \$2}' | head -n 1")

if [[ ${HOST_IP} == "" ]]; then
  # For Docker running on Linux used 172.17.0.1 which is the Docker-host in Docker’s default-network.
  HOST_IP="172.17.0.1"
fi

echo ${HOST_IP}
