#!/usr/bin/env bash
# Run once by hand when openbao-0 is Running but not Ready. The output never goes to Git.
set -euo pipefail

bao() { kubectl -n openbao exec -i openbao-0 -- env BAO_TOKEN="${BAO_TOKEN:-}" bao "$@"; }

bao operator init -key-shares=1 -key-threshold=1 -format=json > init.json   # kind only: 1 key
bao operator unseal "$(jq -r '.unseal_keys_b64[0]' init.json)"
export BAO_TOKEN="$(jq -r .root_token init.json)"

bao secrets enable -path=platform kv-v2
read -rs CF_TOKEN && printf %s "$CF_TOKEN" | bao kv put -mount=platform cloudflare api-token=-
read -rs SLACK_URL && printf %s "$SLACK_URL" | bao kv put -mount=platform falco slack-webhook-url=-

bao auth enable kubernetes
bao write auth/kubernetes/config kubernetes_host=https://kubernetes.default.svc

bao policy write eso-read - <<'POLICY'
path "platform/data/*" { capabilities = ["read"] }
POLICY

bao write auth/kubernetes/role/eso \
  bound_service_account_names=external-secrets \
  bound_service_account_namespaces=external-secrets \
  policies=eso-read ttl=1h
