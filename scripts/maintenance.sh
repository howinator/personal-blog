#!/usr/bin/env bash
#
# Switch howinator.io between normal mode (Cloudflare Tunnel) and
# maintenance mode (Cloudflare Pages static fallback).
#
# Usage: ./scripts/maintenance.sh on|off
#
# Prerequisites (one-time):
#   1. Create a Cloudflare Pages project connected to the personal-blog repo
#      (Dashboard → Pages → Create → Connect GitHub → howinator/personal-blog)
#      Build config: root dir "site", build cmd "hugo --minify", output "public",
#      env var HUGO_VERSION=0.147.0
#   2. Store the Pages domain in Pulumi config:
#      pulumi config set homeserver:pagesDomain <project>.pages.dev -C $HOMESERVER_DIR
#
# Required Pulumi config keys (in homeserver stack):
#   homeserver:cloudflareApiToken
#   homeserver:cloudflareZoneId
#   homeserver:tunnelDomain
#   homeserver:pagesDomain

set -euo pipefail

DOMAIN="howinator.io"
CF_API="https://api.cloudflare.com/client/v4"

usage() {
    echo "Usage: $0 on|off"
    echo ""
    echo "  on   Switch to maintenance mode (Cloudflare Pages)"
    echo "  off  Switch back to normal mode (Cloudflare Tunnel)"
    exit 1
}

if [[ $# -ne 1 ]] || [[ "$1" != "on" && "$1" != "off" ]]; then
    usage
fi

MODE="$1"

: "${HOMESERVER_DIR:?HOMESERVER_DIR must be set (path to homeserver Pulumi project)}"

# Read config from Pulumi
pulumi_get() {
    pulumi config get "$1" -C "$HOMESERVER_DIR" 2>/dev/null
}

echo "Reading config from Pulumi..."
CF_API_TOKEN=$(pulumi_get "homeserver:cloudflareApiToken")
ZONE_ID=$(pulumi_get "homeserver:cloudflareZoneId")
TUNNEL_DOMAIN=$(pulumi_get "homeserver:tunnelDomain")
PAGES_DOMAIN=$(pulumi_get "homeserver:pagesDomain")

if [[ -z "$CF_API_TOKEN" || -z "$ZONE_ID" || -z "$TUNNEL_DOMAIN" || -z "$PAGES_DOMAIN" ]]; then
    echo "Error: missing required Pulumi config values." >&2
    echo "Ensure these are set in the homeserver stack:" >&2
    echo "  homeserver:cloudflareApiToken" >&2
    echo "  homeserver:cloudflareZoneId" >&2
    echo "  homeserver:tunnelDomain" >&2
    echo "  homeserver:pagesDomain" >&2
    exit 1
fi

if [[ "$MODE" == "on" ]]; then
    TARGET="$PAGES_DOMAIN"
    LABEL="maintenance (Pages)"
else
    TARGET="$TUNNEL_DOMAIN"
    LABEL="normal (Tunnel)"
fi

# Find the CNAME record for the root domain
echo "Looking up CNAME record for $DOMAIN..."
RESPONSE=$(curl -sf \
    -H "Authorization: Bearer $CF_API_TOKEN" \
    -H "Content-Type: application/json" \
    "$CF_API/zones/$ZONE_ID/dns_records?type=CNAME&name=$DOMAIN")

RECORD_ID=$(echo "$RESPONSE" | jq -r '.result[0].id // empty')
CURRENT_TARGET=$(echo "$RESPONSE" | jq -r '.result[0].content // empty')

if [[ -z "$RECORD_ID" ]]; then
    echo "Error: no CNAME record found for $DOMAIN" >&2
    exit 1
fi

echo "Current CNAME target: $CURRENT_TARGET"

if [[ "$CURRENT_TARGET" == "$TARGET" ]]; then
    echo "Already in $LABEL mode. Nothing to do."
    exit 0
fi

# Update the CNAME record
echo "Switching to $LABEL mode..."
UPDATE_RESPONSE=$(curl -sf -X PATCH \
    -H "Authorization: Bearer $CF_API_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"content\": \"$TARGET\", \"proxied\": true}" \
    "$CF_API/zones/$ZONE_ID/dns_records/$RECORD_ID")

SUCCESS=$(echo "$UPDATE_RESPONSE" | jq -r '.success')

if [[ "$SUCCESS" != "true" ]]; then
    echo "Error: failed to update DNS record" >&2
    echo "$UPDATE_RESPONSE" | jq '.errors' >&2
    exit 1
fi

echo "Done: $CURRENT_TARGET -> $TARGET"
echo "Mode: $LABEL"
