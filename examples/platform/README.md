# platform

The working code behind the V3 series: a production-like platform on GKE, with CI/CD and GitOps.
Everything runs locally first, on floci-gcp (Terraform) and kind (the in-cluster stack).

- `terraform/` -> VPC, node identity, private GKE cluster, CI identity, OpenBao identity, Velero bucket
- `terraform/kms/` -> the secrets key and OpenBao's unseal key, a separate root that is applied once and never destroyed
- `docker-compose.yaml` -> floci-gcp, a local GCP API emulator
- `gitops/` -> what Argo CD syncs, from `examples/platform/gitops` in this repo:
  - `bootstrap/root-<env>.yaml` -> the root Application, one per cluster
  - `apps/` -> the app-of-apps, a Helm chart with one Application per tool in `templates/`
    and `values-kind.yaml` / `values-gke.yaml` setting `env`
  - `projects/` -> the AppProjects, synced at wave -1. `default` is locked down, root runs in `bootstrap`
  - `platform/<tool>/` -> values and manifests per tool
  - `clusters/<env>/` -> per-cluster values and the Gateway
  - `workloads/api/` -> kustomize base and `overlays/<env>`
- `app/` -> the `api` service and its Dockerfile, built by `.github/workflows/platform-api.yaml`
  at the repo root

Everything lives in one repo, `semirm-dev/goplusplus`. Argo CD points at it with
`basePath: examples/platform/gitops` (`gitops/apps/values.yaml`), and the root apps hard-code
`path: examples/platform/gitops/apps`. kind follows `main`, GKE follows the `env/gke` branch:
promoting is a PR that moves `env/gke` to a commit already green on kind.

CI on `main` builds `examples/platform/app` once, scans it with Trivy, tests that same image on kind,
then pushes and signs it and opens a PR against `env/gke` (branch `deploy/api-gke`, default
`GITHUB_TOKEN`) that bumps the digest in `gitops/workloads/api/overlays/gke`. Merging it is the
deploy. Turn on "Allow GitHub Actions to create and approve pull requests" in the repo settings for
that step to work. A PR opened with `GITHUB_TOKEN` runs no workflows; use a PAT or GitHub App token
if deploy PRs need checks.
Only that workflow can get a GCP token (`terraform/ci-identity.tf`) or sign an image Kyverno
accepts (`gitops/platform/kyverno-policies/verify-images.yaml`).

In a team you'd usually split the GitOps tree into its own repo: separate access and review,
and app commits don't wake Argo CD. One repo keeps this example readable in one place.

## Before you start

Tools: Docker, Terraform >= 1.5, kubectl, helm, kind, jq, the gcloud CLI, and the `cilium`,
`hubble`, `argocd`, `velero` and `cosign` CLIs. Give Docker room: the whole platform on a 3-node
kind cluster is heavy, plan for about 6 CPUs and 12 GB of memory.

Accounts. On kind any value works for everything but Cloudflare:
- a domain on Cloudflare and an API token for it (`Zone:DNS:Edit`, `Zone:Zone:Read`)
- two Slack incoming webhooks, for Falco and Alertmanager
- a Grafana admin user and password of your choice
- a GCP project with billing on, from P4
- a GitHub OAuth app for Argo CD on GKE (callback `https://argocd.<your domain>/api/dex/callback`)

Fill in the domain, the GCP project and the OAuth app, then commit and push: Argo CD and CI read
from GitHub, not from your checkout.
```bash
cd examples/platform
export DOMAIN=yourdomain.dev PROJECT_ID=your-project
grep -rl 'example\.com' gitops | xargs sed -i "s/example\.com/$DOMAIN/g"
sed -i "s/PROJECT_ID/$PROJECT_ID/g" gitops/platform/velero/values.yaml \
  gitops/clusters/gke/openbao.yaml gitops/platform/kyverno/values.yaml
# and in gitops/platform/argocd/values.yaml: the OAuth app's client ID and the GitHub org
```

## Steps

One step per post, in order, all from `examples/platform`. Argo CD installs a track only when
its switch in `gitops/apps/values.yaml` is on, and each switch is turned on by the step that
introduces it: edit, commit, push, and Argo CD syncs it. GKE steps bring a cluster up with
`./gke-up.sh` and take it down with `./gke-down.sh` the same day.

