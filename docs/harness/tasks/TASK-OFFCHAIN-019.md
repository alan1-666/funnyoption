# TASK-OFFCHAIN-019

## Summary

Redesign the public market detail page around the information hierarchy and
visual density of Worm's market detail experience: strong event hero,
matchup/context card, compact trading rail, and one clearer activity surface.
Keep FunnyOption's own branding, trading contract, and lifecycle truth.

## Scope

- build on the current market-detail runtime and lifecycle work from
  `TASK-OFFCHAIN-018` and `TASK-CHAIN-025`
- keep current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already `Mode B`
- redesign only the market detail presentation layer:
  - use Worm's detail page as the reference for information hierarchy,
    density, and interaction grouping
  - keep FunnyOption-specific copy, lifecycle labels, and order-entry contract
  - improve the visual relationship between event context, price/chart data,
    tabs, and the trading ticket
  - make it clearer where a user can see:
    - event context
    - current market pricing
    - recent activity / rules
    - their own orders
- remove or restyle any remaining repetitive detail-page blocks that dilute the
  page hierarchy
- do not implement:
  - backend API changes unless a tiny read-only hook is strictly required
  - rollup/prover changes
  - repo-wide structure cleanup

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-018.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-018.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-018.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-018.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md](/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md)
- [`https://www.worm.wtf/market/24v3P83mnK4nEY8o91qRtmC8q3ufrL5SRRWNW9aNpzCU`](https://www.worm.wtf/market/24v3P83mnK4nEY8o91qRtmC8q3ufrL5SRRWNW9aNpzCU)
- [/Users/zhangza/code/funnyoption/web/app/markets/[marketId]/page.tsx](/Users/zhangza/code/funnyoption/web/app/markets/[marketId]/page.tsx)
- [/Users/zhangza/code/funnyoption/web/app/markets/[marketId]/page.module.css](/Users/zhangza/code/funnyoption/web/app/markets/[marketId]/page.module.css)
- [/Users/zhangza/code/funnyoption/web/components/live-market-panel.tsx](/Users/zhangza/code/funnyoption/web/components/live-market-panel.tsx)
- [/Users/zhangza/code/funnyoption/web/components/live-market-panel.module.css](/Users/zhangza/code/funnyoption/web/components/live-market-panel.module.css)
- [/Users/zhangza/code/funnyoption/web/components/order-ticket.tsx](/Users/zhangza/code/funnyoption/web/components/order-ticket.tsx)
- [/Users/zhangza/code/funnyoption/web/components/market-order-activity.tsx](/Users/zhangza/code/funnyoption/web/components/market-order-activity.tsx)
- [/Users/zhangza/code/funnyoption/web/lib/types.ts](/Users/zhangza/code/funnyoption/web/lib/types.ts)

## Owned files

- `web/app/markets/[marketId]/**`
- `web/components/**`
- `web/lib/**` only if a tiny presentation-facing helper or type tweak is required
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-019.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-019.md`

## Acceptance criteria

- the market detail page clearly mirrors the high-level Worm-style hierarchy:
  - event hero / matchup context on the left
  - compact trading rail on the right
  - chart / activity / rules grouped as one coherent middle surface
- the page no longer feels like stacked generic panels; spacing, hierarchy, and
  density should look intentionally designed
- the connected user's order visibility remains present and understandable
- current lifecycle states such as `CLOSED`, `WAITING_RESOLUTION`, and
  `RESOLVED` remain truthful in the new design
- validation includes:
  - `cd web && npm run build`
  - one staging deploy and one visual verification pass on the redesigned page

## Validation

- `cd web && npm run build`
- staging deploy + browser verification on `https://funnyoption.xyz/markets/:marketId`
- `git diff --check`

## Dependencies

- `TASK-OFFCHAIN-018` completed
- `TASK-CHAIN-025` completed

## Handoff

- return changed files, chosen visual contract, before/after detail-page
  behavior, validation commands, staging verification notes, and residual
  limitations
