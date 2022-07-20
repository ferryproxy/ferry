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



CURRENT="$(dirname "${BASH_SOURCE}")"
ROOT="$(realpath "${CURRENT}/..")"
ENVIRONMENT_NAME="${1:-}"

ENVIRONMENT_DIR="${ROOT}/environments/${ENVIRONMENT_NAME}"

"${ROOT}/hack/kind/up.sh" "${ENVIRONMENT_NAME}"
"${ROOT}/hack/clusters/up.sh" "${ENVIRONMENT_NAME}"

if [ -f "${ENVIRONMENT_DIR}/init.sh" ]; then
  "${ENVIRONMENT_DIR}/init.sh"
fi