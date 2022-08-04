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

if [ "${REGION_ID}" == "" ] && [ "${ALIYUN_REGION_ID}" != "" ]; then
  REGION_ID="${ALIYUN_REGION_ID}"
fi

if [ "${REGION_ID}" == "" ]; then
  REGION_ID="cn-hongkong"
fi

if [ "${ZONE_ID}" == "" ] && [ "${ALIYUN_ZONE_ID}" != "" ]; then
  ZONE_ID="${ALIYUN_ZONE_ID}"
fi

if [ "${ZONE_ID}" == "" ]; then
  ZONE_ID="${REGION_ID}-b"
fi

if [ "${CLUSTER_NAME}" == "" ] || [ "${REGION_ID}" == "" ]; then
  echo "Usage: ${0} <cluster-name> <region-id>"
  exit 1
fi

body=$(
  cat <<EOF
{
  "cluster_type": "ManagedKubernetes",
  "name": "${CLUSTER_NAME}",
  "kubernetes_version": "1.22.10-aliyun.1",
  "region_id": "${REGION_ID}",
  "endpoint_public_access": true,
  "service_discovery_types": [
    "CoreDNS"
  ],
  "tags": [],
  "deletion_protection": false,
  "service_cidr": "172.21.0.0/20",
  "timezone": "UTC",
  "addons": [],
  "profile": "Serverless",
  "snat_entry": true,
  "zoneid": "${ZONE_ID}",
  "cluster_spec": "ack.pro.small",
  "load_balancer_spec": "slb.s1.small"
}
EOF
)

aliyun cs CreateCluster \
  --header "Content-Type=application/json" \
  --body "${body}"

while true; do
  state=$(aliyun cs DescribeClusters --name "${CLUSTER_NAME}" |
    jq -r '.[0].state')
  if [ "${state}" == "running" ]; then
    break
  fi
  echo "$(date) Waiting for cluster to be running... (state: ${state})"
  sleep 10
done

echo "$(date) Cluster ${CLUSTER_NAME} is running!"
