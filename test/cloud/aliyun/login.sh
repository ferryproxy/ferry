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

ACCESS_KEY_ID=$1
ACCESS_KEY_SECRET=$2
REGION_ID=$3

if [ "${ACCESS_KEY_ID}" == "" ] && [ "${ALIYUN_ACCESS_KEY_ID}" != "" ]; then
  ACCESS_KEY_ID="${ALIYUN_ACCESS_KEY_ID}"
fi

if [ "${ACCESS_KEY_SECRET}" == "" ] && [ "${ALIYUN_ACCESS_KEY_SECRET}" != "" ]; then
  ACCESS_KEY_SECRET="${ALIYUN_ACCESS_KEY_SECRET}"
fi

if [ "${REGION_ID}" == "" ] && [ "${ALIYUN_REGION_ID}" != "" ]; then
  REGION_ID="${ALIYUN_REGION_ID}"
fi

if [ "${REGION_ID}" == "" ]; then
  REGION_ID="cn-hongkong"
fi

if [ "${ACCESS_KEY_ID}" == "" ] || [ "${ACCESS_KEY_SECRET}" == "" ] || [ "${REGION_ID}" == "" ]; then
  echo "Usage: ${0} <access-key-id> <access-key-secret> <region-id>"
  exit 1
fi

aliyun configure set \
  --access-key-id "${ACCESS_KEY_ID}" \
  --access-key-secret "${ACCESS_KEY_SECRET}" \
  --region="${REGION_ID}"
