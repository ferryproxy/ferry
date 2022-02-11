#!/usr/bin/env bash

KUBECONFIG=./kubeconfig/control-plane-cluster go run ./cmd/controller
