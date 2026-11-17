#!/usr/bin/env bash
# Restore drill on a fresh kind cluster, with a read-only key that is deleted after.
set -euo pipefail

: "${PROJECT_ID:?set PROJECT_ID}"
sa="velero-drill@${PROJECT_ID}.iam.gserviceaccount.com"

# A separate GSA that can only read the bucket: kind has no Workload Identity
gcloud iam service-accounts create velero-drill --project "$PROJECT_ID"
gcloud storage buckets add-iam-policy-binding "gs://${PROJECT_ID}-velero" \
  --member "serviceAccount:${sa}" --role roles/storage.objectViewer
gcloud iam service-accounts keys create ./restore-drill-key.json --iam-account "$sa"
trap 'rm -f ./restore-drill-key.json; gcloud iam service-accounts delete "$sa" --quiet' EXIT

kind create cluster --name restore-drill
velero install --provider gcp \
  --plugins velero/velero-plugin-for-gcp:v1.12.0 \
  --bucket "${PROJECT_ID}-velero" \
  --secret-file ./restore-drill-key.json \
  --use-node-agent --use-volume-snapshots=false
kubectl -n velero patch backupstoragelocation default --type merge \
  -p '{"spec":{"accessMode":"ReadOnly"}}'
velero backup get
# openbao's data is sealed by Cloud KMS on GKE, it can't unseal on kind
velero restore create drill --from-schedule daily --exclude-namespaces openbao --wait
velero restore describe drill --details
kubectl -n api get pods,pvc
