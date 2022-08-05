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

if [ "${REGION_ID}" == "" ] && [ "${AWS_REGION_ID}" != "" ]; then
  REGION_ID="${AWS_REGION_ID}"
fi

if [ "${REGION_ID}" == "" ]; then
  REGION_ID="us-east-1"
fi


eksctl delete cluster \
  --name "${CLUSTER_NAME}" \
  --region "${REGION_ID}" \
  --force
