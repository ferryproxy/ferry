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

echo "::group::Control plane initialization"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl control-plane init "--control-plane-tunnel-address=${HOST_IP}:31000" --enable-register
ferryctl control-plane init "--control-plane-tunnel-address=${HOST_IP}:31000" --enable-register
kubectl --kubeconfig="${KUBECONFIG}" wait --for=condition=Ready pods,hubs.traffic.ferryproxy.io --all -A
echo "::endgroup::"

echo "::group::Data plane cluster-1 initialization"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl data-plane auto cluster-1 "--register-url=http://127.0.0.1:31080/hubs"
ferryctl data-plane auto cluster-1 "--register-url=http://127.0.0.1:31080/hubs"
kubectl --kubeconfig="${KUBECONFIG}" wait --for=condition=Ready pods --all -A
echo "::endgroup::"

echo "::group::Data plane cluster-2 initialization"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-2.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl data-plane auto cluster-2 "--register-url=http://127.0.0.1:31080/hubs"
ferryctl data-plane auto cluster-2 "--register-url=http://127.0.0.1:31080/hubs"
kubectl --kubeconfig="${KUBECONFIG}" wait --for=condition=Ready pods --all -A
echo "::endgroup::"
