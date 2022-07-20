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


# Need Docker 18.03
HOST_IP=$(docker run --rm docker.io/library/alpine sh -c "nslookup host.docker.internal | grep 'Address' | grep -v '#' | grep -v ':53' | awk '{print \$2}' | head -n 1")

if [[ ${HOST_IP} == "" ]]; then
  # For Docker running on Linux used 172.17.0.1 which is the Docker-host in Docker’s default-network.
  HOST_IP="172.17.0.1"
fi

echo ${HOST_IP}
