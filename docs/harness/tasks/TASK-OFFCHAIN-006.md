# TASK-OFFCHAIN-006

## Summary

Make homepage, market detail, and control SSR read paths fail honestly when the local API is unavailable instead of silently rendering empty-state content that looks like valid data.

## Scope

- replace silent `[]` / `null` fallbacks in server-side API helpers with explicit degraded-state signaling
- ensure homepage, market detail, and control can distinguish between:
  - real empty data
  - market not found
  - API unavailable / broken response
- keep the scope focused on truthful read failure handling, not broader UI redesign

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-006.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-006.md)

## Owned files

- `web/lib/api.ts`
- `web/app/page.tsx`
- `web/app/control/**`
- `web/app/markets/**`
- `web/lib/types.ts`

## Acceptance criteria

- SSR pages no longer treat API failure as a truthful empty dataset
- homepage and control distinguish “API unavailable” from “empty local DB”
- market detail distinguishes “market not found” from “market fetch failed”
- worker records a pass/fail matrix for degraded read behavior in the worklog

## Validation

- `cd /Users/zhangza/code/funnyoption/web && npm run build`
- browser or curl smoke with the API available
- one degraded-path smoke where the API is intentionally unavailable or the base URL is intentionally invalid

## Dependencies

- `TASK-OFFCHAIN-003` output is the baseline

## Handoff

- return the updated degraded-state behavior for homepage, detail, and control
- note any remaining read-surface truthfulness gaps before chain hardening starts
