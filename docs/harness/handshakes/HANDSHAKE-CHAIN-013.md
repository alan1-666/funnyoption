# HANDSHAKE-CHAIN-013

## Task

- [TASK-CHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-013.md)

## Thread owner

- chain/rollup worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-012.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-012.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-012.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/sql/schema.md`
- `internal/rollup/**`
- `internal/api/**`
- `internal/shared/auth/**`
- `cmd/local-lifecycle/**`
- `scripts/**`
- `contracts/src/FunnyRollupCore.sol`
- this handshake
- `WORKLOG-CHAIN-013.md`

## Files in scope

- `internal/rollup/**`
- `internal/api/**` only where needed for canonical auth witness capture
- `internal/shared/auth/**` only where needed for canonical V2 witness shape
- `cmd/local-lifecycle/**` only if needed to migrate verifier-eligible proof tooling
- `scripts/**` only if needed to migrate verifier-eligible proof tooling
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-013.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-013.md`
- `contracts/src/**` and `contracts/test/**` only for narrow metadata-aligned
  comments/placeholders if justified

## Inputs from other threads

- `TASK-CHAIN-012` landed:
  - tranche-1 keeps the monotonic-floor nonce contract
  - verifier-gated `FunnyRollupCore` acceptance cannot trust prior
    operator-side auth checks
  - deprecated `/api/v1/sessions` rows are not verifier-eligible baseline
- commander review accepted `TASK-CHAIN-012` as completed
- commander wants the next slice to land one narrow auth witness contract and
  migrate verifier-eligible proof tooling away from the deprecated auth lane

## Outputs back to commander

- changed files
- auth witness contract
- validation commands
- migrated proof-tooling paths
- residual limitations
- recommended verifier-gated implementation follow-up

## Handoff notes

- narrow canonical V2 auth witness lane landed without changing the
  `shadow-batch-v1` public-input shape:
  - canonical V2 `POST /api/v1/trading-keys` registration now appends
    `TRADING_KEY_AUTHORIZED` witness-only shadow journal entries
  - each API-accepted `NONCE_ADVANCED` shadow payload now carries
    `order_authorization` with the exact order-intent message/hash/signature
    plus `authorization_ref`
  - `authorization_ref = trading_key_id:challenge` is the durable join between
    the registration witness and the nonce-advance witness
- replay/public-input boundary stayed narrow:
  - `TRADING_KEY_AUTHORIZED` is replay no-op witness material
  - `orders_root.nonce_root` remains the same monotonic-floor contract
  - no prover, verifier, or withdrawal-claim rewrite landed
- verifier-eligible proof tooling migrated off deprecated `/api/v1/sessions`:
  - canonical readback route is now `GET /api/v1/trading-keys`
  - `cmd/local-lifecycle` trading-key flow now reads back via
    `/api/v1/trading-keys`
  - `scripts/staging-concurrency-orders.mjs` now uses
    `POST /api/v1/trading-keys/challenge` + `POST /api/v1/trading-keys`
    instead of `POST /api/v1/sessions`
- deprecated blank-vault `/api/v1/sessions` rows remain compatibility-only and
  are explicitly marked non-verifier-eligible in the order auth witness

## Blockers

- do not claim the product is already Mode B
- do not widen into prover, verifier, or production withdrawal-claim rewrite
- if contracts are touched, stay on Foundry only

## Status

- completed
