#!/usr/bin/env bash
# GKE with every track turned on in gitops/apps/values.yaml, from examples/platform:
#   PROJECT_ID=your-project ./gke-up.sh
# Needs the state bucket and kms/ from P4. Tear it down the same day with ./gke-down.sh.
set -euo pipefail

: "${PROJECT_ID:?set PROJECT_ID}"
cd "$(dirname "$0")"
vars=(-var "project_id=${PROJECT_ID}" -var emulator=false -var "admin_cidr=$(curl -s ifconfig.me)/32")

# GKE follows env/gke. The first time it's created from main; after that it must already
# contain main, which is a PR from main to env/gke: the promotion.
git fetch -q origin
if ! git rev-parse -q --verify origin/env/gke > /dev/null; then
  git push origin origin/main:refs/heads/env/gke
elif ! git merge-base --is-ancestor origin/main origin/env/gke; then
  echo "env/gke is behind main: merge a PR from main into env/gke first" >&2
  exit 1
fi

(cd terraform/kms && terraform init -reconfigure -input=false \
  -backend-config="bucket=${PROJECT_ID}-tfstate" -backend-config=prefix=kms > /dev/null \
  && terraform apply -auto-approve -var "project_id=${PROJECT_ID}" -var emulator=false)
(cd terraform && terraform init -reconfigure -input=false \
  -backend-config="bucket=${PROJECT_ID}-tfstate" -backend-config=prefix=platform > /dev/null \
  && terraform apply -auto-approve "${vars[@]}")
gcloud container clusters get-credentials platform --zone europe-west3-a --project "$PROJECT_ID"

helm upgrade --install argocd argo/argo-cd --version 8.5.8 -n argocd --create-namespace \
  -f gitops/platform/argocd/values.yaml -f gitops/clusters/gke/argocd.yaml --wait
kubectl apply -f gitops/projects/project-bootstrap.yaml
kubectl apply -f gitops/bootstrap/root-gke.yaml

# With the secrets track on, the root waits at wave 1 for OpenBao to be initialized
if grep -q '^  secrets: true' gitops/apps/values.yaml; then
  echo "waiting for openbao-0..." >&2
  until kubectl -n openbao get pod openbao-0 > /dev/null 2>&1; do sleep 10; done
  kubectl -n openbao wait pod/openbao-0 --for=jsonpath='{.status.phase}'=Running --timeout=15m
  gitops/platform/openbao/bootstrap.sh gke
  kubectl annotate externalsecret -A --all force-sync="$(date +%s)" --overwrite
fi

kubectl -n argocd get applications
