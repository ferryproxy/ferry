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

FROM --platform=$TARGETPLATFORM docker.io/library/alpine:3.16
ARG TARGETPLATFORM
COPY --chmod=0755 bin/$TARGETPLATFORM/ferry-tunnel-controller /usr/local/bin/
COPY --chmod=0755 bin/$TARGETPLATFORM/ferry-tunnel /usr/local/bin/

ENTRYPOINT [ "/usr/local/bin/ferry-tunnel-controller" ]
