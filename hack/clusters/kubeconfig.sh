#!/usr/bin/env bash

dir="$(dirname "${BASH_SOURCE}")"

out="${dir}/../../kubeconfig"
mkdir -p ${out}
for name in $(${dir}/kind/list.sh); do
  kubectl --context="kind-${name}" config view --minify --raw=true > ${out}/"${name}"
done

