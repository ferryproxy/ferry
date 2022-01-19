#!/usr/bin/env bash

dir="$(dirname "${BASH_SOURCE}")"
out=$(realpath "${dir}/../../../kubeconfig")

mkdir -p "${out}"
for name in $(${dir}/list.sh); do
  if [[ -f "${out}/${name}" ]]; then
    continue
  fi

  kind create cluster --name "${name}" --config "${config}" --image docker.io/kindest/node:v1.23.1
  kubectl --context="kind-${name}" config view --minify --raw=true > ${out}/"${name}.yaml"

  ip="$(${dir}/host-docker-internal.sh)"
  echo "Host: ${ip}"
  kubeconfig=$(cat ${out}/"${name}.yaml" | sed "s/127.0.0.1/${ip//[[:space:]]/}/g" | sed 's/certificate-authority-data: .\+/insecure-skip-tls-verify: true/g')
  echo "${kubeconfig}" > ${out}/"${name}"
done

