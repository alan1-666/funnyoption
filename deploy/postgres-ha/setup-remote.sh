#!/usr/bin/env bash
# From your laptop: sync deploy/postgres-ha to all nodes and print next commands.
# Usage: ./setup-remote.sh /path/to/funnyoption
set -euo pipefail

ROOT="${1:-.}"
SRC="$(cd "${ROOT}/deploy/postgres-ha" && pwd)"

RSYNC=(rsync -avz --delete --exclude 'env-common.env' --exclude 'env-node.local.env' --exclude 'pg-data/**' --exclude 'etcd-data/**')

"${RSYNC[@]}" "${SRC}/" root@76.13.220.236:/opt/postgres-ha/
"${RSYNC[@]}" "${SRC}/" root@117.72.160.220:/opt/postgres-ha/
"${RSYNC[@]}" "${SRC}/" ubuntu@118.193.33.87:/opt/postgres-ha/

echo "Synced. On each host:"
echo "  cd /opt/postgres-ha"
echo "  cp -n env-common.example env-common.env && cp -n env-node.example env-node.local.env"
echo "  # edit env files, then:"
echo "  docker compose -f docker-compose.ha.yml --env-file env-common.env --env-file env-node.local.env build"
echo "  docker compose -f docker-compose.ha.yml --env-file env-common.env --env-file env-node.local.env up -d"
