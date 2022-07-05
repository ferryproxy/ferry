#!/usr/bin/env bash

CURRENT="$(dirname "${BASH_SOURCE}")"
ROOT="$(realpath "${CURRENT}/../..")"
ENVIRONMENT_NAME="${CURRENT##*/}"

ENVIRONMENT_DIR="${ROOT}/environments/${ENVIRONMENT_NAME}"
KUBECONFIG_DIR="${ROOT}/kubeconfigs"

HOST_IP="$(${ROOT}/hack/host-docker-internal.sh)"
echo "Host IP: ${HOST_IP}"

export KUBECONFIG
export FERRY_CONTROLLER_IMAGE
export FERRY_TUNNEL_IMAGE

FERRY_CONTROLLER_IMAGE=ferry-controller:test
FERRY_TUNNEL_IMAGE=ferry-tunnel:test

echo "::group::Control plane initialization"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl control-plane init "--control-plane-tunnel-address=${HOST_IP}:31000"
ferryctl control-plane init "--control-plane-tunnel-address=${HOST_IP}:31000"
echo "::endgroup::"

echo "::group::Data plane cluster-1 pre-join"
KUBECONFIG="${KUBECONFIG_DIR}/control-plane.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl control-plane pre-join direct cluster-1 "--data-plane-tunnel-address=${HOST_IP}:31001" "--data-plane-apiserver-address=${HOST_IP}:32001"
SEND_TO_CLUSTER_1="$(ferryctl control-plane pre-join direct cluster-1 "--data-plane-tunnel-address=${HOST_IP}:31001" "--data-plane-apiserver-address=${HOST_IP}:32001" 2>/dev/null)"
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
