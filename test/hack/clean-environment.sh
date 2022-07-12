#!/usr/bin/env bash


CURRENT="$(dirname "${BASH_SOURCE}")"
ROOT="$(realpath "${CURRENT}/..")"
ENVIRONMENT_NAME="${1:-}"

"${ROOT}/hack/kind/down.sh" "${ENVIRONMENT_NAME}"
