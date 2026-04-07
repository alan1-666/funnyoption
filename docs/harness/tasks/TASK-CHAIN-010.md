# TASK-CHAIN-010

## Summary

Extend the shadow-rollup lane from trading phase into settlement phase:
materialize market-resolution and settlement-payout shadow inputs, make the
`shadow-batch-v1` witness/public-input contract explicit, and add the smallest
L1 batch-metadata surface needed for later prover work.

## Scope

- build directly on `TASK-CHAIN-009`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- extend shadow capture to cover settlement-phase state transitions:
  - market-resolution-triggered order cancellation markers
  - settlement payout markers
  - any minimal resolution metadata needed to keep replay deterministic
- make `shadow-batch-v1` explicit enough for later prover work:
  - canonical field list
  - witness/public-input boundary
  - which namespaces are truthful
  - which namespaces are still zero/default placeholders
- narrow the remaining replay gap around `orders_root`:
  - either add a truthful shadow nonce namespace
  - or make the zero-nonce limitation explicit in the witness contract and tests
- add the minimum L1 contract surface notes or placeholders needed for the next
  tranche:
  - `batch_data_hash`
  - `prev_state_root`
  - `next_state_root`
  - batch metadata event contract
- if any contract placeholder is added, keep it on the repo's existing Foundry
  layout only
- do not implement:
  - prover generation
  - verifier logic
  - production claim rewrite
  - forced-withdrawal runtime
  - full rollup contract system

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-009.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-009.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-009.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-009.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/backend/internal/settlement](/Users/zhangza/code/funnyoption/backend/internal/settlement)
- [/Users/zhangza/code/funnyoption/backend/internal/matching](/Users/zhangza/code/funnyoption/backend/internal/matching)
- [/Users/zhangza/code/funnyoption/foundry.toml](/Users/zhangza/code/funnyoption/foundry.toml)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-010.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-010.md)

## Owned files

- `internal/rollup/**`
- `internal/settlement/**` only where needed for shadow input capture
- `migrations/**`
- `docs/sql/**`
- `docs/architecture/**`
- `contracts/src/**` only if a minimal batch-metadata placeholder is justified
- `contracts/test/**` only if a minimal batch-metadata placeholder is justified
- `docs/harness/handshakes/HANDSHAKE-CHAIN-010.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-010.md`

## Acceptance criteria

- shadow replay extends beyond trading phase into settlement-phase inputs
- `shadow-batch-v1` witness/public-input contract is explicit enough that a
  later prover worker does not reopen the batch-shape debate
- the zero-nonce or truthful-nonce decision is explicit and tested
- the repo has one minimal L1 batch-metadata contract boundary for the next
  tranche, without pretending verifier/proof integration is done
- docs remain explicit that production truth is unchanged

## Validation

- targeted Go tests for shadow settlement replay
- if contract placeholders are added, Foundry syntax/test validation
- `git diff --check`
- one deterministic replay proof that includes settlement-phase inputs

## Dependencies

- `TASK-CHAIN-009` completed

## Handoff

- return changed files, the extended shadow replay contract, validation
  commands, and the recommended prover/L1 follow-up
- state explicitly what is still shadow-only and what is now fixed enough for a
  prover worker to consume
