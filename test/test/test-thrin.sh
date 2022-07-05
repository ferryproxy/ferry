#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE}")/helpers.sh"

function check-both() {
  echo "::group::Check both"
  resource-apply control-plane <<EOF
apiVersion: ferry.zsm.io/v1alpha1
kind: FerryPolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  rules:
    - exports:
        - clusterName: cluster-2
          match:
            labels:
              app: web-2
      imports:
        - clusterName: cluster-1

    - exports:
        - clusterName: cluster-1
          match:
            labels:
              app: web-1
      imports:
        - clusterName: cluster-2
EOF

  sleep 30
  fetch-mapping-rule control-plane

  fetch-tunnel-config cluster-2
  fetch-tunnel-config cluster-1

  echo "==== Check both: test cluster-1 to cluster-2 ===="
  check cluster-1 web-1 web-2.test.svc:80 "MESSAGE: cluster-2"
  check cluster-1 web-1 web-2.test.svc:8080 "MESSAGE: cluster-2"

  echo "==== Check both: test cluster-2 to cluster-1 ===="
  check cluster-2 web-2 web-1.test.svc:80 "MESSAGE: cluster-1"
  check cluster-2 web-2 web-1.test.svc:8080 "MESSAGE: cluster-1"

  echo "::endgroup::"
}

function check-2-to-1() {
  echo "::group::Check 2 to 1"
  resource-apply control-plane <<EOF
apiVersion: ferry.zsm.io/v1alpha1
kind: FerryPolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  rules:
    - exports:
        - clusterName: cluster-2
          match:
            labels:
              app: web-2
      imports:
        - clusterName: cluster-1
EOF

  sleep 30
  fetch-mapping-rule control-plane

  fetch-tunnel-config cluster-2
  fetch-tunnel-config cluster-1

  echo "==== Check 2 to 1: test cluster-1 to cluster-2 ===="
  check cluster-1 web-1 web-2.test.svc:80 "MESSAGE: cluster-2"
  check cluster-1 web-1 web-2.test.svc:8080 "MESSAGE: cluster-2"

  echo "==== Check 2 to 1: test cluster-2 to cluster-1 ===="
  check-should-failed cluster-2 web-2 web-1.test.svc:80
  check-should-failed cluster-2 web-2 web-1.test.svc:8080

  echo "::endgroup::"
}

function check-1-to-2() {
  echo "::group::Check 1 to 2"
  resource-apply control-plane <<EOF
apiVersion: ferry.zsm.io/v1alpha1
kind: FerryPolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  rules:
    - exports:
        - clusterName: cluster-1
          match:
            labels:
              app: web-1
      imports:
        - clusterName: cluster-2
EOF

  sleep 30
  fetch-mapping-rule control-plane

  fetch-tunnel-config cluster-2
  fetch-tunnel-config cluster-1

  echo "==== Check 1 to 2: test cluster-1 to cluster-2 ===="
  check-should-failed cluster-1 web-1 web-2.test.svc:80
  check-should-failed cluster-1 web-1 web-2.test.svc:8080

  echo "==== Check 1 to 2: test cluster-2 to cluster-1 ===="
  check cluster-2 web-2 web-1.test.svc:80 "MESSAGE: cluster-1"
  check cluster-2 web-2 web-1.test.svc:8080 "MESSAGE: cluster-1"

  echo "::endgroup::"
}

function check-0-to-1() {
  echo "::group::Check 0 to 1"
  resource-apply control-plane <<EOF
apiVersion: ferry.zsm.io/v1alpha1
kind: FerryPolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  rules:
    - exports:
        - clusterName: control-plane
          match:
            labels:
              app: web-0
      imports:
        - clusterName: cluster-1
EOF

  sleep 30
  fetch-mapping-rule control-plane

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
apiVersion: ferry.zsm.io/v1alpha1
kind: FerryPolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  rules:
    - exports:
        - clusterName: cluster-1
          match:
            labels:
              app: web-1
      imports:
        - clusterName: control-plane
EOF

  sleep 30
  fetch-mapping-rule control-plane

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
apiVersion: ferry.zsm.io/v1alpha1
kind: FerryPolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  rules: []
EOF

  sleep 10
  fetch-mapping-rule control-plane

  fetch-tunnel-config cluster-2
  fetch-tunnel-config cluster-1

  echo "==== Check none: test cluster-1 to cluster-2 ===="
  check-should-failed cluster-1 web-1 web-2.test.svc:80
  check-should-failed cluster-1 web-1 web-2.test.svc:8080

  echo "==== Check none: test cluster-2 to cluster-1 ===="
  check-should-failed cluster-2 web-2 web-1.test.svc:80
  check-should-failed cluster-2 web-2 web-1.test.svc:8080

  echo "::endgroup::"
}

wait-controller-ready control-plane
wait-tunnel-ready cluster-1
wait-tunnel-ready cluster-2

sleep 30

show-cluster-info control-plane
show-cluster-info cluster-2
show-cluster-info cluster-1

show-ferry-info control-plane
show-ferry-info cluster-2
show-ferry-info cluster-1

fetch-controller-log control-plane &
fetch-tunnel-log control-plane &
fetch-tunnel-log cluster-2 &
fetch-tunnel-log cluster-1 &

check-both
stats

check-2-to-1
stats

check-none
stats

check-1-to-0
stats

check-1-to-2
stats

check-2-to-1
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

check-2-to-1
stats

check-1-to-2
stats

check-2-to-1
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

check-2-to-1
stats

check-1-to-2
stats

check-2-to-1
stats

check-both
stats
