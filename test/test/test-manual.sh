#!/usr/bin/env bash

source "$(dirname "${BASH_SOURCE}")/helpers.sh"

function check-forward-dial() {
  local pid

  echo "::group::Check forward dial"
  check-should-failed "" "" 127.0.0.1:18080

  ferryctl local forward dial 0.0.0.0:18080 web-0.test.svc:80 &
  pid=$!
  sleep 10

  check "" "" 127.0.0.1:18080 "MESSAGE: cluster-0"

  kill "${pid}"
  echo "::endgroup::"
}

function check-forward-listen() {
  local pid

  echo "::group::Check forward listen"
  check-should-failed cluster-0 web-0 local.test.svc:80 "MESSAGE: local"

  docker run --name ferry-test-forward-listen -d -p 28080:8080 -e "MESSAGE=local" ghcr.io/wzshiming/echoserver/echoserver:v0.0.1
  ferryctl local forward listen local.test.svc:80 127.0.0.1:28080 &
  pid=$!
  sleep 20

  check cluster-0 web-0 local.test.svc:80 "MESSAGE: local"
  kill "${pid}"
  docker rm -f ferry-test-forward-listen
  echo "::endgroup::"
}

check-forward-dial
stats

sleep 5

check-forward-listen
stats
