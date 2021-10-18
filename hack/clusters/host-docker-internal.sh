#!/usr/bin/env bash

# Need Docker 18.03
docker run --rm -it alpine sh -c "nslookup host.docker.internal | grep 'Address' | grep -v '#' | grep -v ':53' | awk '{print \$2}' | head -n 1"
