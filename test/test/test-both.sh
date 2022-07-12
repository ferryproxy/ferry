#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE}")/helpers.sh"

function check-both() {
  echo "::group::Check both"
  resource-apply control-plane <<EOF
apiVersion: traffic.ferryproxy.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  exports:
    - hubName: control-plane
      service:
        labels:
          app: web-0
    - hubName: cluster-1
      service:
        labels:
          app: web-1
  imports:
    - hubName: cluster-1
    - hubName: control-plane
EOF

  sleep 30
  fetch-route control-plane

  fetch-tunnel-config control-plane
  fetch-tunnel-config cluster-1

  echo "==== Check both: test cluster-1 to control-plane ===="
  check cluster-1 web-1 web-0.test.svc:80 "MESSAGE: cluster-0"
  check cluster-1 web-1 web-0.test.svc:8080 "MESSAGE: cluster-0"

  echo "==== Check both: test control-plane to cluster-1 ===="
  check control-plane web-0 web-1.test.svc:80 "MESSAGE: cluster-1"
  check control-plane web-0 web-1.test.svc:8080 "MESSAGE: cluster-1"

  echo "::endgroup::"
}

function check-0-to-1() {
  echo "::group::Check 0 to 1"
  resource-apply control-plane <<EOF
apiVersion: traffic.ferryproxy.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  exports:
    - hubName: control-plane
      service:
        labels:
          app: web-0
  imports:
    - hubName: cluster-1
EOF

  sleep 30
  fetch-route control-plane

  fetch-tunnel-config control-plane
  fetch-tunnel-config cluster-1

  echo "==== Check 0 to 1: test cluster-1 to control-plane ===="
  check cluster-1 web-1 web-0.test.svc:80 "MESSAGE: cluster-0"
  check cluster-1 web-1 web-0.test.svc:8080 "MESSAGE: cluster-0"

  echo "==== Check 0 to 1: test control-plane to cluster-1 ===="
  check-should-failed control-plane web-0 web-1.test.svc:80
  check-should-failed control-plane web-0 web-1.test.svc:8080

  echo "::endgroup::"
}

function check-1-to-0() {
  echo "::group::Check 1 to 0"
  resource-apply control-plane <<EOF
apiVersion: traffic.ferryproxy.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  exports:
    - hubName: cluster-1
      service:
        labels:
          app: web-1
  imports:
    - hubName: control-plane
EOF

  sleep 30
  fetch-route control-plane

  fetch-tunnel-config control-plane
  fetch-tunnel-config cluster-1

  echo "==== Check 1 to 0: test cluster-1 to control-plane ===="
  check-should-failed cluster-1 web-1 web-0.test.svc:80
  check-should-failed cluster-1 web-1 web-0.test.svc:8080

  echo "==== Check 1 to 0: test control-plane to cluster-1 ===="
  check control-plane web-0 web-1.test.svc:80 "MESSAGE: cluster-1"
  check control-plane web-0 web-1.test.svc:8080 "MESSAGE: cluster-1"

  echo "::endgroup::"
}


function check-none() {
  echo "::group::Check none"
  resource-apply control-plane <<EOF
apiVersion: traffic.ferryproxy.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  exports: []
  imports: []
EOF

  sleep 5
  fetch-route control-plane

  fetch-tunnel-config control-plane
  fetch-tunnel-config cluster-1

  echo "==== Check none: test cluster-1 to control-plane ===="
  check-should-failed cluster-1 web-1 web-0.test.svc:80
  check-should-failed cluster-1 web-1 web-0.test.svc:8080

  echo "==== Check none: test control-plane to cluster-1 ===="
  check-should-failed control-plane web-0 web-1.test.svc:80
  check-should-failed control-plane web-0 web-1.test.svc:8080

  echo "::endgroup::"
}

wait-controller-ready control-plane
wait-tunnel-ready cluster-1
wait-tunnel-ready control-plane

sleep 30

show-cluster-info control-plane
show-cluster-info cluster-1

show-hub control-plane
show-hub cluster-1

fetch-controller-log control-plane &
fetch-tunnel-log control-plane &
fetch-tunnel-log cluster-1 &

check-both
stats

check-0-to-1
stats

check-none
stats

check-1-to-0
stats

check-0-to-1
stats

check-both
stats

recreate-tunnel cluster-1
wait-tunnel-ready cluster-1
fetch-tunnel-log cluster-1 &

check-both
stats

check-0-to-1
stats

check-1-to-0
stats

check-0-to-1
stats

check-none
stats

check-both
stats

recreate-controller control-plane
wait-controller-ready control-plane
fetch-controller-log control-plane &

check-both
stats

check-none
stats

check-0-to-1
stats

check-1-to-0
stats

check-0-to-1
stats

check-both
stats
