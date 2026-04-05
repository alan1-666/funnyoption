# TASK-OFFCHAIN-018

## Summary

Finish the current main product lane by closing the remaining market-lifecycle
runtime gap on the backend and improving market-detail visibility on the
frontend: proactively cancel active orders once a market passes `close_at`, and
show the connected user's live order/fill state directly on the market detail
page while removing redundant left-side summary blocks.

## Scope

- build on the runtime-effective lifecycle baseline from `TASK-CHAIN-024`
- keep current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already `Mode B`
- finish the remaining backend lifecycle tail:
  - do not leave post-`close_at` active orders only as inert in-memory matcher
    state
  - add one narrow backend contract that truthfully cancels expired active
    orders and updates read/runtime surfaces consistently
  - keep oracle auto-resolution semantics anchored to `resolve_at`
- improve market detail UX:
  - expose connected-user order/fill visibility on the market detail page
  - make it easy to tell whether an order is queued, open, partially filled,
    filled, or cancelled
  - remove the duplicated left-side summary/info blocks that repeat data already
    shown elsewhere on the page
  - preserve current wallet/trading-key flow; do not reopen auth UX in this
    task
- do not implement:
  - rollup/prover changes
  - repo-wide structure refactors
  - a new trading protocol or websocket contract unless a tiny narrow hook is
    truly required

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md](/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-024.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-024.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-024.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-024.md)
- [/Users/zhangza/code/funnyoption/internal/matching/service](/Users/zhangza/code/funnyoption/internal/matching/service)
- [/Users/zhangza/code/funnyoption/internal/matching/engine](/Users/zhangza/code/funnyoption/internal/matching/engine)
- [/Users/zhangza/code/funnyoption/internal/api/handler](/Users/zhangza/code/funnyoption/internal/api/handler)
- [/Users/zhangza/code/funnyoption/web/app/markets/[marketId]/page.tsx](/Users/zhangza/code/funnyoption/web/app/markets/[marketId]/page.tsx)
- [/Users/zhangza/code/funnyoption/web/components/order-ticket.tsx](/Users/zhangza/code/funnyoption/web/components/order-ticket.tsx)
- [/Users/zhangza/code/funnyoption/web/components/live-market-panel.tsx](/Users/zhangza/code/funnyoption/web/components/live-market-panel.tsx)
- [/Users/zhangza/code/funnyoption/web/lib/api.ts](/Users/zhangza/code/funnyoption/web/lib/api.ts)
- [/Users/zhangza/code/funnyoption/web/lib/types.ts](/Users/zhangza/code/funnyoption/web/lib/types.ts)

## Owned files

- `internal/matching/**`
- `internal/api/handler/**` only if narrow lifecycle/read contract changes are required
- `internal/shared/kafka/**` only if a tiny event contract extension is required
- `web/app/markets/[marketId]/**`
- `web/components/**`
- `web/lib/**`
- `docs/harness/handshakes/HANDSHAKE-OFFCHAIN-018.md`
- `docs/harness/worklogs/WORKLOG-OFFCHAIN-018.md`
- `docs/architecture/order-flow.md` only if lifecycle truth would otherwise be unclear
- `docs/sql/schema.md` only if the chosen close-time cancellation contract affects documented runtime truth

## Acceptance criteria

- post-`close_at` active orders are not left as only inert matcher state; the
  backend has one truthful cancellation contract for them
- market detail page shows connected-user order/fill state clearly enough to
  answer “was my order still挂单, partial, or fully成交?”
- redundant left-side market summary/info blocks are removed without losing the
  important market context
- validation includes:
  - targeted Go tests for the close-time cancellation contract
  - `cd web && npm run build`
  - one deployed staging verification pass for the updated detail-page flow

## Validation

- targeted Go tests for `internal/matching/...` and any touched API handlers
- `cd web && npm run build`
- staging deploy + post-deploy verification
- `git diff --check`

## Dependencies

- `TASK-CHAIN-024` completed

## Handoff

- return changed files, validation commands, backend lifecycle contract,
  detail-page before/after behavior, staging verification notes, and residual
  limitations
