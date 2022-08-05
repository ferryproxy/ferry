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

if [ "${REGION_ID}" == "" ] && [ "${AZURE_REGION_ID}" != "" ]; then
  REGION_ID="${AZURE_REGION_ID}"
fi

if [ "${REGION_ID}" == "" ]; then
  REGION_ID="westus"
fi

if [ "${CLUSTER_NAME}" == "" ]; then
  echo "Usage: ${0} <cluster-name> [region-id]"
  exit 1
fi

az group create --location "${REGION_ID}" --resource-group "${CLUSTER_NAME}Group"

az aks create \
  --name "${CLUSTER_NAME}" \
  --resource-group "${CLUSTER_NAME}Group" \
  --node-count 1 \
  --no-ssh-key
