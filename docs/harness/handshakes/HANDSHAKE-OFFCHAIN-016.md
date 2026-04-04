# HANDSHAKE-OFFCHAIN-016

## Task

- [TASK-OFFCHAIN-016.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-016.md)

## Thread owner

- off-chain auth/schema worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/sql/schema.md`
- `HANDSHAKE-OFFCHAIN-015.md`
- `WORKLOG-OFFCHAIN-015.md`
- `HANDSHAKE-OFFCHAIN-013.md`
- `WORKLOG-OFFCHAIN-013.md`
- `internal/api/handler/sql_store.go`
- `internal/api/handler/order_handler.go`
- `web/lib/session-client.ts`
- this handshake
- `WORKLOG-OFFCHAIN-016.md`

## Files in scope

- `internal/api/handler/sql_store.go`
- `internal/api/handler/order_handler.go` only if vault-scoped readback needs a
  narrow contract fix
- `internal/api/dto/order.go` only if a narrow response change is required
- `internal/api/handler/order_handler_test.go` for a narrow list/readback
  contract regression test
- `internal/api/handler/sql_store_scope_test.go` for targeted
  registration / rotation / lookup coverage across two vault scopes
- `migrations/**`
- `docs/sql/schema.md`
- `docs/architecture/direct-deposit-session-key.md`
- `web/lib/session-client.ts` only if restore readback alignment is necessary
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-016.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-016.md`

## Inputs from other threads

- `TASK-OFFCHAIN-015` landed V2 trading-key registration and truthful restore,
  but left one explicit residual boundary:
  - `wallet_sessions` already durably persists `vault_address`
  - active-key rotation / listing already scope to `wallet + chain + vault`
  - the remaining blocker is the old
    `UNIQUE (wallet_address, session_public_key)` rule, which still prevents
    reusing one trading public key across two vaults on the same
    `wallet + chain`
- `TASK-OFFCHAIN-013` completed the frontend restore UX and kept the
  server-side vault readback contract explicit; this follow-up must make that
  contract durable even when the same trading public key is reused across two
  vaults

## Outputs back to commander

- changed files
- validation commands
- one clear before/after summary of:
  - active-key registration
  - key rotation across two vault scopes
  - restore readback truthfulness

## Blockers

- do not widen into a full auth naming cleanup
- do not remove `/api/v1/sessions` compat tooling in this slice
- keep canonical browser registration on `/api/v1/trading-keys/challenge` +
  `/api/v1/trading-keys`
- do not move private keys to the server
- no open code blocker remains in this slice after
  `migrations/013_wallet_sessions_vault_key_uniqueness.sql`
- residual compatibility tradeoff to keep explicit after closure:
  - deprecated `/api/v1/sessions` create rows still carry blank
    `vault_address`
  - that blank-vault carrier remains intentional because the old wallet-signed
    session contract never included a vault field

## Status

- completed

## Handoff notes

- `wallet_sessions` now durably persists `vault_address` for canonical
  trading-key rows, and active-key rotation now scopes to
  `wallet + chain + vault` instead of collapsing to `wallet + chain`.
- the legacy `UNIQUE (wallet_address, session_public_key)` rule is now
  replaced by durable canonical uniqueness on
  `wallet + chain + vault + session_public_key`, so one wallet can reuse the
  same trading public key across two vaults on the same chain without a
  database collision.
- `GET /api/v1/sessions` now returns and can filter by `vault_address`, so
  browser restore can disambiguate remote active keys by vault during readback.
- deprecated `/api/v1/sessions` compatibility tooling remains in place on
  purpose:
  - legacy session-grant rows still carry blank `vault_address` because that
    signed contract never included a vault field
  - canonical browser registration remains
    `/api/v1/trading-keys/challenge` + `/api/v1/trading-keys`
- regression proof now exists for
  `same wallet + same chain + same public key + two vaults`, including both:
  - PostgreSQL-backed `SQLStore` registration / rotation coverage
  - raw SQL before/after proof showing the old unique rule fails and the new
    durable rule allows `2` rows across `2` vaults
