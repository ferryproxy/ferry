#!/usr/bin/env bash

failed=()

function fetch-tunnel-config() {
  local cluster=$1
  echo "==== Fetch ${cluster} config ===="
  kubectl --kubeconfig=kubeconfig/${cluster} exec deploy/ferry-tunnel -n ferry-tunnel-system -- cat bridge.conf
  echo
}

function check(){
  local cluster=$1
  local deploy=$2
  local target=$3
  local wanted=$4
  local got=$(kubectl --kubeconfig=kubeconfig/${cluster} exec deploy/${deploy} -n test -- wget -T 1 -S -O- ${target} 2>&1)
  if [[ "${got}" =~ "${wanted}" ]]; then
    echo "check passed for ${cluster} ${deploy} ${target}"
  else
    failed=("${failed[@]}" "check failed for ${cluster} ${deploy} ${target}")
    echo "check failed for ${cluster} ${deploy} ${target}"
    echo "wanted: ${wanted}"
    echo "got: ${got}"
    return 1
  fi
}

function check-consistency() {
  echo "==== Test data-plane-cluster-1 to control-plane-cluster ===="
  check data-plane-cluster-1 web-1 web-0.test.svc:80 "MESSAGE: cluster-0"
  check data-plane-cluster-1 web-1 web-0.test.svc:8080 "MESSAGE: cluster-0"
  check data-plane-cluster-1 web-1 web-0-0.test.svc:80 "MESSAGE: cluster-0"
  check data-plane-cluster-1 web-1 web-0-0.test.svc:8080 "MESSAGE: cluster-0"

  echo "==== Test control-plane-cluster to data-plane-cluster-1 ===="
  check control-plane-cluster web-0 web-1.test.svc:80 "MESSAGE: cluster-1"
  check control-plane-cluster web-0 web-1.test.svc:8080 "MESSAGE: cluster-1"
  check control-plane-cluster web-0 web-1-1.test.svc:80 "MESSAGE: cluster-1"
  check control-plane-cluster web-0 web-1-1.test.svc:8080 "MESSAGE: cluster-1"

  echo "==== Test data-plane-cluster-2 to control-plane-cluster ===="
  check data-plane-cluster-2 web-2 web-0.test.svc:80 "MESSAGE: cluster-0"
  check data-plane-cluster-2 web-2 web-0.test.svc:8080 "MESSAGE: cluster-0"
  check data-plane-cluster-2 web-2 web-0-0.test.svc:80 "MESSAGE: cluster-0"
  check data-plane-cluster-2 web-2 web-0-0.test.svc:8080 "MESSAGE: cluster-0"

  echo "==== Test control-plane-cluster to data-plane-cluster-2 ===="
  check control-plane-cluster web-0 web-2.test.svc:80 "MESSAGE: cluster-2"
  check control-plane-cluster web-0 web-2.test.svc:8080 "MESSAGE: cluster-2"
  check control-plane-cluster web-0 web-2-2.test.svc:80 "MESSAGE: cluster-2"
  check control-plane-cluster web-0 web-2-2.test.svc:8080 "MESSAGE: cluster-2"
}

function recreate-tunnel() {
  local cluster=$1
  kubectl --kubeconfig=kubeconfig/${cluster} delete pod -n ferry-tunnel-system  --all
}

function wait-tunnel-ready() {
  local cluster=$1

  while [[ $(kubectl --kubeconfig=kubeconfig/${cluster} get pod -n ferry-tunnel-system | grep "Running") == "" ]]; do
    echo "waiting for cluster ${cluster} to be ready"
    sleep 5
  done
  echo "cluster ${cluster} tunnel is ready"
}

wait-tunnel-ready data-plane-cluster-2
wait-tunnel-ready data-plane-cluster-1
wait-tunnel-ready control-plane-cluster


fetch-tunnel-config control-plane-cluster
fetch-tunnel-config data-plane-cluster-1
fetch-tunnel-config data-plane-cluster-2


check-consistency

recreate-tunnel data-plane-cluster-1
wait-tunnel-ready data-plane-cluster-1

sleep 20
check-consistency

recreate-tunnel control-plane-cluster
wait-tunnel-ready control-plane-cluster

sleep 20
check-consistency

if [[ ${#failed[@]} -eq 0 ]]; then
  echo "All checks passed"
  exit 0
else
  echo "Some checks failed"
  for i in "${failed[@]}"; do
    echo "${i}"
  done
  exit 1
fi
