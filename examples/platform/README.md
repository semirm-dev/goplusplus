# platform

The working code behind the V3 series: a production-like platform on GKE, with CI/CD and GitOps.
Everything runs locally first, on floci-gcp (Terraform) and kind (the in-cluster stack).

- `terraform/` -> VPC, node identity, private GKE cluster, CI identity, Velero bucket
- `terraform/kms/` -> the secrets key, a separate root that is applied once and never destroyed
- `docker-compose.yaml` -> floci-gcp, a local GCP API emulator
- `gitops/` -> what Argo CD syncs, from `examples/platform/gitops` in this repo:
  - `bootstrap/root-<env>.yaml` -> the root Application, one per cluster
  - `apps/` -> the app-of-apps, a Helm chart with one Application per tool in `templates/`
    and `values-kind.yaml` / `values-gke.yaml` setting `env`
  - `projects/` -> the AppProjects, synced at wave -1
  - `platform/<tool>/` -> values and manifests per tool
  - `clusters/<env>/` -> per-cluster values and the Gateway
  - `workloads/api/` -> kustomize base and `overlays/<env>`
- `app/` -> the `api` service and its Dockerfile, built by `.github/workflows/platform-api.yaml`
  at the repo root

Everything lives in one repo, `semirm-dev/goplusplus`. Argo CD points at it with
`basePath: examples/platform/gitops` (`gitops/apps/values.yaml`), and the root apps hard-code
`path: examples/platform/gitops/apps`. CI on `main` builds `examples/platform/app`, pushes and
signs the image, then opens a PR (branch `deploy/api-gke`, default `GITHUB_TOKEN`) that bumps
the digest in `gitops/workloads/api/overlays/gke`. Merging it is the deploy. Turn on "Allow GitHub
Actions to create and approve pull requests" in the repo settings for that step to work.
Only that workflow can get a GCP token (`terraform/ci-identity.tf`) or sign an image Kyverno
accepts (`gitops/platform/kyverno-policies/verify-images.yaml`).

In a team you'd usually split the GitOps tree into its own repo: separate access and review,
and app commits don't wake Argo CD. One repo keeps this example readable in one place.

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
3. Argo CD and the root app. After this Argo CD manages itself from `apps/templates/argocd.yaml`:
   ```bash
   helm upgrade --install argocd argo/argo-cd --version 8.5.8 -n argocd --create-namespace -f gitops/platform/argocd/values.yaml
   kubectl apply -f gitops/bootstrap/root-kind.yaml
   ```
   Argo CD reads `main` of `semirm-dev/goplusplus` on GitHub, not this checkout: push changes
   under `gitops/` before you expect them in the cluster. On a fork, change `repoURL` in
   `gitops/apps/values.yaml`, `gitops/bootstrap/root-*.yaml` and `gitops/projects/`.
4. When `openbao-0` is Running but not Ready, run `gitops/platform/openbao/bootstrap.sh`. Wave 0 waits for it.
5. The api image on kind: `docker build -t api:dev app && kind load docker-image api:dev --name platform`.
6. Real GKE, destroyed the same day:
   ```bash
   (cd terraform/kms && terraform apply -var project_id=$PROJECT_ID -var emulator=false)
   cd terraform && rm backend_override.tf
   terraform init -reconfigure -backend-config=bucket=$PROJECT_ID-tfstate -backend-config=prefix=platform
   terraform apply -var project_id=$PROJECT_ID -var emulator=false -var admin_cidr=$(curl -s ifconfig.me)/32
   gcloud container clusters get-credentials platform --zone europe-west3-a --project $PROJECT_ID
   ```
   Then step 3 with `bootstrap/root-gke.yaml`, and step 4. Replace `PROJECT_ID` in
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
| R1 | `gitops/bootstrap/root-kind.yaml`, `gitops/apps/values.yaml`, `gitops/apps/templates/argocd.yaml`, `gitops/platform/argocd/values.yaml`, `gitops/projects/` |
| R2 | `gitops/apps/templates/cert-manager.yaml`, `gitops/apps/templates/cert-manager-config.yaml`, `gitops/apps/templates/projects.yaml`, `gitops/platform/argocd/values.yaml` |
| R3 | `gitops/apps/values-kind.yaml`, `gitops/apps/values-gke.yaml`, `gitops/apps/templates/cert-manager.yaml`, `gitops/apps/templates/velero.yaml`, `gitops/apps/templates/api.yaml`, `gitops/clusters/` |
| S1 | `gitops/platform/openbao/bootstrap.sh`, `gitops/platform/external-secrets/cluster-secret-store.yaml`, `gitops/platform/cert-manager/cloudflare-token.yaml` |
| S2 | `gitops/platform/cert-manager/cluster-issuers.yaml`, `gitops/platform/gateway/certificate.yaml`, `gitops/clusters/*/gateway.yaml` |
| S3 | `gitops/platform/external-dns/values.yaml`, `gitops/workloads/api/base/httproute.yaml` |
| T1 | `app/Dockerfile`, `.github/workflows/platform-api.yaml` (`test`, `e2e`) |
| T2 | `terraform/ci-identity.tf`, `.github/workflows/platform-api.yaml` (`push`) |
| T3 | `.github/workflows/platform-api.yaml` (`scan-sign`) |
| T4 | `.github/workflows/platform-api.yaml` (`deliver`), `gitops/workloads/api/overlays/gke/kustomization.yaml` |
| U1 | `gitops/platform/kyverno-policies/pod-security.yaml`, `gitops/platform/kyverno-policies/images-and-resources.yaml` |
| U2 | `gitops/platform/kyverno-policies/verify-images.yaml` |
| U3 | `gitops/platform/trivy-operator/values.yaml`, `gitops/platform/falco/values.yaml` |
| V1 | `gitops/platform/kube-prometheus-stack/values.yaml`, `gitops/platform/loki/values.yaml`, `gitops/platform/alloy/values.yaml`, `gitops/workloads/api/base/servicemonitor.yaml`, `gitops/workloads/api/base/prometheusrule.yaml` |
| V2 | `gitops/workloads/api/base/deployment-resources.yaml`, `gitops/workloads/api/base/hpa.yaml`, `gitops/workloads/api/base/pdb.yaml`, `terraform/node_pool.tf` |
| V3 | `terraform/velero.tf`, `gitops/platform/velero/` |
