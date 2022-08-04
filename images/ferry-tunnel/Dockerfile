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

FROM docker.io/library/golang:alpine AS builder
WORKDIR /go/src/github.com/ferryproxy/ferry/
COPY . .
ENV CGO_ENABLED=0
RUN apk add git
RUN go install ./cmd/ferry-tunnel-controller
RUN go install ./cmd/ferry-tunnel

FROM ghcr.io/wzshiming/bridge/bridge:v0.8.5
COPY --from=builder /go/bin/ferry-tunnel-controller /usr/local/bin/
COPY --from=builder /go/bin/ferry-tunnel /usr/local/bin/
ENTRYPOINT [ "/usr/local/bin/ferry-tunnel-controller" ]
