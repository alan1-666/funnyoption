# HANDSHAKE-OFFCHAIN-015

## Task

- [TASK-OFFCHAIN-015.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-015.md)

## Thread owner

- off-chain auth implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/sql/schema.md`
- `web/lib/session-client.ts`
- `web/components/trading-session-provider.tsx`
- `internal/shared/auth/session.go`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/chain/service/sql_store.go`
- this handshake
- `WORKLOG-OFFCHAIN-015.md`

## Files in scope

- `web/lib/session-client.ts`
- `web/components/trading-session-provider.tsx`
- `web/components/**` only for narrow V2 auth / restore UX updates
- `web/lib/types.ts` only if V2 auth response typing needs a narrow update
- `internal/shared/auth/session.go`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/api/dto/order.go`
- `internal/api/routes_auth.go`
- `internal/api/middleware.go` only if auth-lane error wording must stay aligned
  with the V2 trading-key runtime
- `internal/api/server.go`
- `internal/chain/service/sql_store.go` only if a narrow wallet-binding lookup
  change is required for truthfulness
- `internal/chain/service/listener.go` only if wallet-binding wording would
  otherwise drift from runtime truth
- `docs/architecture/direct-deposit-session-key.md` for runtime-contract
  boundary notes if the compatibility carrier would otherwise over-claim
- `docs/sql/schema.md` for the same compatibility and scope boundary notes
- `internal/shared/auth/session_test.go`
- `internal/api/handler/order_handler_test.go`
- `internal/api/router_test.go`
- `migrations/011_trading_key_challenges.sql` for one-time challenge storage
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-015.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md`
- narrow docs updates only if runtime contract drift would otherwise be unclear

## Inputs from other threads

- `TASK-OFFCHAIN-014` is complete and set the V2 auth contract:
  - reject deterministic signature-derived trading keys
  - accept browser-local random `ED25519` trading keys
  - require wallet-signed `EIP-712` authorization of the public key
- this first slice should stay narrow:
  - challenge issuance
  - trading-key registration
  - compatibility storage
  - truthful local restore
- the older `TASK-OFFCHAIN-013` session-UX lane is blocked until this V2
  baseline exists in runtime form
- deposit attribution should move toward durable wallet binding semantics, not
  "active local key exists" semantics

## Outputs back to commander

- changed files
- validation commands
- one clear before/after summary of:
  - fresh registration
  - refresh restore
  - wallet or chain mismatch handling
  - key rotate or revoke behavior

## Blockers

- do not derive trading private keys from wallet signatures
- do not move private keys to the server
- do not widen into a full order-write payload rename if the compatibility layer
  can keep this slice narrow
- keep deposit / withdrawal custody semantics unchanged
- remaining boundary to state clearly, not block on:
  - `wallet_sessions` still has no durable `vault_address`, so active-key
    rotation and listing remain effectively `wallet + chain` under the current
    single-vault-per-environment assumption

## Status

- completed

## Handoff notes

- V2 runtime now issues one-time SQL-backed challenges and verifies
  `EIP-712` wallet authorization of one browser-local `ED25519` trading key.
- `POST /api/v1/sessions` is restored as a deprecated compatibility route so
  repo proof tooling such as `cmd/local-lifecycle` and
  `scripts/staging-concurrency-orders.mjs` no longer regress on route removal.
- `wallet_sessions` remains the compatibility carrier for active trading-key
  state and order nonces; the order write path still accepts the current
  `session_*` payload names.
- truthful restore now depends on local metadata + IndexedDB private-key
  presence + wallet / chain / vault match + server-side active-key readback.
- chain deposit / withdrawal attribution now resolves from
  `user_profiles.wallet_address`, not from “active local key exists”.
- residual risk still carried into follow-up:
  - `wallet_sessions` compatibility storage does not durably carry
    `vault_address`, so durable active-key rotation and listing still collapse
    to `wallet + chain` instead of the fuller
    `wallet + chain + vault` contract
  - current runtime truth for the fuller contract therefore still depends on
    the current single-vault-per-environment assumption plus browser-side
    vault-scoped restore checks
