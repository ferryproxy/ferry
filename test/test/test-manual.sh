#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE}")/helpers.sh"

function check-both() {
  echo "::group::Check both"

  sleep 30
  fetch-tunnel-config cluster-0
  fetch-tunnel-config cluster-1

  echo "==== Check both: test cluster-1 to cluster-0 for 80 ===="
  check cluster-1 web-1 web-0-80.ferry-tunnel-system.svc:80 "MESSAGE: cluster-0"

  echo "==== Check both: test cluster-0 to cluster-1 for 80 ===="
  check cluster-0 web-0 web-1-80.ferry-tunnel-system.svc:80 "MESSAGE: cluster-1"

  echo "==== Check both: test cluster-1 to cluster-0 for 8080 ===="
  check cluster-1 web-1 web-0-8080.ferry-tunnel-system.svc:8080 "MESSAGE: cluster-0"

  echo "==== Check both: test cluster-0 to cluster-1 for 8080 ===="
  check cluster-0 web-0 web-1-8080.ferry-tunnel-system.svc:8080 "MESSAGE: cluster-1"

  echo "::endgroup::"
}

check-both
stats
