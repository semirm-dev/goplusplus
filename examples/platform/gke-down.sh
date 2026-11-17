#!/usr/bin/env bash
# Tears down what gke-up.sh created, from examples/platform:
#   PROJECT_ID=your-project ./gke-down.sh
# kms/ and the state bucket stay.
set -euo pipefail

: "${PROJECT_ID:?set PROJECT_ID}"
cd "$(dirname "$0")"

# The Gateway's load balancer and the PVCs' disks are created by GKE, not Terraform.
# Delete them while the cluster still runs, or they outlive it and keep costing money.
kubectl -n argocd delete application root --ignore-not-found          # stop it recreating the apps
kubectl -n argocd delete applications --all --timeout=10m --ignore-not-found
kubectl delete gateway -A --all --timeout=10m --ignore-not-found 2> /dev/null || true
kubectl delete pvc -A --all --timeout=10m

(cd terraform && terraform destroy -auto-approve -var "project_id=${PROJECT_ID}" \
  -var emulator=false -var "admin_cidr=$(curl -s ifconfig.me)/32")
