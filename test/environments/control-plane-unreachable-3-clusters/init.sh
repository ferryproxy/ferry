#!/usr/bin/env bash

CURRENT="$(dirname "${BASH_SOURCE}")"
ROOT="$(realpath "${CURRENT}/../..")"
ENVIRONMENT_NAME="${CURRENT##*/}"

ENVIRONMENT_DIR="${ROOT}/environments/${ENVIRONMENT_NAME}"
KUBECONFIG_DIR="${ROOT}/kubeconfigs"

HOST_IP="$(${ROOT}/hack/host-docker-internal.sh)"
echo "Host IP: ${HOST_IP}"

export KUBECONFIG

echo "::group::Control plane initialization"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl control-plane init --control-plane-reachable=false
ferryctl control-plane init --control-plane-reachable=false
echo "::endgroup::"

echo "::group::Data plane cluster-1 join"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl control-plane join cluster-1 "--data-plane-tunnel-address=${HOST_IP}:31001" --control-plane-reachable=false --data-plane-reception-way=control-plane
SEND_TO_CLUSTER_1="$(ferryctl control-plane join cluster-1 "--data-plane-tunnel-address=${HOST_IP}:31001" --control-plane-reachable=false --data-plane-reception-way=control-plane 2>/dev/null)"
echo "::endgroup::"

echo "::group::Data plane cluster-1 join"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_1}"
SEND_TO_CONTROL_PLANE="$(eval "${SEND_TO_CLUSTER_1}" 2>/dev/null)"
echo "::endgroup::"

echo "::group::Controll plane confirm cluster-1 join"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CONTROL_PLANE}"
eval "${SEND_TO_CONTROL_PLANE}"
echo "::endgroup::"

echo "::group::Data plane cluster-2 join"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl control-plane join cluster-2 "--data-plane-tunnel-address=${HOST_IP}:31002" --control-plane-reachable=false --data-plane-reception-way=control-plane
SEND_TO_CLUSTER_2="$(ferryctl control-plane join cluster-2 "--data-plane-tunnel-address=${HOST_IP}:31002" --control-plane-reachable=false --data-plane-reception-way=control-plane 2>/dev/null)"
echo "::endgroup::"

echo "::group::Data plane cluster-2 join"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-2.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_2}"
SEND_TO_CONTROL_PLANE="$(eval "${SEND_TO_CLUSTER_2}" 2>/dev/null)"
echo "::endgroup::"

echo "::group::Controll plane confirm cluster-2 join"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CONTROL_PLANE}"
eval "${SEND_TO_CONTROL_PLANE}"
echo "::endgroup::"