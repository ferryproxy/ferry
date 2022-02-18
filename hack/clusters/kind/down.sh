#!/usr/bin/env bash

dir="$(dirname "${BASH_SOURCE}")"
out=$(realpath "${dir}/../../../kubeconfig")

exist=$(kind get clusters)
for name in $(${dir}/list.sh); do
  if [[ $exist != *"$name"* ]]; then
    continue
  fi
  kind delete clusters "${name}"
  rm -f ${out}/"${name}-in-cluster.yaml" ${out}/"${name}" || :
done
