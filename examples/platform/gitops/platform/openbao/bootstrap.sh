#!/usr/bin/env bash
# Run once by hand when openbao-0 is Running but not Ready: bootstrap.sh kind|gke
# The keys are printed once and never written to the working directory or Git.
set -euo pipefail

env=${1:?usage: bootstrap.sh kind|gke}
bao() { kubectl -n openbao exec -i openbao-0 -- env BAO_TOKEN="${BAO_TOKEN:-}" bao "$@"; }

umask 077
init=$(mktemp)
trap 'shred -u "$init" 2>/dev/null || rm -f "$init"' EXIT

case $env in
  gke)
    # Cloud KMS unseals. The recovery keys only authorize operations such as generate-root.
    bao operator init -recovery-shares=5 -recovery-threshold=3 -format=json > "$init" ;;
  kind)
    bao operator init -key-shares=5 -key-threshold=3 -format=json > "$init"
    for i in 0 1 2; do bao operator unseal "$(jq -r ".unseal_keys_b64[$i]" "$init")" > /dev/null; done ;;
  *) echo "usage: bootstrap.sh kind|gke" >&2; exit 1 ;;
esac

echo "Move these keys to a password manager now. This is the only copy:" >&2
jq '{unseal_keys_b64, recovery_keys_b64}' "$init"

export BAO_TOKEN="$(jq -r .root_token "$init")"
until bao token lookup > /dev/null 2>&1; do sleep 2; done   # until raft has a leader

bao secrets enable -path=platform kv-v2
read -rsp "Cloudflare API token: " CF_TOKEN && echo
printf %s "$CF_TOKEN" | bao kv put -mount=platform cloudflare api-token=-
read -rsp "Falco Slack webhook URL: " FALCO_SLACK && echo
printf %s "$FALCO_SLACK" | bao kv put -mount=platform falco slack-webhook-url=-
read -rsp "Alertmanager Slack webhook URL: " AM_SLACK && echo
printf %s "$AM_SLACK" | bao kv put -mount=platform alertmanager slack-webhook-url=-
read -rp "Grafana admin user: " GRAFANA_USER
read -rsp "Grafana admin password: " GRAFANA_PASSWORD && echo
printf %s "$GRAFANA_PASSWORD" | bao kv put -mount=platform grafana admin-user="$GRAFANA_USER" admin-password=-
read -rsp "Argo CD Dex GitHub client secret: " DEX_SECRET && echo
printf %s "$DEX_SECRET" | bao kv put -mount=platform argocd dex-github-client-secret=-

bao auth enable kubernetes
bao write auth/kubernetes/config kubernetes_host=https://kubernetes.default.svc

bao policy write eso-read - <<'POLICY'
path "platform/data/*" { capabilities = ["read"] }
POLICY

bao write auth/kubernetes/role/eso \
  bound_service_account_names=external-secrets \
  bound_service_account_namespaces=external-secrets \
  policies=eso-read ttl=1h

[ "$env" = gke ] && bao operator raft list-peers   # openbao-1 and -2 joined on their own

# Nothing needs root from here on. Regenerate one with the keys above: bao operator generate-root
bao token revoke -self
