#!/usr/bin/env bash

failed=()

ROOT="$(dirname "${BASH_SOURCE}")/.."

function resource-apply() {
    local cluster=$1
    kubectl --kubeconfig="${ROOT}/kubeconfig/${cluster}" apply -f -
}

function fetch-tunnel-config() {
  local cluster=$1
  echo "==== Fetch ${cluster} tunnel config ===="
  kubectl --kubeconfig="${ROOT}/kubeconfig/${cluster}" exec deploy/ferry-tunnel -n ferry-tunnel-system -- cat bridge.conf
  echo
}

function fetch-tunnel-log() {
  local cluster=$1
  echo "==== Fetch ${cluster} tunnel log ===="
  kubectl --kubeconfig="${ROOT}/kubeconfig/${cluster}" logs deploy/ferry-tunnel -n ferry-tunnel-system
}

function fetch-controller-log() {
  local cluster=$1
  echo "==== Fetch ${cluster} controller log ===="
  kubectl --kubeconfig="${ROOT}/kubeconfig/${cluster}" logs deploy/ferry -n ferry-system
}

function check(){
  local cluster=$1
  local deploy=$2
  local target=$3
  local wanted=$4
  local got=$(kubectl --kubeconfig="${ROOT}/kubeconfig/${cluster}" exec deploy/${deploy} -n test -- wget -T 1 -S -O- ${target} 2>&1)
  if [[ "${got}" =~ "${wanted}" ]]; then
    echo "check passed for ${cluster} ${deploy} ${target} : ${NAME:-}"
  else
    failed=("${failed[@]}" "check failed for ${cluster} ${deploy} ${target} : ${NAME:-}")
    echo "check failed for ${cluster} ${deploy} ${target} : ${NAME:-}"
    echo "wanted: ${wanted}"
    echo "got: ${got}"
    return 1
  fi
}

function recreate-tunnel() {
  local cluster=$1
  kubectl --kubeconfig="${ROOT}/kubeconfig/${cluster}" delete pod -n ferry-tunnel-system  --all
}

function wait-tunnel-ready() {
  local cluster=$1

  while [[ $(kubectl --kubeconfig="${ROOT}/kubeconfig/${cluster}" get pod -n ferry-tunnel-system | grep "Running") == "" ]]; do
    echo "waiting for cluster ${cluster} tunnel to be ready"
    sleep 5
  done
  echo "cluster ${cluster} tunnel is ready"
}

function recreate-controller() {
  local cluster=$1
  kubectl --kubeconfig="${ROOT}/kubeconfig/${cluster}" delete pod -n ferry-system  --all
}

function wait-controller-ready() {
  local cluster=$1

  while [[ $(kubectl --kubeconfig="${ROOT}/kubeconfig/${cluster}" get pod -n ferry-system | grep "Running") == "" ]]; do
    echo "waiting for cluster ${cluster} controller to be ready"
    sleep 5
  done
  echo "cluster ${cluster} controller is ready"
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
