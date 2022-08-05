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

CURRENT_DIR="$(dirname "${BASH_SOURCE[0]}")"

CLUSTER_NAME=$1

REGION_ID=$2

ZONE_ID=$3

if [ "${REGION_ID}" == "" ] && [ "${AWS_REGION_ID}" != "" ]; then
  REGION_ID="${AWS_REGION_ID}"
fi

if [ "${REGION_ID}" == "" ]; then
  REGION_ID="us-east-1"
fi

if [ "${CLUSTER_NAME}" == "" ]; then
  echo "Usage: ${0} <cluster-name> [region-id]"
  exit 1
fi

region_code="${REGION_ID}"
cluster_name="${CLUSTER_NAME}"
account_id=$(aws sts get-caller-identity | jq -r '.Account')

cluster_endpoint=$(aws eks describe-cluster \
  --region $region_code \
  --name $cluster_name \
  --query "cluster.endpoint" \
  --output text)

certificate_data=$(aws eks describe-cluster \
  --region $region_code \
  --name $cluster_name \
  --query "cluster.certificateAuthority.data" \
  --output text)

cat <<EOF
apiVersion: v1
clusters:
- cluster:
    certificate-authority-data: $certificate_data
    server: $cluster_endpoint
  name: arn:aws:eks:$region_code:$account_id:cluster/$cluster_name
contexts:
- context:
    cluster: arn:aws:eks:$region_code:$account_id:cluster/$cluster_name
    user: arn:aws:eks:$region_code:$account_id:cluster/$cluster_name
  name: arn:aws:eks:$region_code:$account_id:cluster/$cluster_name
current-context: arn:aws:eks:$region_code:$account_id:cluster/$cluster_name
kind: Config
preferences: {}
users:
- name: arn:aws:eks:$region_code:$account_id:cluster/$cluster_name
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args:
        - --region
        - $region_code
        - eks
        - get-token
        - --cluster-name
        - $cluster_name
EOF
