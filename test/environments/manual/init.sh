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

echo "::group::Data plane initialization"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl data-plane init
ferryctl data-plane init
echo "::endgroup::"
