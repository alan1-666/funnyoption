# HANDSHAKE-OFFCHAIN-006

## Task

- [TASK-OFFCHAIN-006.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-006.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `TASK-OFFCHAIN-003.md`
- `HANDSHAKE-OFFCHAIN-003.md`
- `WORKLOG-OFFCHAIN-003.md`
- this handshake
- `WORKLOG-OFFCHAIN-006.md`

## Files in scope

- `web/lib/api.ts`
- `web/app/page.tsx`
- `web/app/control/**`
- `web/app/markets/**`
- `web/lib/types.ts`

## Inputs from other threads

- `TASK-OFFCHAIN-003` made the read surfaces runtime-backed but still leaves one truthfulness gap in SSR failure handling
- chain hardening should wait until this gap is addressed or explicitly accepted

## Outputs back to commander

- changed files
- degraded-path validation notes
- clear statement of whether SSR now distinguishes outage vs. empty-state correctly

## Handoff notes back to commander

- SSR read helpers in `web/lib/api.ts` now return explicit collection/item states for homepage, detail, and control while preserving the old array/null wrappers for out-of-scope consumers.
- homepage now labels market/trade API failures as unavailable instead of flattening them into an empty market board or quiet tape.
- control now separates empty-state from broken-response state; when `/api/v1/chain-transactions` returns `{"items":null}`, SSR marks the queue unavailable instead of showing `EMPTY`.
- market detail now distinguishes invalid id, not found, and API unavailable; when `/api/v1/trades?market_id=<id>` returns `{"items":null}`, the page keeps the market payload visible but marks the trade snapshot unavailable.
- validation summary:
  - `cd /Users/zhangza/code/funnyoption/web && npm run build`: PASS
  - real API SSR smoke on `http://127.0.0.1:3001`: homepage PASS, detail PASS, control PASS
  - forced degraded-path SSR smoke on `http://127.0.0.1:3003` with invalid `NEXT_PUBLIC_API_BASE_URL`: homepage PASS, detail PASS, control PASS

## Blockers

- no frontend blocker remains
- the original backend/API contract follow-up was closed by `TASK-OFFCHAIN-008`

## Status

- completed
