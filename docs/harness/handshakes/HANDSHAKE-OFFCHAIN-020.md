# HANDSHAKE-OFFCHAIN-020

## Task

- [TASK-OFFCHAIN-020.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-020.md)

## Thread owner

- offchain/frontend worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `TASK-OFFCHAIN-019.md`
- `HANDSHAKE-OFFCHAIN-019.md`
- `WORKLOG-OFFCHAIN-019.md`
- `web/app/markets/[marketId]/page.tsx`
- `web/components/live-market-panel.tsx`
- `web/components/order-ticket.tsx`
- `web/components/market-order-activity.tsx`
- this handshake
- `WORKLOG-OFFCHAIN-020.md`

## Files in scope

- `web/app/markets/[marketId]/**`
- `web/components/**` only for the touched market-detail surface
- `docs/architecture/frontend-surface-copy.md`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-020.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-020.md`

## Inputs from other threads

- `TASK-OFFCHAIN-019` already delivered the new Worm-style hierarchy
- latest product feedback from the user:
  - internal design/explanation copy must not leak into the page
  - the repo should keep one explicit frontend rule and follow it strictly

## Outputs back to commander

- changed files
- new frontend surface-copy guideline
- before/after wording summary
- validation commands
- staging verification notes

## Handoff notes

- prefer deleting meta copy over replacing it with another paragraph
- keep labels factual, short, and product-facing
- this is a narrow cleanup/guideline pass, not a wider marketing rewrite

## Blockers

- do not widen into unrelated page copy
- do not regress the new hierarchy/layout from `TASK-OFFCHAIN-019`

## Status

- completed
