# TASK-API-004

## Summary

Define and enforce semantic uniqueness for privileged bootstrap orders so a fresh `requested_at` alone cannot silently authorize a second otherwise-identical bootstrap sell order.

## Scope

- inspect the replay/idempotency model delivered by `TASK-API-003`
- choose one explicit semantic-uniqueness policy for privileged bootstrap sell orders beyond replay of the exact same signed proof
- if a second bootstrap order with the same market/user/terms should be allowed, require one explicit distinct operator action handle or equally clear contract instead of relying on a changed `requested_at` alone
- if it should not be allowed, reject the second request even when it carries a newly signed operator proof
- keep the dedicated admin bootstrap flow working on the first legitimate attempt
- keep normal session-backed order flow unchanged
- keep the exact-proof replay protection from `TASK-API-003`
- do not widen into a full redesign of user-session nonces, general order semantics, or matching strategy

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-003.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-004.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-API-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-API-004.md)

## Owned files

- `internal/api/**`
- `internal/api/handler/**`
- `internal/api/dto/**`
- `admin/app/api/operator/**`
- `admin/lib/**` if narrowly required
- related docs/runbooks

## Acceptance criteria

- the repo explicitly defines whether a newly signed bootstrap proof with a different `requested_at` may authorize a second otherwise-identical bootstrap sell order
- the chosen policy is enforced consistently in the dedicated admin bootstrap flow and the shared API order-ingress path
- the first legitimate bootstrap order still succeeds
- exact-proof replay protection from `TASK-API-003` still holds
- normal session-backed order flow still succeeds unchanged
- tests or proof cover:
  - first privileged bootstrap order succeeds
  - a second otherwise-identical bootstrap order with a fresh proof is either rejected or requires an explicit distinct handle

## Validation

- `go test ./internal/api/...`
- `cd /Users/zhangza/code/funnyoption/admin && npm run build`
- one proof for first privileged bootstrap order success
- one proof for the chosen semantic-uniqueness behavior on a second bootstrap attempt with a new `requested_at`

## Dependencies

- `TASK-API-003` output is the baseline

## Handoff

- return the chosen semantic-uniqueness policy for privileged bootstrap orders
- state what field, handle, or semantic key is now authoritative for intentional second bootstrap actions
- note any remaining deeper order-ingress hardening gaps after this change
