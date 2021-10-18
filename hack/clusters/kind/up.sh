#!/usr/bin/env bash

dir="$(dirname "${BASH_SOURCE}")"
for config in $(ls "${dir}"/*.yaml); do
  name="${config%.*}"
  name="${name##*/}"
  kind create cluster --name "${name}" --config "${config}" --image ghcr.io/mirrorshub/kindest/node:v1.22.2
done
