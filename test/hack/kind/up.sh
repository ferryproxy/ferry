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

KIND_IMAGE="docker.io/kindest/node:v1.22.5"

images=(
  "ghcr.io/wzshiming/echoserver/echoserver:v0.0.1"
)

ferry_image="$(ferryctl --help | grep ' image (default ' | grep -o '".\+"' | tr -d '"')"

for image in ${ferry_image}; do
  images+=("${image}")
done

HOST_IP="$(${ROOT}/hack/host-docker-internal.sh)"
echo "Host IP: ${HOST_IP}"

for image in "${images[@]}"; do
  docker inspect "${image}" >/dev/null 2>&1 || docker pull "${image}"
done

mkdir -p "${KUBECONFIG_DIR}"
for name in $(ls ${ENVIRONMENT_DIR} | grep -v in-cluster | grep .yaml); do
  name="${name%.*}"
  env_name="ferry-test-${name}"
  if [[ ! -f "${KUBECONFIG_DIR}/${name}-in-cluster.yaml" ]]; then
    echo kind create cluster --name "${env_name}" --config "${ENVIRONMENT_DIR}/${name}.yaml" --image "${KIND_IMAGE}"
    kind create cluster --name "${env_name}" --config "${ENVIRONMENT_DIR}/${name}.yaml" --image "${KIND_IMAGE}"
    echo kubectl --context="kind-${env_name}" config view --minify --raw=true
    kubectl --context="kind-${env_name}" config view --minify --raw=true >"${KUBECONFIG_DIR}/${name}-raw.yaml"

    cat "${KUBECONFIG_DIR}/${name}-raw.yaml" |
      sed "s/0\.0\.0\.0/127.0.0.1/g" |
      sed 's/certificate-authority-data: .\+/insecure-skip-tls-verify: true/g' >"${KUBECONFIG_DIR}/${name}.yaml"
    cat "${KUBECONFIG_DIR}/${name}-raw.yaml" |
      sed "s/0\.0\.0\.0/${HOST_IP//[[:space:]]/}/g" |
      sed 's/certificate-authority-data: .\+/insecure-skip-tls-verify: true/g' >"${KUBECONFIG_DIR}/${name}-in-cluster.yaml"
  fi

  for image in "${images[@]}"; do
    kind load docker-image --name "${env_name}" "${image}"
  done
done
