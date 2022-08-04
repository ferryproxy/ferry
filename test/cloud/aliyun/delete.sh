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

CURRENT_DIR="$(dirname "${BASH_SOURCE[0]}")"

CLUSTER_NAME=$1

REGION_ID=$2

ZONE_ID=$3

if [ "${CLUSTER_NAME}" == "" ]; then
  echo "Usage: ${0} <cluster-name>"
  exit 1
fi

TMPFILE=$(mktemp)

"${CURRENT_DIR}"/get_kubeconfig.sh "${CLUSTER_NAME}" "${REGION_ID}" "${ZONE_ID}" > "${TMPFILE}"

KUBECONFIG="${TMPFILE}" kubectl delete deploy,job -A --all

for _ in {1..10}; do
  state=$(aliyun cs DescribeClusters --name "${CLUSTER_NAME}" |
    jq -r '.[0].state')
  if [ "${state}" == "deleting" ]; then
    break
  fi

  aliyun cs DeleteCluster \
    --ClusterId $("${CURRENT_DIR}"/get.sh "${CLUSTER_NAME}")

  sleep 5
done

rm -f "${TMPFILE}"
