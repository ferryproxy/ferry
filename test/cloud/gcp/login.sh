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

PROJECT_ID=$1

CRED_DATA=$2

if [ "${PROJECT_ID}" == "" ] && [ "${GCP_PROJECT_ID}" != "" ]; then
  PROJECT_ID="${GCP_PROJECT_ID}"
fi

if [ "${CRED_DATA}" == "" ] && [ "${GCP_CRED_DATA}" != "" ]; then
  CRED_DATA="${GCP_CRED_DATA}"
fi

if [ "${PROJECT_ID}" == "" ] || [ "${CRED_DATA}" == "" ]; then
  echo "Usage: ${0} <project-id> <credentials-data-base64>"
  exit 1
fi

TMPFILE="GCP_CRED_DATA.tmp"

echo "${CRED_DATA}" | base64 --decode > "${TMPFILE}"

gcloud auth login --cred-file="${TMPFILE}"
rm -f "${TMPFILE}"

gcloud projects list --quiet

gcloud config set project "${PROJECT_ID}"
