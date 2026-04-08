# Oracle worker — staging operations

## Role

The `oracle` service polls PostgreSQL for CRYPTO markets that are due for oracle resolution (`metadata.resolution.mode = ORACLE_PRICE`), fetches price from the configured HTTP provider (default: Binance public API), writes `market_resolutions`, and publishes `market.event` with `status=RESOLVED` for the settlement consumer.

## Compose

Service definition: [deploy/staging/docker-compose.staging.yml](/deploy/staging/docker-compose.staging.yml) (`oracle`).

Default host bind: `127.0.0.1:9191` → container `:9191` for observability HTTP.

## Observability

| Endpoint | Purpose |
|----------|---------|
| `GET http://127.0.0.1:9191/healthz` | Liveness JSON `{"status":"ok","service":"oracle"}` |
| `GET http://127.0.0.1:9191/debug/oracle` | Counters, last poll sizes, Kafka topic name, replay hints |

Counters include: `polls_total`, `poll_errors_total`, `frozen_skips_total`, `publish_ok_total`, `publish_fail_total`, and last-poll eligible/processed counts.

Structured logs: poll failures log at `ERROR` from the worker loop (`oracle poll failed`).

## Replay / audit

**Off-chain replay** does not re-execute HTTP fetches automatically. For investigation:

1. **Database**: `market_resolutions` holds `resolver_ref`, `resolved_outcome`, `evidence` (including `observation`, `dispatch`, `retry`).
2. **Kafka**: Re-consume `funnyoption.market.event` (prefix from `FUNNYOPTION_KAFKA_TOPIC_PREFIX`) from a known offset after fixing downstream idempotency; compare `event_id` / `market_id` with DB.

The `/debug/oracle` payload includes `replay.market_event_topic` and a short note for operators.

## Environment (shared with other services)

Uses the same `deploy/staging/.env.staging` as other Go services for:

- `FUNNYOPTION_POSTGRES_DSN`
- `FUNNYOPTION_KAFKA_BROKERS`
- `FUNNYOPTION_KAFKA_TOPIC_PREFIX` (default `funnyoption.`)

Oracle-specific:

| Variable | Default | Meaning |
|----------|---------|---------|
| `FUNNYOPTION_ORACLE_HTTP_ADDR` | `:9191` | Bind address for health + debug (set empty in custom setups only if you disable HTTP — not recommended) |
| `FUNNYOPTION_ORACLE_POLL_INTERVAL` | `5s` | Poll cadence |
| `FUNNYOPTION_ORACLE_BINANCE_BASE_URL` | `https://api.binance.com` | Price HTTP base URL |
| `FUNNYOPTION_ORACLE_SIGNER_KEY` | (empty) | Optional ECDSA key for attestation fields in evidence |

## Deploy

The `oracle` service is included in [scripts/deploy-staging.sh](/scripts/deploy-staging.sh) backend lists. First deploy or after infra changes:

```bash
# On the server, from repo root
./scripts/deploy-staging.sh --service oracle --ref main
# Or full stack
./scripts/deploy-staging.sh --all-services --ref main
```

## Next step (product roadmap)

On-chain read path (`EVM_READ`) is **not** part of this staging slice; see [on-chain-oracle-roadmap.md](/docs/architecture/on-chain-oracle-roadmap.md).
