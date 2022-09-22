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


source "$(dirname "${BASH_SOURCE[0]}")/helpers.sh"

ROUTE_NAME="${ROUTE_NAME:-ferry-test}"
CONTROL_PLANE="${CONTROL_PLANE:-control-plane}"
CLUSTER_1="${CLUSTER_1:-cluster-1}"
CLUSTER_2="${CLUSTER_2:-cluster-2}"
CLUSTER_3="${CLUSTER_3:-cluster-3}"
TARGET_1="${TARGET_1:-web-1}"
TARGET_2="${TARGET_2:-web-2}"
TARGET_3="${TARGET_2:-web-3}"

function check-cross() {
  echo "::group::Check cross"
  resource-apply "${CONTROL_PLANE}" <<EOF
apiVersion: traffic.ferryproxy.io/v1alpha2
kind: RoutePolicy
metadata:
  name: ${ROUTE_NAME}
  namespace: ferry-system
spec:
  exports:
    - hubName: ${CLUSTER_2}
      service:
        namespace: test
        name: ${TARGET_2}
    - hubName: ${CLUSTER_3}
      service:
        namespace: test
        name: ${TARGET_3}
  imports:
    - hubName: ${CLUSTER_1}
      service:
        namespace: test
        name: sum
EOF

  wait-routes-ready "${CONTROL_PLANE}"
  fetch-route "${CONTROL_PLANE}"
  fetch-tunnel-config "${CLUSTER_3}"
  fetch-tunnel-config "${CLUSTER_2}"
  fetch-tunnel-config "${CLUSTER_1}"

  echo "==== Check both: test ${CLUSTER_1} to ${CLUSTER_2} ===="
  check "${CLUSTER_1}" "${TARGET_1}" "sum.test.svc:80" "MESSAGE: ${TARGET_2}" 10
  check "${CLUSTER_1}" "${TARGET_1}" "sum.test.svc:8080" "MESSAGE: ${TARGET_2}" 10

  echo "==== Check both: test ${CLUSTER_1} to ${CLUSTER_3} ===="
  check "${CLUSTER_1}" "${TARGET_1}" "sum.test.svc:80" "MESSAGE: ${TARGET_3}" 10
  check "${CLUSTER_1}" "${TARGET_1}" "sum.test.svc:8080" "MESSAGE: ${TARGET_3}" 10

  echo "::endgroup::"
}

wait-hubs-ready "${CONTROL_PLANE}"
wait-pods-ready "${CONTROL_PLANE}"
wait-pods-ready "${CLUSTER_1}"
wait-pods-ready "${CLUSTER_2}"
wait-pods-ready "${CLUSTER_3}"

show-cluster-info "${CONTROL_PLANE}"

show-hub "${CONTROL_PLANE}"

fetch-controller-log "${CONTROL_PLANE}" &
fetch-tunnel-log "${CLUSTER_3}" &
fetch-tunnel-log "${CLUSTER_2}" &
fetch-tunnel-log "${CLUSTER_1}" &

check-cross
