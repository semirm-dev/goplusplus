# platform

The working code behind the V3 series: a production-like platform on GKE, with CI/CD and GitOps.
Everything runs locally first, on floci-gcp (Terraform) and kind (the in-cluster stack).

- `terraform/` -> VPC, node identity, private GKE cluster, CI identity, Velero bucket
- `terraform/kms/` -> the secrets key, a separate root that is applied once and never destroyed
- `docker-compose.yaml` -> floci-gcp, a local GCP API emulator
- `gitops/` -> the `platform-gitops` repo that Argo CD syncs
- `app/` -> the `api` repo's Dockerfile and CI workflow

## Bring-up

1. Terraform on floci (no credits needed):
   ```bash
   docker compose up -d
   (cd terraform/kms && terraform init && terraform apply)
   cd terraform
   echo 'terraform { backend "local" {} }' > backend_override.tf
   terraform init
   # floci has no Artifact Registry, GCS or WIF, so only the cluster and what it needs
   terraform apply -target=google_container_node_pool.workers -var admin_cidr=0.0.0.0/0
   ```
2. kind with Cilium:
   ```bash
   kind create cluster --config gitops/clusters/kind/kind.yaml
   kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml
   kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.3.0/config/crd/experimental/gateway.networking.k8s.io_tlsroutes.yaml
   helm upgrade --install cilium cilium/cilium --version 1.18.2 -n kube-system -f gitops/platform/cilium/values.yaml
   kubectl apply -f gitops/platform/cilium/lb-pool.yaml -f gitops/platform/cilium/l2-policy.yaml
   ```
3. Argo CD and the root app:
   ```bash
   helm upgrade --install argocd argo/argo-cd --version 8.5.8 -n argocd --create-namespace -f gitops/platform/argocd/values.yaml
   kubectl apply -f gitops/bootstrap/cluster.yaml -f gitops/bootstrap/root-app.yaml
   ```
4. When `openbao-0` is Running but not Ready, run `gitops/platform/openbao/bootstrap.sh`. Wave 0 waits for it.
5. The api image on kind: `docker build -t api:dev . && kind load docker-image api:dev --name platform`.
6. Real GKE, destroyed the same day:
   ```bash
   (cd terraform/kms && terraform apply -var project_id=$PROJECT_ID -var emulator=false)
   cd terraform && rm backend_override.tf
   terraform init -reconfigure -backend-config=bucket=$PROJECT_ID-tfstate -backend-config=prefix=platform
   terraform apply -var project_id=$PROJECT_ID -var emulator=false -var admin_cidr=$(curl -s ifconfig.me)/32
   gcloud container clusters get-credentials platform --zone europe-west3-a --project $PROJECT_ID
   ```
   Then step 3 with `env: gke` in `bootstrap/cluster.yaml`, and step 4. Replace `PROJECT_ID` in
   `gitops/platform/velero/values.yaml` first.

## Posts

| Post | Files |
| :--- | :--- |
| P1 | `docker-compose.yaml`, `terraform/providers.tf`, `terraform/variables.tf` |
| P2 | `terraform/network.tf` |
| P3 | `terraform/kms/`, `terraform/iam.tf` |
| P4 | `terraform/backend.tf`, `terraform/cluster.tf` |
| Q1 | `gitops/clusters/kind/kind.yaml`, `gitops/platform/cilium/values.yaml` |
| Q2 | `gitops/platform/cilium/lb-pool.yaml`, `gitops/platform/cilium/l2-policy.yaml`, `gitops/clusters/kind/gateway.yaml`, `gitops/workloads/api/base/httproute.yaml`, `gitops/workloads/api/overlays/gke/healthcheckpolicy.yaml` |
| Q3 | `gitops/workloads/api/base/networkpolicy.yaml`, `gitops/workloads/api/overlays/*/allow-from-gateway.yaml` |
| R1 | `gitops/platform/argocd/values.yaml`, `gitops/bootstrap/root-app.yaml`, `gitops/apps/project-platform.yaml` |
| R2 | `gitops/apps/cert-manager.yaml`, `gitops/apps/cert-manager-config.yaml`, `gitops/platform/cert-manager/cluster-issuers.yaml` |
| R3 | `gitops/bootstrap/cluster.yaml`, `gitops/apps/cert-manager.yaml`, `gitops/apps/api.yaml`, `gitops/clusters/` |
| S1 | `gitops/platform/openbao/bootstrap.sh`, `gitops/platform/external-secrets/cluster-secret-store.yaml`, `gitops/platform/cert-manager/cloudflare-token.yaml` |
| S2 | `gitops/platform/cert-manager/cluster-issuers.yaml`, `gitops/platform/gateway/certificate.yaml`, `gitops/clusters/*/gateway.yaml` |
| S3 | `gitops/platform/external-dns/values.yaml`, `gitops/workloads/api/base/httproute.yaml` |
| T1 | `app/Dockerfile`, `app/.github/workflows/ci.yaml` (`test`, `e2e`) |
| T2 | `terraform/ci-identity.tf`, `app/.github/workflows/ci.yaml` (`push`) |
| T3 | `app/.github/workflows/ci.yaml` (`scan-sign`) |
| T4 | `app/.github/workflows/ci.yaml` (`deliver`), `gitops/workloads/api/overlays/gke/kustomization.yaml` |
| U1 | `gitops/platform/kyverno-policies/pod-security.yaml`, `gitops/platform/kyverno-policies/images-and-resources.yaml` |
| U2 | `gitops/platform/kyverno-policies/verify-images.yaml` |
| U3 | `gitops/platform/trivy-operator/values.yaml`, `gitops/platform/falco/values.yaml` |
| V1 | `gitops/platform/kube-prometheus-stack/values.yaml`, `gitops/platform/loki/values.yaml`, `gitops/platform/alloy/values.yaml`, `gitops/workloads/api/base/servicemonitor.yaml`, `gitops/workloads/api/base/prometheusrule.yaml` |
| V2 | `gitops/workloads/api/base/deployment-resources.yaml`, `gitops/workloads/api/base/hpa.yaml`, `gitops/workloads/api/base/pdb.yaml`, `terraform/node_pool.tf` |
| V3 | `terraform/velero.tf`, `gitops/platform/velero/` |
