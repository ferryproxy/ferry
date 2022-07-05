#!/usr/bin/env bash

failed=()

CURRENT="$(dirname "${BASH_SOURCE}")"
ROOT="$(realpath "${CURRENT}/..")"

KUBECONFIG_DIR="${ROOT}/kubeconfigs"

function resource-apply() {
  local cluster=$1
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" apply -f -
}

function fetch-mapping-rule() {
  local cluster=$1
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" get mappingrule.ferry.zsm.io -n ferry-system
}

function fetch-tunnel-config() {
  local cluster=$1
  echo "==== Fetch ${cluster} tunnel config ===="
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" exec deploy/ferry-tunnel -n ferry-tunnel-system -- cat bridge.conf
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
  local got
  if [[ "${deploy}" != "" && "${cluster}" != "" ]]; then
    got=$(kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" exec "deploy/${deploy}" -n test -- wget -T 1 -S -O- "${target}" 2>&1)
    if [[ $? == 0 && "${got}" =~ "${wanted}" ]]; then
      echo "check passed for ${cluster} ${deploy} ${target}"
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
  if [[ "${deploy}" != "" && "${cluster}" != "" ]]; then
    kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" exec "deploy/${deploy}" -n test -- wget -T 1 -S -O- "${target}" 2>&1
    if [[ $? == 0 ]]; then
      failed=("${failed[@]}" "check should failed for ${cluster} ${deploy} ${target}")
      echo "check-should-failed failed for ${cluster} ${deploy} ${target}"
      return 1
    else
      echo "check-should-failed passed for ${cluster} ${deploy} ${target}"
    fi
  else
    wget -T 1 -S -O- "${target}"
    if [[ $? == 0 ]]; then
      failed=("${failed[@]}" "check should failed for ${target}")
      echo "check-should-failed failed for ${target}"
      return 1
    else
      echo "check-should-failed passed for ${target}"
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

function show-ferry-info() {
  local cluster=$1
  echo "==== Fetch ${cluster} ferry information ===="
  kubectl --kubeconfig="${KUBECONFIG_DIR}/${cluster}.yaml" get clusterinformations.ferry.zsm.io,ferrypolicies.ferry.zsm.io -A
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
