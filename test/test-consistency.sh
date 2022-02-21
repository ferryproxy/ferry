#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE}")/helpers.sh"

function check-consistency() {
  echo "==== Test data-plane-cluster-1 to control-plane-cluster ===="
  check data-plane-cluster-1 web-1 web-0.test.svc:80 "MESSAGE: cluster-0"
  check data-plane-cluster-1 web-1 web-0.test.svc:8080 "MESSAGE: cluster-0"
  check data-plane-cluster-1 web-1 web-0-0.test.svc:80 "MESSAGE: cluster-0"
  check data-plane-cluster-1 web-1 web-0-0.test.svc:8080 "MESSAGE: cluster-0"

  echo "==== Test control-plane-cluster to data-plane-cluster-1 ===="
  check control-plane-cluster web-0 web-1.test.svc:80 "MESSAGE: cluster-1"
  check control-plane-cluster web-0 web-1.test.svc:8080 "MESSAGE: cluster-1"
  check control-plane-cluster web-0 web-1-1.test.svc:80 "MESSAGE: cluster-1"
  check control-plane-cluster web-0 web-1-1.test.svc:8080 "MESSAGE: cluster-1"

  echo "==== Test data-plane-cluster-2 to control-plane-cluster ===="
  check data-plane-cluster-2 web-2 web-0.test.svc:80 "MESSAGE: cluster-0"
  check data-plane-cluster-2 web-2 web-0.test.svc:8080 "MESSAGE: cluster-0"
  check data-plane-cluster-2 web-2 web-0-0.test.svc:80 "MESSAGE: cluster-0"
  check data-plane-cluster-2 web-2 web-0-0.test.svc:8080 "MESSAGE: cluster-0"

  echo "==== Test control-plane-cluster to data-plane-cluster-2 ===="
  check control-plane-cluster web-0 web-2.test.svc:80 "MESSAGE: cluster-2"
  check control-plane-cluster web-0 web-2.test.svc:8080 "MESSAGE: cluster-2"
  check control-plane-cluster web-0 web-2-2.test.svc:80 "MESSAGE: cluster-2"
  check control-plane-cluster web-0 web-2-2.test.svc:8080 "MESSAGE: cluster-2"

  echo "==== Test data-plane-cluster-1 to data-plane-cluster-2 ===="
  check data-plane-cluster-1 web-1 web-2.test.svc:80 "MESSAGE: cluster-2"
  check data-plane-cluster-1 web-1 web-2.test.svc:8080 "MESSAGE: cluster-2"

  echo "==== Test data-plane-cluster-2 to data-plane-cluster-1 ===="
  check data-plane-cluster-2 web-2 web-1.test.svc:80 "MESSAGE: cluster-1"
  check data-plane-cluster-2 web-2 web-1.test.svc:8080 "MESSAGE: cluster-1"
}

resource-apply control-plane-cluster <<EOF
apiVersion: ferry.zsm.io/v1alpha1
kind: FerryPolicy
metadata:
  name: ferry-test
  namespace: ferry-system
spec:
  rules:
    - exports:
        - clusterName: cluster-0
          match:
            labels:
              app: web-0

      imports:
        - clusterName: cluster-1
        - clusterName: cluster-2

    - exports:
        - clusterName: cluster-1
          match:
            labels:
              app: web-1
        - clusterName: cluster-2
          match:
            labels:
              app: web-2

      imports:
        - clusterName: cluster-0


    - exports:
        - clusterName: cluster-0
          match:
            namespace: test
            name: web-0

      imports:
        - clusterName: cluster-1
          match:
            namespace: test
            name: web-0-0
        - clusterName: cluster-2
          match:
            namespace: test
            name: web-0-0

    - exports:
        - clusterName: cluster-1
          match:
            namespace: test
            name: web-1

      imports:
        - clusterName: cluster-0
          match:
            namespace: test
            name: web-1-1

    - exports:
        - clusterName: cluster-2
          match:
            namespace: test
            name: web-2

      imports:
        - clusterName: cluster-0
          match:
            namespace: test
            name: web-2-2

    - exports:
        - clusterName: cluster-2
          match:
            namespace: test
            name: web-2

      imports:
        - clusterName: cluster-1
          match:
            namespace: test
            name: web-2

    - exports:
        - clusterName: cluster-1
          match:
            namespace: test
            name: web-1

      imports:
        - clusterName: cluster-2
          match:
            namespace: test
            name: web-1
EOF

wait-controller-ready control-plane-cluster
wait-tunnel-ready data-plane-cluster-2
wait-tunnel-ready data-plane-cluster-1
wait-tunnel-ready control-plane-cluster

fetch-tunnel-config control-plane-cluster
fetch-tunnel-config data-plane-cluster-1
fetch-tunnel-config data-plane-cluster-2

fetch-controller-log control-plane-cluster &
fetch-tunnel-log control-plane-cluster &
fetch-tunnel-log data-plane-cluster-1 &
fetch-tunnel-log data-plane-cluster-2 &

NAME=base check-consistency
stats

recreate-controller control-plane-cluster
wait-controller-ready control-plane-cluster
sleep 5

NAME="recreate controller" check-consistency
stats

recreate-tunnel data-plane-cluster-1
wait-tunnel-ready data-plane-cluster-1
sleep 5

NAME="recreate tunnel of cluster-1" check-consistency
stats

recreate-tunnel control-plane-cluster
wait-tunnel-ready control-plane-cluster
sleep 5

NAME="recreate tunnel of plane-cluster" check-consistency
stats