**P1** floci-gcp: `docker compose up -d`, then `docker compose ps`.

**P2** the network, with local state on floci:
```bash
cd terraform
echo 'terraform { backend "local" {} }' | tee backend_override.tf kms/backend_override.tf
terraform init
terraform apply -target=google_compute_subnetwork.gke_subnet -var admin_cidr=$(curl -s ifconfig.me)/32
```

**P3** the keys, still in `terraform/`: `(cd kms && terraform init && terraform apply)`.

**P4** the cluster on floci, then on real GKE, then gone:
```bash
terraform apply -target=google_container_node_pool.workers -var admin_cidr=$(curl -s ifconfig.me)/32
gcloud auth login && gcloud auth application-default login   # a fresh shell, no floci overrides
gcloud storage buckets create gs://$PROJECT_ID-tfstate --location=europe-west3 --uniform-bucket-level-access
gcloud storage buckets update gs://$PROJECT_ID-tfstate --versioning
rm backend_override.tf kms/backend_override.tf
(cd kms && terraform init -reconfigure -backend-config=bucket=$PROJECT_ID-tfstate -backend-config=prefix=kms \
  && terraform apply -var project_id=$PROJECT_ID -var emulator=false)
terraform init -reconfigure -backend-config=bucket=$PROJECT_ID-tfstate -backend-config=prefix=platform
terraform apply -var project_id=$PROJECT_ID -var emulator=false -var admin_cidr=$(curl -s ifconfig.me)/32
gcloud container clusters get-credentials platform --zone europe-west3-a --project $PROJECT_ID
kubectl get nodes -o wide
terraform destroy -var project_id=$PROJECT_ID -var emulator=false -var admin_cidr=$(curl -s ifconfig.me)/32
cd .. && docker compose down -v
```
`kms/` and the state bucket stay for the rest of the tutorial.

**Q1** kind with Cilium:
```bash
helm repo add cilium https://helm.cilium.io && helm repo add argo https://argoproj.github.io/argo-helm
kind create cluster --config gitops/clusters/kind/kind.yaml
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.3.0/standard-install.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gateway-api/v1.3.0/config/crd/experimental/gateway.networking.k8s.io_tlsroutes.yaml
helm upgrade --install cilium cilium/cilium --version 1.18.2 -n kube-system -f gitops/platform/cilium/values.yaml
kubectl apply -f gitops/platform/cilium/lb-pool.yaml -f gitops/platform/cilium/l2-policy.yaml
cilium status --wait
```

**Q2** the Gateway and the api, by hand for now. Argo CD takes them over in R1.
```bash
# the api overlay has a ServiceMonitor and a PrometheusRule; monitoring comes in V1
kubectl apply --server-side \
  -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.85.0/example/prometheus-operator-crd/monitoring.coreos.com_servicemonitors.yaml \
  -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/v0.85.0/example/prometheus-operator-crd/monitoring.coreos.com_prometheusrules.yaml
kubectl create namespace gateway && kubectl create namespace api
kubectl label namespace api public-gateway=true
docker build -t api:dev app && kind load docker-image api:dev --name platform
kubectl apply -f gitops/clusters/kind/gateway.yaml
kubectl apply -k gitops/workloads/api/overlays/kind
kubectl -n gateway get gateway public   # the address from the pool
docker run --rm --network kind curlimages/curl -s -H "Host: api.$DOMAIN" http://<address>/v1/hello
```

**Q3** the NetworkPolicies came with the overlay. Watch what they drop:
`cilium hubble port-forward &` and `hubble observe --namespace api --verdict DROPPED`.

**R1** Argo CD. Turn on `gateway` and `api` in `gitops/apps/values.yaml`, commit, push, then:
```bash
helm upgrade --install argocd argo/argo-cd --version 8.5.8 -n argocd --create-namespace \
  -f gitops/platform/argocd/values.yaml -f gitops/clusters/kind/argocd.yaml
kubectl apply -f gitops/projects/project-bootstrap.yaml
kubectl apply -f gitops/bootstrap/root-kind.yaml
kubectl -n argocd get applications   # root, argocd, projects, gateway, api: Synced, Healthy
```
Log in as admin: the password is in Secret `argocd-initial-admin-secret`, then
`kubectl -n argocd port-forward svc/argocd-server 8080:443`.

