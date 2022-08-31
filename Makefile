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

GO_CMD ?= go

DRY_RUN ?=
PUSH ?=

GH_RELEASE ?=

VERSION ?= $(shell git describe --tags --dirty --always)

BASE_REF ?= $(shell git rev-parse --abbrev-ref HEAD)

EXTRA_TAGS ?=

BINARY ?= ferryctl

IMAGE_BINARY ?= ferry-controller ferry-tunnel ferry-tunnel-controller

IMAGE_PREFIX ?= ghcr.io/ferryproxy/ferry

CONTROLLER_IMAGE ?= $(IMAGE_PREFIX)/ferry-controller

TUNNEL_IMAGE ?= $(IMAGE_PREFIX)/ferry-tunnel

IMAGE_PLATFORMS ?= linux/amd64 linux/arm64

BINARY_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

DOCKER_CLI_EXPERIMENTAL ?= enabled

.PHONY: default
default: help

vendor:
	$(GO_CMD) mod vendor

## unit-test: Run unit tests
.PHONY: unit-test
unit-test: vendor
	$(GO_CMD) test ./pkg/...

## verify: Verify code
.PHONY: verify
verify:
	@./hack/verify-all.sh

## build: Build binary
.PHONY: build
build: vendor
	@./hack/releases.sh \
		$(addprefix --bin=, $(BINARY)) \
		$(addprefix --extra-tag=, $(EXTRA_TAGS)) \
		--gh-release=${GH_RELEASE} \
		--image-prefix=${IMAGE_PREFIX} \
		--version=${VERSION} \
		--dry-run=${DRY_RUN} \
		--push=${PUSH}

## cross-build: Build all supported platforms
.PHONY: cross-build
cross-build: vendor
	@./hack/releases.sh \
		$(addprefix --bin=, $(BINARY)) \
		$(addprefix --platform=, $(BINARY_PLATFORMS)) \
		$(addprefix --extra-tag=, $(EXTRA_TAGS)) \
		--gh-release=${GH_RELEASE} \
		--image-prefix=${IMAGE_PREFIX} \
		--version=${VERSION} \
		--dry-run=${DRY_RUN} \
		--push=${PUSH}

## image: Build image
.PHONY: image
image:
	@./hack/releases.sh \
		$(addprefix --bin=, $(IMAGE_BINARY)) \
		--gh-release=${GH_RELEASE} \
		--image-prefix=${IMAGE_PREFIX} \
		--version=${VERSION} \
		--dry-run=${DRY_RUN}
	@./images/ferry-controller/build.sh \
		$(addprefix --extra-tag=, $(EXTRA_TAGS)) \
		--image=${CONTROLLER_IMAGE} \
		--version=${VERSION} \
		--dry-run=${DRY_RUN} \
		--push=${PUSH}
	@./images/ferry-tunnel/build.sh \
		$(addprefix --extra-tag=, $(EXTRA_TAGS)) \
		--image=${TUNNEL_IMAGE} \
		--version=${VERSION} \
		--dry-run=${DRY_RUN} \
		--push=${PUSH}

## cross-image: Build images for all supported platforms
.PHONY: cross-image
cross-image:
	@./hack/releases.sh \
		$(addprefix --bin=, $(IMAGE_BINARY)) \
		$(addprefix --platform=, $(IMAGE_PLATFORMS)) \
		--gh-release=${GH_RELEASE} \
		--image-prefix=${IMAGE_PREFIX} \
		--version=${VERSION} \
		--dry-run=${DRY_RUN}
	@./images/ferry-controller/build.sh \
		$(addprefix --platform=, $(IMAGE_PLATFORMS))  \
		$(addprefix --extra-tag=, $(EXTRA_TAGS)) \
		--image=${CONTROLLER_IMAGE} \
		--version=${VERSION} \
		--dry-run=${DRY_RUN} \
		--push=${PUSH}
	@./images/ferry-tunnel/build.sh \
		$(addprefix --platform=, $(IMAGE_PLATFORMS))  \
		$(addprefix --extra-tag=, $(EXTRA_TAGS)) \
		--image=${TUNNEL_IMAGE} \
		--version=${VERSION} \
		--dry-run=${DRY_RUN} \
		--push=${PUSH}

## help: Show this help message
.PHONY: help
help:
	@cat $(MAKEFILE_LIST) | grep -e '^## ' | sed -e 's/^## //'
