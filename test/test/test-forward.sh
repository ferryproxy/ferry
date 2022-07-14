#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE}")/helpers.sh"

function check-forward-dial() {
  local pid

  echo "::group::Check forward dial"
  check-should-failed "" "" 127.0.0.1:18080 10

  ferryctl local forward dial 0.0.0.0:18080 web-0.test.svc:80 &
  pid=$!

  check "" "" 127.0.0.1:18080 "MESSAGE: web-0" 10

  kill "${pid}"
  echo "::endgroup::"
}

function check-forward-listen() {
  local pid

  echo "::group::Check forward listen"
  check-should-failed cluster-0 web-0 local.test.svc:80 10
  check-should-failed "" "" 127.0.0.1:28080 10

  docker run --name ferry-test-forward-listen -d -p 28080:8080 -e "MESSAGE=local" ghcr.io/wzshiming/echoserver/echoserver:v0.0.1
  ferryctl local forward listen local.test.svc:80 127.0.0.1:28080 &
  pid=$!

  check "" "" 127.0.0.1:28080 "MESSAGE: local" 10
  check cluster-0 web-0 local.test.svc:80 "MESSAGE: local" 10
  kill "${pid}"
  docker rm -f ferry-test-forward-listen
  echo "::endgroup::"
}

wait-tunnel-ready cluster-0

check-forward-dial
stats

check-forward-listen
stats
