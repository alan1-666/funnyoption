#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${PATRONI_NAME:-}" || -z "${NODE_IP:-}" ]]; then
  echo "patroni-entrypoint: PATRONI_NAME and NODE_IP must be set" >&2
  exit 1
fi

if [[ ! -f /etc/patroni/patroni.yml.template ]]; then
  echo "patroni-entrypoint: missing /etc/patroni/patroni.yml.template" >&2
  exit 1
fi

# Only substitute known placeholders so passwords may contain other symbols.
export PATRONI_NAME NODE_IP ETCD_HOSTS POSTGRES_SUPERUSER_PASSWORD REPLICATOR_PASSWORD

# envsubst requires whitespace-separated $VAR names (see gettext envsubst(1)).
envsubst '$PATRONI_NAME $NODE_IP $ETCD_HOSTS $POSTGRES_SUPERUSER_PASSWORD $REPLICATOR_PASSWORD' \
  < /etc/patroni/patroni.yml.template \
  > /tmp/patroni.yml

# PostgreSQL refuses a data directory that is group/other writable (e.g. 1777 from basebackup/volume quirks).
pgdata=/var/lib/postgresql/data
if [[ -d "$pgdata" ]]; then
  chmod 700 "$pgdata" || true
fi

exec patroni /tmp/patroni.yml
