# TASK-CHAIN-005

## Summary

Define the oracle-settled crypto market contract and the first safe
implementation cut so crypto markets can auto-resolve from an external price
source with auditable evidence and a manual operator override.

## Scope

- stay design-first for this task; do not jump straight into a broad runtime
  implementation before the contract is explicit
- define the canonical market metadata for oracle-settled crypto markets:
  - source kind
  - instrument / symbol pair
  - comparator rule
  - strike or threshold if needed
  - settlement timestamp / window
  - evidence fields needed for later audit and UI display
- choose and document the first resolver boundary:
  - dedicated oracle worker
  - chain-service extension
  - settlement pre-resolver
  - or another narrowly justified boundary
- define how auto-resolution should work end to end:
  - fetch external price
  - persist observation / evidence
  - emit idempotent market resolution
  - preserve manual operator resolve as a safe fallback / override path
- define failure handling:
  - delayed price availability
  - source outage
  - conflicting observations
  - retriable versus terminal error states
- if small scaffolding is genuinely low-risk, allow the smallest docs/schema/DTO
  placeholders needed to unblock the follow-up implementation task, but do not
  widen into a full runtime resolver unless the contract is already explicit

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/market-taxonomy-and-options.md](/Users/zhangza/code/funnyoption/docs/architecture/market-taxonomy-and-options.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/admin/components/market-studio.tsx](/Users/zhangza/code/funnyoption/admin/components/market-studio.tsx)
- [/Users/zhangza/code/funnyoption/admin/app/api/operator/markets/route.ts](/Users/zhangza/code/funnyoption/admin/app/api/operator/markets/route.ts)
- [/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go](/Users/zhangza/code/funnyoption/internal/api/handler/sql_store.go)
- [/Users/zhangza/code/funnyoption/internal/settlement/service/processor.go](/Users/zhangza/code/funnyoption/internal/settlement/service/processor.go)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-005.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-005.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-005.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-005.md)

## Owned files

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-005.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-005.md`
- if truly needed for the design handoff:
  - narrow DTO or schema placeholder changes only
  - no broad runtime resolver implementation in this task

## Acceptance criteria

- one explicit design document or design-section update covers:
  - metadata contract for oracle-settled crypto markets
  - resolver service boundary
  - evidence persistence shape
  - idempotent auto-resolution flow
  - failure / retry / manual override rules
- the design states the first implementation cut clearly enough that a follow-up
  worker can implement it without reopening architecture
- existing manual operator resolve remains the fallback path in the design
- any optional schema / DTO placeholders stay narrow and are justified

## Validation

- docs are internally consistent with current market creation and settlement
  flows
- if schema placeholders are added, they line up with `docs/sql/schema.md`
- worklog records the chosen contract and the most important rejected options

## Dependencies

- current staging and local flows are the stable baseline

## Handoff

- return changed files, the final oracle-market contract, and the recommended
  next implementation slice
- state explicitly what remains out of scope for the follow-up runtime worker
