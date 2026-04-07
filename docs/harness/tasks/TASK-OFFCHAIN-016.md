# TASK-OFFCHAIN-016

## Summary

Make the trading-key runtime durably truthful to the intended
`wallet + chain + vault` scope by persisting `vault_address` server-side and
stopping active-key rotation / lookup from collapsing to `wallet + chain`.

## Scope

- close the residual V2 auth boundary left by `TASK-OFFCHAIN-015` and
  `TASK-OFFCHAIN-013`
- add durable server-side `vault_address` scope to the trading-key carrier:
  - either extend `wallet_sessions` safely
  - or introduce the narrowest compatible carrier needed for truthful
    `wallet + chain + vault` scoping
- update active-key registration / rotation logic so:
  - registering a key for vault `A` does not revoke an active key for vault `B`
    on the same wallet + chain
  - remote active-key lookup and restore readback can disambiguate by vault
- keep the slice narrow:
  - do not widen into a full session-to-trading-key rename
  - do not remove `/api/v1/sessions` compat tooling in the same task
  - do not change deposit / withdrawal custody semantics
  - do not derive trading private keys from wallet signatures

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-015.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-015.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-015.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md)
- [/Users/zhangza/code/funnyoption/backend/internal/api/handler/sql_store.go](/Users/zhangza/code/funnyoption/backend/internal/api/handler/sql_store.go)
- [/Users/zhangza/code/funnyoption/backend/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/backend/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/web/lib/session-client.ts](/Users/zhangza/code/funnyoption/web/lib/session-client.ts)

## Owned files

- `internal/api/handler/sql_store.go`
- `internal/api/handler/order_handler.go` only if query/readback contract needs
  a narrow vault-scoped fix
- `internal/api/dto/order.go` only if a narrow readback shape update is needed
- `migrations/**` for the narrow schema change
- `docs/sql/schema.md`
- `docs/architecture/direct-deposit-session-key.md`
- `web/lib/session-client.ts` only if restore readback needs a narrow alignment
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-016.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-016.md`

## Acceptance criteria

- server-side active trading-key scope is durably truthful to
  `wallet + chain + vault`
- registering a key for one vault no longer revokes another vault's active key
  on the same wallet + chain
- restore readback can disambiguate remote active keys by vault instead of
  depending on the current single-vault-per-environment assumption
- docs stop describing the vault boundary as an assumption and instead describe
  the landed runtime truth
- validation includes:
  - targeted Go tests for registration / rotation / lookup behavior
  - migration proof or dry-run notes
  - one local or scripted proof for same wallet + chain across two vault scopes

## Validation

- targeted Go tests for auth store / lookup behavior
- migration dry-run or apply notes
- one local or scripted proof for multi-vault scoping

## Dependencies

- `TASK-OFFCHAIN-015` runtime baseline is complete
- `TASK-OFFCHAIN-013` restore UX baseline is complete

## Handoff

- return changed files, validation commands, and before/after scope semantics
- call out any remaining compatibility debt such as `/api/v1/sessions` tooling
