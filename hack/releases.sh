#!/usr/bin/env bash
# Copyright 2022 FerryProxy Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

DIR="$(dirname "${BASH_SOURCE[0]}")"

ROOT_DIR="$(realpath "${DIR}/..")"

GH_RELEASE=""
IMAGE_PREFIX=""
VERSION=""
DRY_RUN=false
PUSH=false
BINS=()
EXTRA_TAGS=()
PLATFORMS=()
LDFLAGS=()

function usage() {
  echo "Usage: ${0} [--help] [--bin <bin> ...] [--extra-tag <extra-tag> ...] [--platform <platform> ...] [--image-prefix <image-prefix>] [--version <version>] [--push] [--dry-run]"
  echo "  --bin <bin> is binary, is required"
  echo "  --extra-tag <extra-tag> is extra tag"
  echo "  --platform <platform> is multi-platform capable for binary"
  echo "  --gh-release <gh-release> is github release"
  echo "  --image-prefix <image-prefix> is image prefix"
  echo "  --version <version> is version of binary"
  echo "  --push will push binary to gh release"
  echo "  --dry-run just show what would be done"
}

function args() {
  local arg
  while [[ $# -gt 0 ]]; do
    arg="$1"
    case "${arg}" in
    --bin | --bin=*)
      [[ "${arg#*=}" != "${arg}" ]] && BINS+=("${arg#*=}") || { BINS+=("${2}") && shift; }
      shift
      ;;
    --extra-tag | --extra-tag=*)
      [[ "${arg#*=}" != "${arg}" ]] && EXTRA_TAGS+=("${arg#*=}") || { EXTRA_TAGS+=("${2}") && shift; }
      shift
      ;;
    --platform | --platform=*)
      [[ "${arg#*=}" != "${arg}" ]] && PLATFORMS+=("${arg#*=}") || { PLATFORMS+=("${2}") && shift; }
      shift
      ;;
    --gh-release | --gh-release=*)
      [[ "${arg#*=}" != "${arg}" ]] && GH_RELEASE="${arg#*=}" || { GH_RELEASE="${2}" && shift; }
      shift
      ;;
    --image-prefix | --image-prefix=*)
      [[ "${arg#*=}" != "${arg}" ]] && IMAGE_PREFIX="${arg#*=}" || { IMAGE_PREFIX="${2}" && shift; }
      shift
      ;;
    --version | --version=*)
      [[ "${arg#*=}" != "${arg}" ]] && VERSION="${arg#*=}" || { VERSION="${2}" && shift; }
      shift
      ;;
    --push | --push=*)
      [[ "${arg#*=}" != "${arg}" ]] && PUSH="${arg#*=}" || PUSH="true"
      shift
      ;;
    --dry-run | --dry-run=*)
      [[ "${arg#*=}" != "${arg}" ]] && DRY_RUN="${arg#*=}" || DRY_RUN="true"
      shift
      ;;
    --help)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: ${arg}"
      usage
      exit 1
      ;;
    esac
  done

  if [[ "${#BINS}" -eq 0 ]]; then
    echo "--bin is required"
    usage
    exit 1
  fi

  if [[ "${#PLATFORMS}" -eq 0 ]]; then
    PLATFORMS+=(
      linux/amd64
    )
  fi
}

function dry_run() {
  echo "${@}"
  if [[ "${DRY_RUN}" != "true" ]]; then
    eval "${@}"
  fi
}

function main() {
  local os
  local dist
  local src
  local bin
  local tmp_bin
  local extra_args=()

  if [[ "${VERSION}" != "" ]]; then
    LDFLAGS+=("-X github.com/ferryproxy/ferry/pkg/consts.Version=${VERSION}")
  fi
  if [[ "${IMAGE_PREFIX}" != "" ]]; then
    LDFLAGS+=("-X github.com/ferryproxy/ferry/pkg/consts.ImagePrefix=${IMAGE_PREFIX}")
  fi

  if [[ "${#LDFLAGS}" -gt 0 ]]; then
    extra_args+=("-ldflags" "'${LDFLAGS[*]}'")
  fi

  for platform in "${PLATFORMS[@]}"; do
    os="${platform%%/*}"
    for binary in "${BINS[@]}"; do
      bin="${binary}"
      if [[ "${os}" == "windows" ]]; then
        bin="${bin}.exe"
      fi
      dist="./bin/${platform}/${bin}"
      src="./cmd/${binary}"
      CGO_ENABLED=0 dry_run GOOS="${platform%%/*}" GOARCH="${platform##*/}" go build "${extra_args[@]}" -o "${dist}" "${src}"
      if [[ "${PUSH}" == "true" ]]; then
        if [[ "${GH_RELEASE}" != "" ]]; then
          tmp_bin="${binary}-${platform%%/*}-${platform##*/}"
          if [[ "${os}" == "windows" ]]; then
            tmp_bin="${tmp_bin}.exe"
          fi
          dry_run cp "${dist}" "${tmp_bin}"
          dry_run gh -R "${GH_RELEASE}" release upload "${VERSION}" "${tmp_bin}"
          if [[ "${#EXTRA_TAGS}" -ne 0 ]]; then
            for extra_tag in "${EXTRA_TAGS[@]}"; do
              dry_run gh -R "${GH_RELEASE}" release upload "${extra_tag}" "${tmp_bin}"
            done
          fi
        fi
      fi
    done
  done
}

args "$@"

cd "${ROOT_DIR}" && main
