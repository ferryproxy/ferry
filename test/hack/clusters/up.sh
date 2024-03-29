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



CURRENT="$(dirname "${BASH_SOURCE[0]}")"
ROOT="$(realpath "${CURRENT}/../..")"
ENVIRONMENT_NAME="${1:-}"

if [ -z "${ENVIRONMENT_NAME}" ]; then
  echo "Usage: ${0} <environment-name>"
  exit 1
fi

ENVIRONMENT_DIR="${ROOT}/environments/${ENVIRONMENT_NAME}"
KUBECONFIG_DIR="${ROOT}/kubeconfigs"

for name in $(ls ${ENVIRONMENT_DIR} | grep -v in-cluster | grep .yaml); do
  name="${name%.*}"
  kubeconfig="${KUBECONFIG_DIR}/${name}.yaml"
  kubectl --kubeconfig "${kubeconfig}" apply -k "${ENVIRONMENT_DIR}/${name}"
done
