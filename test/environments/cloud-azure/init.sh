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

export KUBECONFIG
export FERRY_PEER_KUBECONFIG

echo "::group::Control plane initialization"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl control-plane init --control-plane-reachable=false
ferryctl control-plane init --control-plane-reachable=false
echo "::endgroup::"

echo "::group::Data plane cluster-azure initialization"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-azure.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl data-plane init --tunnel-service-type=LoadBalancer
ferryctl data-plane init --tunnel-service-type=LoadBalancer
echo "::endgroup::"

echo "::group::Data plane cluster-azure join"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
FERRY_PEER_KUBECONFIG="${KUBECONFIG_DIR}/cluster-azure.yaml"
echo "FERRY_PEER_KUBECONFIG=${FERRY_PEER_KUBECONFIG}"
echo ferryctl control-plane join cluster-azure --control-plane-reachable=false
ferryctl control-plane join cluster-azure --control-plane-reachable=false
echo "::endgroup::"