**R2** `kubectl -n argocd get applications -o custom-columns=NAME:.metadata.name,WAVE:.metadata.annotations.argocd\.argoproj\.io/sync-wave`.

**R3** what GKE would get: `helm template root gitops/apps -f gitops/apps/values.yaml -f gitops/apps/values-gke.yaml`.

**S1** secrets. Turn on `secrets`, commit, push. It brings cert-manager too: OpenBao's TLS
certificate comes from it. When `openbao-0` is Running but not Ready:
```bash
gitops/platform/openbao/bootstrap.sh kind   # prints the unseal keys once: keep them
kubectl annotate externalsecret -A --all force-sync=$(date +%s) --overwrite
kubectl get externalsecret -A               # SecretSynced
```
On kind every restart of `openbao-0` seals it again:
`kubectl -n openbao exec -it openbao-0 -- bao operator unseal`, with 3 of the 5 keys.

**S2** the certificate came with S1: `kubectl -n gateway get certificate` shows `READY=True` from
the staging issuer. Switch `issuerRef.name` in `gitops/platform/gateway/certificate.yaml` to
`letsencrypt-prod`, push, then
`docker run --rm --network kind curlimages/curl -s --resolve api.$DOMAIN:443:<address> https://api.$DOMAIN/v1/hello`.

**S3** external-dns, GKE only. Turn on `dns`, commit, push, then
`PROJECT_ID=$PROJECT_ID ./gke-up.sh`. It asks for the same secrets as S1.
`dig +short api.$DOMAIN @1.1.1.1` returns the Gateway's IP in a minute or two. The api itself
has no image on GKE until T4. Then `PROJECT_ID=$PROJECT_ID ./gke-down.sh`.

**T1** CI runs on GitHub. Push a change under `app/`: `test`, `build` and `e2e` go green.

**T2** the CI identity, without GKE, from `terraform/`:
```bash
terraform apply -var project_id=$PROJECT_ID -var emulator=false -var admin_cidr=$(curl -s ifconfig.me)/32 \
  -target=google_iam_workload_identity_pool_provider.github \
  -target=google_service_account_iam_member.ci_wif \
  -target=google_artifact_registry_repository_iam_member.ci_writer
terraform output -raw wif_provider        # repo variable WIF_PROVIDER
terraform output -raw ci_service_account  # repo variable CI_SERVICE_ACCOUNT
```
Add both, and `GCP_PROJECT_ID`, under Settings -> Secrets and variables -> Actions -> Variables.
Push a change under `app/`: `push` goes green.

**T3** the same run signs it. Check from your laptop:
```bash
cosign verify europe-west3-docker.pkg.dev/$PROJECT_ID/apps/api@<digest> \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity https://github.com/semirm-dev/goplusplus/.github/workflows/platform-api.yaml@refs/heads/main
```

**T4** turn on "Allow GitHub Actions to create and approve pull requests" in the repo settings
and push a change under `app/` again: `deliver` opens a PR against `env/gke`. Merging it is the
deploy. From now on `gke-up.sh` stops if `env/gke` is behind `main`: promote with a PR from
`main` into `env/gke`.

**U1** Kyverno. Turn on `guardrails`, commit, push. `kubectl get policyreport -A`: fix what fails
before you switch any policy to `Enforce`.

**U2** signatures are checked on GKE only, kind runs the unsigned `api:dev`. Promote `main` into
`env/gke`, then `./gke-up.sh`, `kubectl -n api get policyreport` (the api passes
`verify-api-images`), and `./gke-down.sh`.

**U3** runtime. Turn on `runtime`, commit, push.
`kubectl get vulnerabilityreports -A` fills in within minutes. Start a shell in `api`,
`kubectl -n api run shell --rm -it --image=busybox -- sh`, and Falco posts to Slack.

