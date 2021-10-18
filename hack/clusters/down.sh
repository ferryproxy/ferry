#!/usr/bin/env bash

dir="$(dirname "${BASH_SOURCE}")"
${dir}/kind/down.sh

out="${dir}/../../kubeconfig"
mkdir -p ${out}
for name in $(${dir}/kind/list.sh); do
  rm ${out}/"${name}" || :
done
