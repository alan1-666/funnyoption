# HANDSHAKE-OFFCHAIN-003

## Task

- [TASK-OFFCHAIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-003.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/order-flow.md`
- `docs/architecture/ledger-service.md`
- `docs/sql/schema.md`
- `TASK-OFFCHAIN-002.md`
- `HANDSHAKE-OFFCHAIN-002.md`
- `WORKLOG-OFFCHAIN-002.md`
- `TASK-OFFCHAIN-004.md`
- `HANDSHAKE-OFFCHAIN-004.md`
- `WORKLOG-OFFCHAIN-004.md`
- this handshake
- `WORKLOG-OFFCHAIN-003.md`

## Files in scope

- `internal/api/**`
- `internal/ws/**`
- `web/app/page.tsx`
- `web/app/control/**`
- `web/app/markets/**`
- `web/components/live-market-panel*`
- `web/components/chain-task-board*`
- `web/lib/api.ts`
- `web/lib/types.ts`
- `README.md`

## Inputs from other threads

- `TASK-OFFCHAIN-004` has restored resolved-market finality and provided the updated regression outcome
- chain hardening should stay out of scope unless a read surface depends on already existing chain queue data

## Outputs back to commander

- changed files
- read-surface validation notes for homepage, detail, and control
- explicit remaining gaps or follow-up tasks if a surface is still partial

## Handoff notes back to commander

- read surfaces now use real local API data instead of frontend mock fallbacks:
  - homepage picks an actually open lead market and renders resolved markets with runtime-derived 100/0 odds
  - market detail no longer fabricates initial depth/ticker ladders when the local `ws` service has no snapshot yet
  - control queue now refreshes in-browser after the API gained local CORS headers for `http://127.0.0.1:3000`
- exact validation result:
  - homepage: PASS
  - detail page (`/markets/220140402`): PASS
  - control page (`/control`): PASS
- remaining gaps surfaced by the truthful read layer:
  - reused local DB still exposes historical resolved-market residue on `market_id=1101` (`runtime.active_order_count=2`) even though the current finality fix path is correct
  - frontend dev pages still emit a `/favicon.ico` 404 in the browser console; this did not block the read surfaces
- commander review found one follow-up gap:
  - `web/lib/api.ts` still swallows non-OK or network failures and returns `[]` / `null`, so SSR pages can still mislabel an API outage as an empty queue or empty market list

## Blockers

- no blocker remains on homepage / detail / control runtime cleanup itself
- the original SSR truthfulness follow-up was closed by `TASK-OFFCHAIN-006`
- historical local DB hygiene was routed separately and does not block this task from being considered complete

## Status

- completed
