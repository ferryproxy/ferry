#!/usr/bin/env bash

ROOT="$(dirname "${BASH_SOURCE}")/.."

image="ghcr.io/ferry-proxy/ferry:dev"

docker build -t "${image}" "${ROOT}"
kind load docker-image --name control-plane-cluster "${image}"

KUBECONFIG="${ROOT}/kubeconfig/control-plane-cluster" kubectl apply -k "${ROOT}/test/deploy"
