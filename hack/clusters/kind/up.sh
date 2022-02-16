#!/usr/bin/env bash

dir="$(dirname "${BASH_SOURCE}")"
out=$(realpath "${dir}/../../../kubeconfig")

images=(
  "ghcr.io/wzshiming/echoserver/echoserver:v0.0.1"
  "ghcr.io/ferry-proxy/ferry-tunnel:v0.0.9"
)

unset http_proxy https_proxy no_proxy HTTP_PROXY HTTPS_PROXY NO_PROXY

for image in "${images[@]}"; do
  docker inspect "${image}" >/dev/null 2>&1 || docker pull "${image}"
done

mkdir -p "${out}"
for name in $(${dir}/list.sh); do
  if [[ -f "${out}/${name}" ]]; then
    for image in "${images[@]}"; do
      kind load docker-image --name "${name}" "${image}"
    done
    continue
  fi

  config="${dir}/${name}.yaml"
  kind create cluster --name "${name}" --config "${config}" --image docker.io/kindest/node:v1.23.1
  for image in "${images[@]}"; do
    kind load docker-image --name "${name}" "${image}"
  done
  kubectl --context="kind-${name}" config view --minify --raw=true > ${out}/"${name}.yaml"

  ip="$(${dir}/host-docker-internal.sh)"
  echo "Host: ${ip}"
  if [[ "${IN_CLUSTER:-}" == "true" ]]; then
    kubeconfig=$(cat ${out}/"${name}.yaml" | sed "s/127.0.0.1/${ip//[[:space:]]/}/g" | sed 's/certificate-authority-data: .\+/insecure-skip-tls-verify: true/g')
    echo "${kubeconfig}" > ${out}/"${name}"
  else
    cp ${out}/"${name}.yaml" ${out}/"${name}"
  fi
done

