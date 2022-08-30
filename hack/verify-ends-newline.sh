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


set -o errexit
set -o nounset
set -o pipefail

ROOT="$(dirname "${BASH_SOURCE[0]}")/.."

ROOT="$(realpath "${ROOT}")"

function check_ends() {
  find . \
    -iname "*.md"\
     -o -iname "*.sh" \
     -o -iname "*.bash" \
     -o -iname "*.go" \
     -o -iname "*.yaml" \
     -o -iname "*.yml" | \
     xargs -I {} bash -c "[ -n \"\$(tail -c 1 {})\" ] && echo {}" || :
}

out="$(check_ends)"
if [[ "${out}" != "" ]]; then
  echo "Add a new line in ends for blow files"
  echo "${out}"
  exit 1
fi
