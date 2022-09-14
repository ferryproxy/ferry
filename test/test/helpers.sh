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


failed=()

CURRENT="$(dirname "${BASH_SOURCE[0]}")"
ROOT="$(realpath "${CURRENT}/..")"

KUBECONFIG_DIR="${ROOT}/kubeconfigs"

function resource-apply() {
  local cluster=$1
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" apply -f -
}

function resource-delete() {
  local cluster=$1
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" delete -f -
}

function fetch-routepolicy() {
  local cluster=$1
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" get routepolicy.traffic.ferryproxy.io -n ferry-system
}

function fetch-route() {
  local cluster=$1
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" get route.traffic.ferryproxy.io -n ferry-system
}

function fetch-tunnel-config() {
  local cluster=$1
  echo "==== Fetch ${cluster} tunnel config ===="
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" exec deploy/ferry-tunnel -n ferry-tunnel-system -- cat /var/ferry/bridge.conf
  echo
}

function fetch-tunnel-log() {
  local cluster=$1
  echo "==== Fetch ${cluster} tunnel log ===="
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" logs --prefix -f deploy/ferry-tunnel -n ferry-tunnel-system
}

function fetch-controller-log() {
  local cluster=$1
  echo "==== Fetch ${cluster} controller log ===="
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" logs --prefix -f deploy/ferry -n ferry-system
}

function check() {
  local cluster=$1
  local deploy=$2
  local target=$3
  local wanted=$4
  local retry=$5
  local got
  if [[ "${deploy}" != "" && "${cluster}" != "" ]]; then
    got=$(kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" exec "deploy/${deploy}" -n test -- wget -T 1 -S -O- "${target}" 2>&1)
    if [[ $? == 0 && "${got}" =~ "${wanted}" ]]; then
      echo "check passed for ${cluster} ${deploy} ${target}"
    elif [[ "${retry}" -gt 0 ]]; then
      echo "check failed for ${target}, retry again later"
      sleep 10
      check "${cluster}" "${deploy}" "${target}" "${wanted}" "$((retry - 1))"
    else
      failed=("${failed[@]}" "check failed for ${cluster} ${deploy} ${target}")
      echo "check failed for ${cluster} ${deploy} ${target}"
      echo "wanted: ${wanted}"
      echo "got: ${got}"
      return 1
    fi
  else
    got=$(wget -T 1 -S -O- "${target}")
    if [[ $? == 0 && "${got}" =~ "${wanted}" ]]; then
      echo "check passed for ${target}"
    elif [[ "${retry}" -gt 0 ]]; then
      echo "check failed for ${target}, retry again later"
      sleep 10
      check "${cluster}" "${deploy}" "${target}" "${wanted}" "$((retry - 1))"
    else
      failed=("${failed[@]}" "check failed for ${target}")
      echo "check failed for ${target}"
      echo "wanted: ${wanted}"
      echo "got: ${got}"
      return 1
    fi
  fi
}

function check-should-failed() {
  local cluster=$1
  local deploy=$2
  local target=$3
  local retry=$4
  if [[ "${deploy}" != "" && "${cluster}" != "" ]]; then
    kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" exec "deploy/${deploy}" -n test -- wget -T 1 -S -O- "${target}" 2>&1
    if [[ $? != 0 ]]; then
      echo "check-should-failed passed for ${cluster} ${deploy} ${target}"
    elif [[ "${retry}" -gt 0 ]]; then
      echo "check-should-failed failed for ${target}, retry again later"
      sleep 10
      check-should-failed "${cluster}" "${deploy}" "${target}" "$((retry - 1))"
    else
      failed=("${failed[@]}" "check should failed for ${cluster} ${deploy} ${target}")
      echo "check-should-failed failed for ${cluster} ${deploy} ${target}"
      return 1
    fi
  else
    wget -T 1 -S -O- "${target}"
    if [[ $? != 0 ]]; then
      echo "check-should-failed passed for ${target}"
    elif [[ "${retry}" -gt 0 ]]; then
      echo "check-should-failed failed for ${target}, retry again later"
      sleep 10
      check-should-failed "${cluster}" "${deploy}" "${target}" "$((retry - 1))"
    else
      failed=("${failed[@]}" "check should failed for ${target}")
      echo "check-should-failed failed for ${target}"
      return 1
    fi
  fi
}

function recreate-tunnel() {
  local cluster=$1
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" delete pod -n ferry-tunnel-system --all
  wait-tunnel-ready "${cluster}"
}

function wait-tunnel-ready() {
  local cluster=$1

  while [[ $(kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" get pod -n ferry-tunnel-system | grep "Running") == "" ]]; do
    echo "waiting for cluster ${cluster} tunnel to be ready"
    sleep 5
  done
  echo "cluster ${cluster} tunnel is ready"
}

function recreate-controller() {
  local cluster=$1
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" delete pod -n ferry-system --all
}

function wait-controller-ready() {
  local cluster=$1

  while [[ $(kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" get pod -n ferry-system | grep "Running") == "" ]]; do
    echo "waiting for cluster ${cluster} controller to be ready"
    sleep 5
  done
  echo "cluster ${cluster} controller is ready"
}

function show-cluster-info() {
  local cluster=$1
  echo "==== Fetch ${cluster} cluster information ===="
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" get cm,pod,svc,ep,node -A
}

function show-hub() {
  local cluster=$1
  echo "==== Fetch ${cluster} ferry information ===="
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" get hub.traffic.ferryproxy.io,routepolicies.traffic.ferryproxy.io -A
}

function stats() {
  if [[ ${#failed[@]} -eq 0 ]]; then
    echo "All checks passed"
    failed=()
  else
    echo "Some checks failed"
    for i in "${failed[@]}"; do
      echo "${i}"
    done
    exit 1
  fi
}
