# TASK-OFFCHAIN-003

## Summary

Clean up the read/query surfaces after the local regression path is stable: homepage, market detail, and operator control should all reflect real local state with clear runtime visibility.

## Scope

- tighten the API and frontend read surfaces used by homepage, market detail, and control
- remove or reduce misleading fallback data once `TASK-OFFCHAIN-002` establishes a reproducible local flow
- verify operator visibility for market status, settlement progress, and pending chain queue state
- document any remaining intentional mock or partial surfaces instead of hiding them

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md](/Users/zhangza/code/funnyoption/docs/architecture/ledger-service.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-002.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-002.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-OFFCHAIN-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-004.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-004.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-OFFCHAIN-003.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-003.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-OFFCHAIN-003.md)

## Owned files

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

## Acceptance criteria

- homepage, market detail, and control page all load against real local data without broken placeholder paths
- operator surfaces show enough market and queue state to explain what happened during the `TASK-OFFCHAIN-002` regression run
- any remaining mock or incomplete read surfaces are called out explicitly in code comments, docs, or worklog notes
- worker records an exact pass/fail matrix for homepage, detail page, and control page read quality

## Validation

- `npm run build`
- targeted local API checks for markets, market detail, and chain tasks
- browser smoke checks for `/`, `/markets/<id>`, and `/control`

## Dependencies

- `TASK-OFFCHAIN-004` must complete first and restore resolved-market finality

## Handoff

- return the cleaned read/query surfaces and any remaining gaps
- identify whether `TASK-CHAIN-001` can start cleanly or if another off-chain follow-up is needed
