# Cloud

The following are the Kubernetes services for various public cloud platforms
that Ferry has tested and validated. If there are multiple Kubernetes services 
on the same platform, we prefer to choose the one that does not require a management server.

- Aliyun (Alibaba Cloud)
  [ ] ACK (Managed Kubernetes Cluster Service)
  [x] ASK (Serverless Kubernetes Cluster Service)
- AWS (Amazon Web Services)
  [x] EKS (Elastic Kubernetes Engine)
- GCP (Google Cloud Platform)
  [ ] GKE Standard (Google Kubernetes Engine Standard)
  [x] GKE Autopilot (Google Kubernetes Engine Autopilot)
- Azure (Microsoft Azure)
  [x] AKS (Azure Kubernets Service)

The following scripts are included in each platform's directory
- login.sh `[This requires special handling, as each platform has a different login method]`
- create.sh `<cluster-name>`
- get_kubeconfig.sh` <cluster-name>`
- list.sh
- delete.sh `<cluster-name>`

## Login

### Aliyun
ALIYUN_ACCESS_KEY_ID=
ALIYUN_ACCESS_KEY_SECRET=
ALIYUN_REGION_ID=
ALIYUN_ZONE_ID=

### AWS
AWS_ACCESS_KEY_ID=
AWS_ACCESS_KEY_SECRET=
AWS_REGION_ID=

### GCP
GCP_PROJECT_ID=
GCP_CRED_DATA=
GCP_REGION_ID=

### Azure
AZURE_APP_ID=
AZURE_PASSWORD=
AZURE_TENANT=
AZURE_REGION_ID=
