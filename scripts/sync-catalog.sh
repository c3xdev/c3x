#!/usr/bin/env bash
# Syncs the vendored catalog snapshot from the knowledge base
# (c3x-pricing-api/catalog → resources/). The pricing-api repo is the
# source of truth; this snapshot only serves --offline and the
# no-network fallback. The catalog-sync workflow fails when the two
# differ, so run this after every catalog change lands in the pricing API.
set -euo pipefail
SRC="${1:-../c3x-pricing-api/catalog}"
DST="$(cd "$(dirname "$0")/.." && pwd)/resources"
[ -d "$SRC" ] || { echo "knowledge base not found at $SRC" >&2; exit 1; }
rsync -a --delete --exclude embed.go "$SRC/aws/" "$DST/aws/"
rsync -a --delete --exclude embed.go "$SRC/azure/" "$DST/azure/"
rsync -a --delete --exclude embed.go "$SRC/gcp/" "$DST/gcp/"
echo "synced $(find "$DST" -name '*.toml' | wc -l | tr -d ' ') catalog files"
