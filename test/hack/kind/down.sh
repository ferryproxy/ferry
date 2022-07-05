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
  kind delete clusters "${ENVIRONMENT_NAME}-${name}" || :
  rm -f "${KUBECONFIG_DIR}/${name}.yaml" "${KUBECONFIG_DIR}/${name}-in-cluster.yaml" || :
done
