#!/usr/bin/env bash


CURRENT="$(dirname "${BASH_SOURCE}")"
ROOT="$(realpath "${CURRENT}/..")"
ENVIRONMENT_NAME="${1:-}"

ENVIRONMENT_DIR="${ROOT}/environments/${ENVIRONMENT_NAME}"

"${ROOT}/hack/kind/up.sh" "${ENVIRONMENT_NAME}"
"${ROOT}/hack/clusters/up.sh" "${ENVIRONMENT_NAME}"

if [ -f "${ENVIRONMENT_DIR}/init.sh" ]; then
  "${ENVIRONMENT_DIR}/init.sh"
fi