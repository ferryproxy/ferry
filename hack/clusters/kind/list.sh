#!/usr/bin/env bash

dir="$(dirname "${BASH_SOURCE}")"
for config in $(ls "${dir}"/*.yaml); do
  name="${config%.*}"
  name="${name##*/}"
  echo "${name}"
done
