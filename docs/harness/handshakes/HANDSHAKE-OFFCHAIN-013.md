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
- this task is now blocked behind `TASK-OFFCHAIN-014` because the product auth
  direction changed from “current session-key UX optimization” to a larger
  Stark-style trading-key architecture discussion

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
- blocked pending `TASK-OFFCHAIN-014`:
  - do not start implementation until the auth contract is re-decided
  - if Stark-style trading keys are adopted, this task will be resliced against
    the new baseline instead of the current ed25519-style session model

## Status

- blocked
