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

function check-export-reachable() {
  echo "::group::Check export reachable"

  echo "==== Check both: test cluster-1 to cluster-0 for 80 ===="
  check cluster-1 web-1 web-0-80.ferry-tunnel-system.svc:80 "MESSAGE: web-0" 10

  echo "==== Check both: test cluster-1 to cluster-0 for 8080 ===="
  check cluster-1 web-1 web-0-8080.ferry-tunnel-system.svc:8080 "MESSAGE: web-0" 10

  echo "::endgroup::"
}

function check-import-reachable() {
  echo "::group::Check export reachable"

  echo "==== Check both: test cluster-0 to cluster-1 for 80 ===="
  check cluster-0 web-0 web-1-80.ferry-tunnel-system.svc:80 "MESSAGE: web-1" 10

  echo "==== Check both: test cluster-0 to cluster-1 for 8080 ===="
  check cluster-0 web-0 web-1-8080.ferry-tunnel-system.svc:8080 "MESSAGE: web-1" 10

  echo "::endgroup::"
}

wait-pods-ready cluster-0
fetch-tunnel-log cluster-0 &
wait-pods-ready cluster-1
fetch-tunnel-log cluster-1 &

fetch-tunnel-config cluster-0
fetch-tunnel-config cluster-1

check-import-reachable

stats

check-export-reachable

stats
