#!/usr/bin/env bash
# Copyright 2022 FerryProxy Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


source "$(dirname "${BASH_SOURCE}")/helpers.sh"

ROUTE_NAME="${ROUTE_NAME:-ferry-test}"
CONTROL_PLANE="${CONTROL_PLANE:-control-plane}"
CLUSTER_1="${CLUSTER_1:-cluster-1}"
CLUSTER_2="${CLUSTER_2:-cluster-2}"
TARGET_1="${TARGET_1:-web-1}"
TARGET_2="${TARGET_2:-web-2}"

function check-both() {
  echo "::group::Check both"
  resource-apply control-plane <<EOF
apiVersion: traffic.ferryproxy.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ${ROUTE_NAME}
  namespace: ferry-system
spec:
  exports:
    - hubName: ${CLUSTER_2}
      service:
        labels:
          app: ${TARGET_2}
    - hubName: ${CLUSTER_1}
      service:
        labels:
          app: ${TARGET_1}
  imports:
    - hubName: ${CLUSTER_1}
    - hubName: ${CLUSTER_2}
EOF

  fetch-route control-plane

  fetch-tunnel-config "${CLUSTER_2}"
  fetch-tunnel-config "${CLUSTER_1}"

  echo "==== Check both: test ${CLUSTER_1} to ${CLUSTER_2} ===="
  check "${CLUSTER_1}" "${TARGET_1}" "${TARGET_2}.test.svc:80" "MESSAGE: ${TARGET_2}" 10
  check "${CLUSTER_1}" "${TARGET_1}" "${TARGET_2}.test.svc:8080" "MESSAGE: ${TARGET_2}" 10

  echo "==== Check both: test ${CLUSTER_2} to ${CLUSTER_1} ===="
  check "${CLUSTER_2}" "${TARGET_2}" "${TARGET_1}.test.svc:80" "MESSAGE: ${TARGET_1}" 10
  check "${CLUSTER_2}" "${TARGET_2}" "${TARGET_1}.test.svc:8080" "MESSAGE: ${TARGET_1}" 10

  echo "::endgroup::"
}

function check-2-to-1() {
  echo "::group::Check 2 to 1"
  resource-apply control-plane <<EOF
apiVersion: traffic.ferryproxy.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ${ROUTE_NAME}
  namespace: ferry-system
spec:
  exports:
    - hubName: ${CLUSTER_2}
      service:
        labels:
          app: ${TARGET_2}
  imports:
    - hubName: ${CLUSTER_1}
EOF

  fetch-route control-plane

  fetch-tunnel-config "${CLUSTER_2}"
  fetch-tunnel-config "${CLUSTER_1}"

  echo "==== Check 2 to 1: test ${CLUSTER_1} to ${CLUSTER_2} ===="
  check "${CLUSTER_1}" "${TARGET_1}" "${TARGET_2}.test.svc:80" "MESSAGE: ${TARGET_2}" 10
  check "${CLUSTER_1}" "${TARGET_1}" "${TARGET_2}.test.svc:8080" "MESSAGE: ${TARGET_2}" 10

  echo "==== Check 2 to 1: test ${CLUSTER_2} to ${CLUSTER_1} ===="
  check-should-failed "${CLUSTER_2}" "${TARGET_2}" "${TARGET_1}.test.svc:80" 10
  check-should-failed "${CLUSTER_2}" "${TARGET_2}" "${TARGET_1}.test.svc:8080" 10

  echo "::endgroup::"
}

function check-1-to-2() {
  echo "::group::Check 1 to 2"
  resource-apply control-plane <<EOF
apiVersion: traffic.ferryproxy.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ${ROUTE_NAME}
  namespace: ferry-system
spec:
  exports:
    - hubName: ${CLUSTER_1}
      service:
        labels:
          app: ${TARGET_1}
  imports:
    - hubName: ${CLUSTER_2}
EOF

  fetch-route control-plane

  fetch-tunnel-config "${CLUSTER_2}"
  fetch-tunnel-config "${CLUSTER_1}"

  echo "==== Check 1 to 2: test ${CLUSTER_2} to ${CLUSTER_1} ===="
  check "${CLUSTER_2}" "${TARGET_2}" "${TARGET_1}.test.svc:80" "MESSAGE: ${TARGET_1}" 10
  check "${CLUSTER_2}" "${TARGET_2}" "${TARGET_1}.test.svc:8080" "MESSAGE: ${TARGET_1}" 10

  echo "==== Check 1 to 2: test ${CLUSTER_1} to ${CLUSTER_2} ===="
  check-should-failed "${CLUSTER_1}" "${TARGET_1}" "${TARGET_2}.test.svc:80" 10
  check-should-failed "${CLUSTER_1}" "${TARGET_1}" "${TARGET_2}.test.svc:8080" 10

  echo "::endgroup::"
}

function check-none() {
  echo "::group::Check none"
  resource-apply "${CONTROL_PLANE}" <<EOF
apiVersion: traffic.ferryproxy.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ${ROUTE_NAME}
  namespace: ferry-system
spec:
  exports: []
  imports: []
EOF

  fetch-route "${CONTROL_PLANE}"

  fetch-tunnel-config "${CLUSTER_2}"
  fetch-tunnel-config "${CLUSTER_1}"

  echo "==== Check none: test ${CLUSTER_1} to ${CLUSTER_2} ===="
  check-should-failed "${CLUSTER_1}" "${TARGET_1}" "${TARGET_2}.test.svc:80" 10
  check-should-failed "${CLUSTER_1}" "${TARGET_1}" "${TARGET_2}.test.svc:8080" 10

  echo "==== Check none: test ${CLUSTER_2} to ${CLUSTER_1} ===="
  check-should-failed "${CLUSTER_2}" "${TARGET_2}" "${TARGET_1}.test.svc:80" 10
  check-should-failed "${CLUSTER_2}" "${TARGET_2}" "${TARGET_1}.test.svc:8080" 10

  echo "::endgroup::"
}

function rand() {
  local max=$1
  echo $(("${RANDOM}" % "${max}"))
}

function steps() {
  local times=$1
  local last=""
  local cur="0"

  for _ in $(seq "${times}"); do
    while [[ "${cur}" == "${last}" ]]; do
      cur="$(rand 4)"
    done

    case "${cur}" in
    0)
      check-both
      ;;
    1)
      check-1-to-2
      ;;
    2)
      check-2-to-1
      ;;
    3)
      check-none
      ;;
    esac

    stats

    last="${cur}"
  done
}

wait-controller-ready "${CONTROL_PLANE}"
wait-tunnel-ready "${CLUSTER_1}"
wait-tunnel-ready "${CLUSTER_2}"

show-cluster-info "${CONTROL_PLANE}"

show-hub "${CONTROL_PLANE}"

fetch-controller-log "${CONTROL_PLANE}" &
fetch-tunnel-log "${CLUSTER_2}" &
fetch-tunnel-log "${CLUSTER_1}" &

steps 10

recreate-tunnel "${CLUSTER_1}"
wait-tunnel-ready "${CLUSTER_1}"
fetch-tunnel-log "${CLUSTER_1}" &

steps 2

recreate-controller "${CONTROL_PLANE}"
wait-controller-ready "${CONTROL_PLANE}"
fetch-controller-log "${CONTROL_PLANE}" &

steps 2

recreate-tunnel "${CLUSTER_2}"
wait-tunnel-ready "${CLUSTER_2}"
fetch-tunnel-log "${CLUSTER_2}" &
