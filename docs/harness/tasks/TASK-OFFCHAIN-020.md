# TASK-OFFCHAIN-020

## Summary

Remove implementation-commentary copy from the public market detail page and
codify one reusable frontend surface-copy guideline so product pages stop
showing internal design rationale such as “这里会显示…”, “把…收成…”, or
“像 X 一样…”.

## Scope

- build on `TASK-OFFCHAIN-019`
- keep product/runtime truth unchanged
- narrow the change to:
  - public market detail page copy
  - any shared presentation component on that page that still exposes
    meta/explanatory design language
  - one repo guideline doc for future frontend copy decisions
- do not widen into:
  - backend behavior changes
  - contract/prover work
  - repo-wide page-by-page copy rewrite

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-019.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-019.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-019.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-019.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-019.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-019.md)
- [/Users/zhangza/code/funnyoption/web/app/markets/[marketId]/page.tsx](/Users/zhangza/code/funnyoption/web/app/markets/[marketId]/page.tsx)
- [/Users/zhangza/code/funnyoption/web/components/live-market-panel.tsx](/Users/zhangza/code/funnyoption/web/components/live-market-panel.tsx)
- [/Users/zhangza/code/funnyoption/web/components/order-ticket.tsx](/Users/zhangza/code/funnyoption/web/components/order-ticket.tsx)
- [/Users/zhangza/code/funnyoption/web/components/market-order-activity.tsx](/Users/zhangza/code/funnyoption/web/components/market-order-activity.tsx)

## Owned files

- `web/app/markets/[marketId]/**`
- `web/components/**` only for the touched market-detail surface
- `docs/architecture/frontend-surface-copy.md`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-020.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-020.md`

## Acceptance criteria

- the market detail page no longer shows meta/design-rationale copy to users
- headers, empty states, and helper text are concise and product-facing
- a checked-in guideline doc explicitly forbids self-referential/meta UI copy
- validation includes:
  - `cd web && npm run build`
  - one staging verification pass on the cleaned-up detail page

## Validation

- `cd web && npm run build`
- staging deploy + browser verification
- `git diff --check`

## Dependencies

- `TASK-OFFCHAIN-019` completed

## Handoff

- return changed files, the new frontend copy rules, before/after wording, and
  staging verification notes
