# TASK-API-006

## Summary

Clean up the current repo structure by narrowing the refactor to
`internal/api`: split route wiring, handler concerns, and SQL-backed store
logic into clearer module-owned packages without changing runtime behavior or
attempting a full-repo service-directory migration.

## Scope

- build on the current API hardening baseline through `TASK-API-005` plus the
  market lifecycle boundary from `TASK-CHAIN-024`
- keep current product truth unchanged:
  - no auth contract changes
  - no order/matching behavior changes
  - no rollup/prover changes
  - no deploy/runtime boundary changes
- target the main pain point only:
  - `internal/api` currently mixes routes, handlers, lifecycle helpers,
    rollup-shadow glue, and SQL store concerns in one broad package tree
- choose the narrowest structural cleanup:
  - keep `cmd/api` as the service entrypoint
  - keep the rest of `internal/*` on the current domain-based layout
  - reorganize `internal/api` into clearer module-owned boundaries such as
    orders / markets / trading-keys / profile / operator / router
  - preserve existing HTTP contracts and test behavior
  - update docs only where the new package boundaries would otherwise be
    unclear
- do not implement:
  - a top-level `/services/*` migration
  - cross-repo package renames outside the API ownership boundary
  - product behavior changes unless a tiny compatibility shim is strictly
    required by the refactor

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/api](/Users/zhangza/code/funnyoption/internal/api)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-024.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-024.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-024.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-024.md)

## Owned files

- `internal/api/**`
- `cmd/api/**` only if narrow import wiring changes are required
- `docs/harness/handshakes/HANDSHAKE-API-006.md`
- `docs/harness/worklogs/WORKLOG-API-006.md`
- `docs/harness/PROJECT_MAP.md` only if the new package boundaries need a
  stable entrypoint update

## Acceptance criteria

- `internal/api` no longer relies on one broad mixed handler/store layout
- the new package/module boundaries are clear enough that future tasks can own
  orders, markets, trading keys, and router wiring more narrowly
- HTTP behavior remains unchanged at the intended boundary
- tests cover the refactor boundary so behavior drift is caught

## Validation

- targeted Go tests for `internal/api/...`
- one build/run validation for `cmd/api` if imports or wiring move
- `git diff --check`

## Dependencies

- `TASK-CHAIN-024` completed

## Handoff

- return changed files, the chosen package boundary map, validation commands,
  and any residual structure debt intentionally left for later