**V1** observability. Turn on `observability`, commit, push.
`kubectl -n monitoring port-forward svc/kube-prometheus-stack-grafana 3000:80` and log in with
the Grafana user from S1. The api target is up under Status -> Targets in Prometheus.

**V2** scaling. On kind HPA needs metrics-server:
```bash
helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server
helm upgrade --install metrics-server metrics-server/metrics-server -n kube-system --set args={--kubelet-insecure-tls}
kubectl -n api get hpa   # a percentage, not <unknown>
```
Node scaling is GKE only: promote, `./gke-up.sh`, load the api until a node joins
(`kubectl get nodes -w`), `./gke-down.sh`.

**V3** backups, GKE only. Turn on `backups`, commit, push, promote, then:
```bash
PROJECT_ID=$PROJECT_ID ./gke-up.sh
velero backup create --from-schedule daily --wait
PROJECT_ID=$PROJECT_ID gitops/platform/velero/restore-drill.sh   # restores into a fresh kind cluster
PROJECT_ID=$PROJECT_ID ./gke-down.sh
kind delete cluster --name restore-drill
```

Done: `kind delete cluster --name platform`. `kms/` and the state bucket are all that's left on GCP.

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
| R1 | `gitops/bootstrap/root-kind.yaml`, `gitops/apps/values.yaml`, `gitops/apps/templates/argocd.yaml`, `gitops/platform/argocd/`, `gitops/projects/` |
| R2 | `gitops/apps/templates/cert-manager.yaml`, `gitops/apps/templates/cert-manager-config.yaml`, `gitops/apps/templates/projects.yaml`, `gitops/platform/argocd/values.yaml` |
| R3 | `gitops/bootstrap/root-gke.yaml`, `gitops/apps/values-kind.yaml`, `gitops/apps/values-gke.yaml`, `gitops/apps/templates/cert-manager.yaml`, `gitops/apps/templates/velero.yaml`, `gitops/apps/templates/api.yaml`, `gitops/clusters/` |
| S1 | `terraform/openbao.tf`, `gitops/platform/openbao/`, `gitops/clusters/*/openbao.yaml`, `gitops/platform/cert-manager/internal-ca.yaml`, `gitops/platform/external-secrets/cluster-secret-store.yaml`, `gitops/platform/cert-manager/cloudflare-token.yaml` |
| S2 | `gitops/platform/cert-manager/cluster-issuers.yaml`, `gitops/platform/gateway/certificate.yaml`, `gitops/clusters/*/gateway.yaml` |
| S3 | `gitops/platform/external-dns/values.yaml`, `gitops/workloads/api/base/httproute.yaml` |
| T1 | `app/Dockerfile`, `.github/workflows/platform-api.yaml` (`secrets`, `test`, `build`, `e2e`) |
| T2 | `terraform/ci-identity.tf`, `.github/workflows/platform-api.yaml` (`push`) |
| T3 | `.github/workflows/platform-api.yaml` (`build` for Trivy, `sign`) |
| T4 | `.github/workflows/platform-api.yaml` (`deliver`), `gitops/workloads/api/overlays/gke/kustomization.yaml` |
| U1 | `gitops/platform/kyverno/values.yaml`, `gitops/platform/kyverno-policies/pod-security.yaml`, `gitops/platform/kyverno-policies/images-and-resources.yaml` |
| U2 | `gitops/platform/kyverno-policies/verify-images.yaml`, `terraform/kyverno.tf` |
| U3 | `gitops/platform/trivy-operator/values.yaml`, `gitops/platform/falco/values.yaml` |
| V1 | `gitops/platform/kube-prometheus-stack/`, `gitops/platform/loki/values.yaml`, `gitops/platform/alloy/values.yaml`, `gitops/workloads/api/base/servicemonitor.yaml`, `gitops/workloads/api/base/prometheusrule.yaml` |
| V2 | `gitops/workloads/api/base/deployment-resources.yaml`, `gitops/workloads/api/base/hpa.yaml`, `gitops/workloads/api/base/pdb.yaml`, `gitops/workloads/api/base/deployment.yaml`, `terraform/node_pool.tf` |
| V3 | `terraform/velero.tf`, `gitops/platform/velero/` |
