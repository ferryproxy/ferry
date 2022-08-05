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
ENVIRONMENT_NAME="${CURRENT##*/}"

ENVIRONMENT_DIR="${ROOT}/environments/${ENVIRONMENT_NAME}"
KUBECONFIG_DIR="${ROOT}/kubeconfigs"

HOST_IP="$(${ROOT}/hack/host-docker-internal.sh)"
echo "Host IP: ${HOST_IP}"

export KUBECONFIG
export FERRY_PEER_KUBECONFIG

echo "::group::Data plane cluster-0 initialization"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl data-plane init
ferryctl data-plane init
echo "::endgroup::"

echo "::group::Data plane cluster-1 initialization"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl data-plane init
ferryctl data-plane init
echo "::endgroup::"

echo "::group::Cluster cluster-0 export web-0 80"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
FERRY_PEER_KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "FERRY_PEER_KUBECONFIG=${FERRY_PEER_KUBECONFIG}"
echo ferryctl local manual export --reachable=true "--tunnel-address=${HOST_IP}:31000" --export-host-port=web-0.test.svc:80 --import-service-name=web-0-80
ferryctl local manual export --reachable=true "--tunnel-address=${HOST_IP}:31000" --export-host-port=web-0.test.svc:80 --import-service-name=web-0-80
echo "::endgroup::"

echo "::group::Cluster cluster-1 export web-1 80"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
FERRY_PEER_KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "FERRY_PEER_KUBECONFIG=${FERRY_PEER_KUBECONFIG}"
echo ferryctl local manual export --reachable=false "--peer-tunnel-address=${HOST_IP}:31000" --export-host-port=web-1.test.svc:80 --import-service-name=web-1-80
ferryctl local manual export --reachable=false "--peer-tunnel-address=${HOST_IP}:31000" --export-host-port=web-1.test.svc:80 --import-service-name=web-1-80
echo "::endgroup::"

echo "::group::Cluster cluster-0 import web-1 8080"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
FERRY_PEER_KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "FERRY_PEER_KUBECONFIG=${FERRY_PEER_KUBECONFIG}"
echo ferryctl local manual import --reachable=true "--tunnel-address=${HOST_IP}:31000" --export-host-port=web-1.test.svc:8080 --import-service-name=web-1-8080
ferryctl local manual import --reachable=true "--tunnel-address=${HOST_IP}:31000" --export-host-port=web-1.test.svc:8080 --import-service-name=web-1-8080
echo "::endgroup::"

echo "::group::Cluster cluster-1 import web-0 8080"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
FERRY_PEER_KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "FERRY_PEER_KUBECONFIG=${FERRY_PEER_KUBECONFIG}"
echo ferryctl local manual import --reachable=false "--peer-tunnel-address=${HOST_IP}:31000" --export-host-port=web-0.test.svc:8080 --import-service-name=web-0-8080
ferryctl local manual import --reachable=false "--peer-tunnel-address=${HOST_IP}:31000" --export-host-port=web-0.test.svc:8080 --import-service-name=web-0-8080
echo "::endgroup::"
