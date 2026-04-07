# TASK-OFFCHAIN-013

## Summary

Optimize the wallet-signed session login / restore UX so users reconnect and
resume trading with fewer unnecessary wallet prompts, while preserving the
current session-key trust model, expiry rules, and replay protections.

## Scope

- inspect the current browser wallet connect + session authorization flow for
  public-web trading
- reduce avoidable wallet friction without weakening security:
  - valid local same-wallet same-chain sessions should restore cleanly after
    refresh without another wallet signature
  - trade preparation should only prompt wallet connect / session signature
    when a valid active session is actually missing
  - wallet switch, chain switch, expiry, revoke, and corrupted local session
    states should clear or degrade explicitly instead of leaving ambiguous UI
- if needed, tighten the API/session readback path so the frontend can tell the
  difference between:
  - valid active session
  - expired session
  - revoked session
  - wallet / chain mismatch
- keep the trust model unchanged:
  - do not derive session private keys from wallet signatures
  - do not move session private keys to the server
  - do not weaken session expiry, nonce, or replay enforcement
- keep operator/admin wallet auth out of scope unless a shared helper must be
  adjusted for consistency

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/web/lib/session-client.ts](/Users/zhangza/code/funnyoption/web/lib/session-client.ts)
- [/Users/zhangza/code/funnyoption/web/components/trading-session-provider.tsx](/Users/zhangza/code/funnyoption/web/components/trading-session-provider.tsx)
- [/Users/zhangza/code/funnyoption/backend/internal/shared/auth/session.go](/Users/zhangza/code/funnyoption/backend/internal/shared/auth/session.go)
- [/Users/zhangza/code/funnyoption/backend/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/backend/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/backend/internal/api/handler/sql_store.go](/Users/zhangza/code/funnyoption/backend/internal/api/handler/sql_store.go)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md)

## Owned files

- `web/lib/session-client.ts`
- `web/components/trading-session-provider.tsx`
- `web/components/**` only where a narrow UX/status update is required for the
  session flow
- `internal/api/handler/order_handler.go`
- `internal/api/handler/sql_store.go`
- `internal/shared/auth/session.go`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-013.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-013.md`
- docs only if a short session-flow clarification is needed

## Acceptance criteria

- a valid local same-wallet same-chain session restores after refresh without a
  new wallet signature prompt
- the UI only asks for wallet connect / signature when a valid trading session
  is actually missing
- stale, revoked, expired, wrong-wallet, and wrong-chain sessions are cleared
  or surfaced explicitly
- session nonce / replay / expiry protections remain intact
- validation includes:
  - frontend build
  - any targeted Go tests if backend session behavior changes
  - one browser or scripted proof covering refresh restore, wallet switch, and
    expired-session fallback

## Validation

- `cd web && npm run build`
- targeted Go tests for session auth if backend files change
- one local browser or scripted proof for:
  - restore with valid session
  - clear on wallet or chain mismatch
  - explicit fallback when session is expired or revoked

## Dependencies

- the current session-key model documented in `direct-deposit-session-key.md`
  remains the baseline

## Handoff

- return changed files, validation commands, and before/after UX behavior
- call out any residual tradeoff, such as local-only restore limits when the
  browser has lost the session private key
