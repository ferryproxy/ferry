#!/usr/bin/env bash

CURRENT="$(dirname "${BASH_SOURCE}")"
ROOT="$(realpath "${CURRENT}/../..")"
ENVIRONMENT_NAME="${CURRENT##*/}"

ENVIRONMENT_DIR="${ROOT}/environments/${ENVIRONMENT_NAME}"
KUBECONFIG_DIR="${ROOT}/kubeconfigs"

HOST_IP="$(${ROOT}/hack/host-docker-internal.sh)"
echo "Host IP: ${HOST_IP}"

export KUBECONFIG
export FERRY_TUNNEL_IMAGE

FERRY_TUNNEL_IMAGE=ferry-tunnel:test

echo "::group::Cluster cluster-0 export web-0 80"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
ferryctl data-plane init
echo ferryctl local manual export --reachable=true "--tunnel-address=${HOST_IP}:31000" --export-host-port=web-0.test.svc:80 --import-service-name=web-0-80
SEND_TO_CLUSTER_1="$(ferryctl local manual export --reachable=true "--tunnel-address=${HOST_IP}:31000" --export-host-port=web-0.test.svc:80 --import-service-name=web-0-80 2>/dev/null)"
echo "::endgroup::"

echo "::group::Cluster cluster-1 import web-0 80"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_1}"
SEND_TO_CLUSTER_0="$(eval "${SEND_TO_CLUSTER_1}" 2>/dev/null)"
echo "::endgroup::"

echo "::group::Cluster shared web-0 80"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_0}"
eval "${SEND_TO_CLUSTER_0}" 2>/dev/null
echo "::endgroup::"

echo "::group::Cluster cluster-1 export web-1 80"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl local manual export --reachable=false "--peer-tunnel-address=${HOST_IP}:31000" --export-host-port=web-1.test.svc:80 --import-service-name=web-1-80
SEND_TO_CLUSTER_0="$(ferryctl local manual export --reachable=false "--peer-tunnel-address=${HOST_IP}:31000" --export-host-port=web-1.test.svc:80 --import-service-name=web-1-80 2>/dev/null)"
echo "::endgroup::"

echo "::group::Cluster cluster-0 import web-1 80"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_0}"
SEND_TO_CLUSTER_1="$(eval "${SEND_TO_CLUSTER_0}" 2>/dev/null)"
echo "::endgroup::"

echo "::group::Cluster shared web-1 80"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_1}"
eval "${SEND_TO_CLUSTER_1}" 2>/dev/null
echo "::endgroup::"

echo "::group::Cluster cluster-0 import web-1 8080"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
ferryctl data-plane init
echo ferryctl local manual import --reachable=true "--tunnel-address=${HOST_IP}:31000" --export-host-port=web-1.test.svc:8080 --import-service-name=web-1-8080
SEND_TO_CLUSTER_1="$(ferryctl local manual import --reachable=true "--tunnel-address=${HOST_IP}:31000" --export-host-port=web-1.test.svc:8080 --import-service-name=web-1-8080 2>/dev/null)"
echo "::endgroup::"

echo "::group::Cluster cluster-1 export web-1 8080"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_1}"
SEND_TO_CLUSTER_0="$(eval "${SEND_TO_CLUSTER_1}" 2>/dev/null)"
echo "::endgroup::"

echo "::group::Cluster shared web-1 8080"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_0}"
eval "${SEND_TO_CLUSTER_0}" 2>/dev/null
echo "::endgroup::"

echo "::group::Cluster cluster-1 import web-0 8080"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo ferryctl local manual import --reachable=false "--peer-tunnel-address=${HOST_IP}:31000" --export-host-port=web-0.test.svc:8080 --import-service-name=web-0-8080
SEND_TO_CLUSTER_0="$(ferryctl local manual import --reachable=false "--peer-tunnel-address=${HOST_IP}:31000" --export-host-port=web-0.test.svc:8080 --import-service-name=web-0-8080 2>/dev/null)"
echo "::endgroup::"

echo "::group::Cluster cluster-0 export web-0 8080"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-0.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_0}"
SEND_TO_CLUSTER_1="$(eval "${SEND_TO_CLUSTER_0}" 2>/dev/null)"
echo "::endgroup::"

echo "::group::Cluster shared web-0 8080"
KUBECONFIG="${KUBECONFIG_DIR}/cluster-1.yaml"
echo "KUBECONFIG=${KUBECONFIG}"
echo "${SEND_TO_CLUSTER_1}"
eval "${SEND_TO_CLUSTER_1}" 2>/dev/null
echo "::endgroup::"
