# HANDSHAKE-ADMIN-001

## Task

- [TASK-ADMIN-001.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-ADMIN-001.md)

## Thread owner

- implementation worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/architecture/order-flow.md`
- `docs/topics/kafka-topics.md`
- `TASK-OFFCHAIN-008.md`
- `HANDSHAKE-OFFCHAIN-008.md`
- `WORKLOG-OFFCHAIN-008.md`
- this handshake
- `WORKLOG-ADMIN-001.md`

## Files in scope

- `web/app/admin/**`
- `web/components/market-studio.tsx`
- `web/components/market-studio.module.css`
- `web/lib/api.ts`
- `web/lib/types.ts`
- `web/app/control/page.tsx`
- any new `cmd/**`, `scripts/**`, or docs files required for the reproducible lifecycle run

## Inputs from other threads

- public user flows were already cleaned so internal operator copy should stay out of `/`, `/markets/*`, and `/portfolio`
- the old `/control` surface is intentionally a placeholder and should remain non-operational for public users
- local off-chain behavior is stable enough to demonstrate matching and settlement, but there is no dedicated reproducible lifecycle runner yet

## Outputs back to commander

- changed files
- exact admin route behavior and any auth assumptions
- exact lifecycle command(s), required env, and observed terminal-state evidence
- clear note about whether deposit credit is driven by a true listener event or an explicit local simulation path

## Blockers

- do not re-expose operator internals on the public site header or market pages
- do not widen into unrelated claim-lane hardening in this task

## Status

- completed
