# TASK-OFFCHAIN-015

## Summary

Implement the first V2 trading-key runtime slice: issue wallet auth challenges,
register a browser-local trading key by `EIP-712`, store it in the current
compatibility slot, and make browser restore truthful on the new auth baseline.

## Scope

- implement the first runtime slice from
  `docs/architecture/direct-deposit-session-key.md`
- add one-time challenge issuance for trading-key authorization:
  - server-generated
  - scoped to `wallet_address + chain_id + vault_address`
  - single-use
  - `5 minute` expiry
- add wallet-signed trading-key registration:
  - `EIP-712` / `eth_signTypedData_v4`
  - authorize one browser-local `ED25519` trading public key
  - verify chain and vault binding from the typed-data domain
- reuse the current `wallet_sessions` persistence slot as the compatibility
  carrier for trading keys:
  - `session_id` -> `trading_key_id`
  - `session_public_key` -> `trading_public_key`
  - `last_order_nonce` remains the replay counter
- update the web auth client so the browser:
  - generates one local `ED25519` trading keypair
  - stores the private key locally
  - stores only lightweight metadata separately
  - restores truthfully after refresh
- keep this slice narrow:
  - do not yet rewrite the full public order-write path onto `trading_*` field
    names if the compatibility layer can avoid it
  - do not widen into admin/operator wallet auth
  - do not move private keys server-side
  - do not derive trading private keys from wallet signatures
- if backend readback is needed for truthful restore, add the smallest session /
  trading-key status surface needed for the web client

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/web/lib/session-client.ts](/Users/zhangza/code/funnyoption/web/lib/session-client.ts)
- [/Users/zhangza/code/funnyoption/web/components/trading-session-provider.tsx](/Users/zhangza/code/funnyoption/web/components/trading-session-provider.tsx)
- [/Users/zhangza/code/funnyoption/internal/shared/auth/session.go](/Users/zhangza/code/funnyoption/internal/shared/auth/session.go)
- [/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go](/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go)
- [/Users/zhangza/code/funnyoption/internal/chain/service/sql_store.go](/Users/zhangza/code/funnyoption/internal/chain/service/sql_store.go)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-015.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-015.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md)

## Owned files

- `web/lib/session-client.ts`
- `web/components/trading-session-provider.tsx`
- `web/components/**` only for narrow V2 auth / restore UX updates
- `internal/shared/auth/session.go`
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/chain/service/sql_store.go` only if a narrow wallet-binding lookup
  change is required for truthfulness
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-015.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md`
- narrow docs updates only if runtime contract drift would otherwise be unclear

## Acceptance criteria

- one wallet can request a challenge and successfully register one browser-local
  `ED25519` trading key using `EIP-712`
- the server stores the authorized key in the current compatibility slot without
  trusting client-provided `user_id`
- duplicate registration of the same key is idempotent
- a new key for the same wallet rotates the old active key cleanly
- the browser restore path is truthful:
  - valid same-wallet same-chain same-vault key metadata restores without a new
    wallet signature
  - wallet mismatch, chain mismatch, missing local private key, revoked key, and
    expired challenge degrade explicitly
- validation includes:
  - targeted Go tests for auth / storage behavior
  - `cd web && npm run build`
  - one local browser or scripted proof of:
    - fresh authorization
    - refresh restore
    - key rotate or revoke behavior

## Validation

- targeted Go tests for auth / storage behavior
- `cd web && npm run build`
- one local browser or scripted proof for:
  - authorize trading key
  - restore with valid key metadata
  - clear on wallet or chain mismatch
  - rotate or revoke fallback

## Dependencies

- `TASK-OFFCHAIN-014` design contract is now the baseline

## Handoff

- return changed files, validation commands, and before/after auth behavior
- call out residual tradeoffs such as compatibility naming or temporary
  `wallet_sessions` reuse
