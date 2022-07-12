#!/usr/bin/env bash

CURRENT="$(dirname "${BASH_SOURCE}")"
ROOT="$(realpath "${CURRENT}/../..")"
ENVIRONMENT_NAME="${1:-}"

if [ -z "${ENVIRONMENT_NAME}" ]; then
  echo "Usage: ${0} <environment-name>"
  exit 1
fi

ENVIRONMENT_DIR="${ROOT}/environments/${ENVIRONMENT_NAME}"
KUBECONFIG_DIR="${ROOT}/kubeconfigs"

for name in $(ls ${ENVIRONMENT_DIR} | grep -v in-cluster | grep .yaml); do
  name="${name%.*}"
  kubeconfig="${KUBECONFIG_DIR}/${name}.yaml"
  kubectl --kubeconfig "${kubeconfig}" delete -k "${ENVIRONMENT_DIR}/${name}"
  if [[ "${name}" == "control-plane" ]]; then
    kubectl --kubeconfig "${kubeconfig}" delete -k https://github.com/ferryproxy/api/config/crd
  fi
done

