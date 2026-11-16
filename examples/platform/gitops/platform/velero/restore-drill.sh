#!/usr/bin/env bash
# Restore drill on a fresh kind cluster, with a read-only key that is deleted after.
set -euo pipefail

: "${PROJECT_ID:?set PROJECT_ID}"

kind create cluster --name restore-drill
velero install --provider gcp \
  --plugins velero/velero-plugin-for-gcp:v1.12.0 \
  --bucket "${PROJECT_ID}-velero" \
  --secret-file ./restore-drill-key.json \
  --use-node-agent --use-volume-snapshots=false
kubectl -n velero patch backupstoragelocation default --type merge \
  -p '{"spec":{"accessMode":"ReadOnly"}}'
velero backup get
velero restore create drill --from-schedule daily --wait
velero restore describe drill --details
kubectl -n api get pods,pvc
