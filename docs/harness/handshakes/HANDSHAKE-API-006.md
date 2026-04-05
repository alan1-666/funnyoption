# HANDSHAKE-API-006

## Task

- [TASK-API-006.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-API-006.md)

## Thread owner

- API worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/order-flow.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/sql/schema.md`
- `internal/api/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-024.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-024.md`
- this handshake
- `WORKLOG-API-006.md`

## Files in scope

- `internal/api/**`
- `cmd/api/**` only if narrow import wiring changes are required
- `docs/harness/handshakes/HANDSHAKE-API-006.md`
- `docs/harness/worklogs/WORKLOG-API-006.md`
- `docs/harness/PROJECT_MAP.md` only if the new package boundaries need a
  stable entrypoint update

## Inputs from other threads

- `TASK-CHAIN-024` just landed:
  - runtime-effective market lifecycle now derives `OPEN/CLOSED` truthfully
    from stored `status + close_at`
  - API behavior changed in a narrow, accepted way
- commander review notes:
  - current repo layout is mostly fine at the domain level
  - the main structural pain point is now `internal/api`, not the entire repo
  - the right next cleanup is a narrow API module split, not a full top-level
    `/services/*` migration

## Outputs back to commander

- changed files
- chosen package boundary map
- validation commands
- residual structure debt

## Handoff notes

- keep the refactor narrow and behavior-preserving
- do not reopen rollup, contracts, or frontend lanes in this task
- do not widen into a repo-wide directory migration

## Blockers

- do not change runtime behavior except for tiny compatibility shims required by
  the refactor
- keep `cmd/api` as the service entrypoint

## Status

- active
