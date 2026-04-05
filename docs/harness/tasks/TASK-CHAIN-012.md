# TASK-CHAIN-012

## Summary

Decide the first proof-lane nonce/auth contract and verifier-gated acceptance
boundary now that `shadow-batch-v1` and truthful nonce shadowing are stable.

## Scope

- build directly on `TASK-CHAIN-011`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- design-first only:
  - do not implement prover generation
  - do not implement verifier logic
  - do not rewrite production withdrawal claim
  - do not implement forced-withdrawal runtime
- make one explicit decision for the first proof lane:
  - whether the current monotonic-floor nonce contract is acceptable
  - or whether the lane must first tighten to gapless / auth-gadget semantics
- define the minimal prover/verifier contract around already-stable inputs:
  - `shadow-batch-v1` witness
  - public inputs
  - `FunnyRollupCore.recordBatchMetadata(...)`
- define the narrow verifier-gated batch-acceptance boundary:
  - what remains metadata-only
  - what becomes acceptance-gated
  - what still stays shadow-only
- document consequences of the chosen nonce/auth semantics on:
  - replay protection
  - proof-friendliness
  - user-facing auth/trading-key behavior
  - migration cost from the current API/auth model

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-011.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-011.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-011.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-011.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-011.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-011.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/rollup](/Users/zhangza/code/funnyoption/internal/rollup)
- [/Users/zhangza/code/funnyoption/internal/api](/Users/zhangza/code/funnyoption/internal/api)
- [/Users/zhangza/code/funnyoption/internal/shared/auth](/Users/zhangza/code/funnyoption/internal/shared/auth)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)

## Owned files

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-012.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-012.md`
- `contracts/src/**` only for metadata-only boundary notes/placeholders if
  justified
- `contracts/test/**` only for metadata-only boundary notes/placeholders if
  justified
- `internal/rollup/**` only if a doc-aligned placeholder or contract comment
  is required

## Acceptance criteria

- the repo has one explicit answer on first-proof-lane nonce/auth semantics:
  - keep monotonic-floor for tranche 1
  - or require a tighter gapless/auth-gadget contract before prover work
- the verifier-gated `FunnyRollupCore` acceptance boundary is written down
  clearly enough that the implementation worker does not reopen the contract
- docs remain explicit that production truth is unchanged
- rejected options and migration consequences are recorded

## Validation

- docs consistency review
- `git diff --check`

## Dependencies

- `TASK-CHAIN-011` completed

## Handoff

- return changed files, the chosen nonce/auth contract, verifier-gated
  acceptance boundary, rejected options, migration consequences, and the
  recommended first prover/implementation tranche
