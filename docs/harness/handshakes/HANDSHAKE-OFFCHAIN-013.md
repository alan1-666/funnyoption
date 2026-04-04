# HANDSHAKE-OFFCHAIN-013

## Task

- [TASK-OFFCHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-013.md)

## Thread owner

- web/off-chain auth worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `web/lib/session-client.ts`
- `web/components/trading-session-provider.tsx`
- `internal/shared/auth/session.go`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- this handshake
- `WORKLOG-OFFCHAIN-013.md`

## Files in scope

- `web/lib/session-client.ts`
- `web/components/trading-session-provider.tsx`
- `web/components/**` only for narrow session UX/status fixes
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/shared/auth/session.go`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md`

## Inputs from other threads

- staging and local flows already validated the current session-key baseline
- the next improvement is UX and state handling, not a trust-model rewrite
- commander wants fewer unnecessary wallet prompts, but not at the cost of:
  - weaker expiry
  - weaker nonce / replay protection
  - server-side storage of session private keys
  - deriving a trading key from the wallet signature
- `TASK-OFFCHAIN-015` is now the landed V2 baseline:
  - fresh browser registration must stay on
    `POST /api/v1/trading-keys/challenge` +
    `POST /api/v1/trading-keys`
  - browser-local private keys remain IndexedDB-only
  - `/api/v1/sessions` stays deprecated compatibility surface only for existing
    repo proof tooling, not for new browser callers

## Outputs back to commander

- changed files
- validation commands
- one clear before/after summary of:
  - refresh restore
  - wallet or chain mismatch handling
  - expired / revoked session fallback

## Blockers

- keep the current session-key trust model
- do not widen into admin/operator wallet auth unless a shared helper requires a
  narrow consistency fix
- do not touch unrelated order-matching or settlement logic
- keep the current single-vault-per-environment assumption explicit:
  - browser restore is namespaced by `wallet + chain + vault`
  - durable server lookup still collapses to `wallet + chain` because
    `wallet_sessions` still has no persisted `vault_address`
  - truthful restore therefore still relies on the current one-vault-per-env
    deployment contract until a later schema change lands

## Status

- completed

## Handoff notes

- implementation stayed inside frontend ownership:
  - restore now checks exact local key truth before any reauthorization
  - expired / revoked / rotated / missing IndexedDB key all fail honestly
  - create / prepare flows now re-run restore before prompting a wallet
    signature, so refresh no longer races into unnecessary resign
- backend API surface was not expanded:
  - canonical V2 browser auth remains `/api/v1/trading-keys/challenge` +
    `/api/v1/trading-keys`
  - `/api/v1/sessions` remains read / revoke / deprecated proof-tool compat
