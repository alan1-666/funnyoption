# TASK-CHAIN-009

## Summary

Implement the first shadow-rollup tranche for Mode B: append-only sequencer
journal, durable batch input materialization, and deterministic shadow roots,
without changing production settlement truth yet.

## Scope

- implement the first safe runtime slice from
  `docs/architecture/mode-b-zk-rollup.md`
- keep the current product truth unchanged:
  - SQL balances / positions / payouts remain production truth
  - Kafka-driven settlement remains production truth
  - this task must not claim FunnyOption is already Mode B
- introduce dedicated rollup artifacts rather than overloading current tables:
  - sequencer journal storage
  - durable batch input storage
  - proven-root / shadow-root metadata storage if needed
- define and implement one narrow runtime path for shadow capture:
  - accepted orders
  - cancellations
  - trades
  - deposit credits
  - withdrawal requests
  - market resolution markers only if needed to keep replay deterministic
- add a deterministic replayer / shadow-state builder that can derive:
  - `balances_root`
  - `orders_root`
  - `positions_funding_root`
  - `withdrawals_root`
  - `state_root`
- if the first cut cannot truthfully populate every namespace yet:
  - make the missing namespace explicit
  - use deterministic zero/default roots only if documented and tested
- define one canonical batch-input encoding / storage contract for the shadow lane
- add the minimum docs/schema updates needed so later prover / contract tasks
  can build on explicit artifacts instead of re-deciding storage shape
- do not implement:
  - prover generation
  - verifier contract
  - production withdrawal claim contract
  - forced withdrawal / freeze runtime
  - broad rewrites of matching / account / settlement production paths

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/internal/matching](/Users/zhangza/code/funnyoption/internal/matching)
- [/Users/zhangza/code/funnyoption/internal/account](/Users/zhangza/code/funnyoption/internal/account)
- [/Users/zhangza/code/funnyoption/internal/settlement](/Users/zhangza/code/funnyoption/internal/settlement)
- [/Users/zhangza/code/funnyoption/internal/chain](/Users/zhangza/code/funnyoption/internal/chain)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-009.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-009.md)

## Owned files

- `internal/rollup/**`
- `migrations/**`
- `docs/sql/**`
- `docs/architecture/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-009.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-009.md`
- narrow touchpoints into existing services only where needed to source
  canonical shadow inputs

## Acceptance criteria

- the repo gains one explicit shadow-rollup storage/runtime boundary:
  - append-only sequencer journal
  - durable batch input artifact
  - deterministic root derivation path
- the implementation is explicit that production settlement truth is unchanged
- one documented replay flow can rebuild shadow roots from durable inputs
- docs/sql updates explain the new rollup-shadow artifacts and their relation
  to existing SQL truth
- validation proves:
  - deterministic replay
  - stable root derivation for the same input
  - no silent dependence on live SQL snapshots or Kafka offsets during replay

## Validation

- targeted Go tests for the new rollup-shadow package / replayer
- migration apply / schema validation for new storage artifacts
- `git diff --check`
- one deterministic replay proof in the worklog

## Dependencies

- `TASK-CHAIN-008` completed

## Handoff

- return changed files, the new shadow-rollup artifacts, validation commands,
  and the next recommended prover / contract tranche
- state explicitly which parts are still shadow-only and not yet production
  settlement truth
