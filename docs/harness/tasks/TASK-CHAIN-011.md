# TASK-CHAIN-011

## Summary

Lift API/auth nonce advances into the shadow-rollup lane so
`orders_root.nonce_root` becomes truthful shadow state instead of a zero
placeholder, and lock the prover-facing public-input contract before any
verifier-gated batch acceptance work starts.

## Scope

- build directly on `TASK-CHAIN-010`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- add durable shadow inputs for replay-protection / auth nonce progression:
  - API/auth challenge or order-write nonce advances only where they are
    canonical for future replay protection
  - any minimal namespace metadata needed so the shadow lane can replay the
    nonce subtree deterministically
- replace the current `orders_root.nonce_root = ZeroNonceRoot()` placeholder
  with truthful shadow state derived only from durable batch input
- keep `shadow-batch-v1` explicit and stable:
  - do not reopen the whole witness/public-input shape
  - only extend it narrowly for the nonce/public-input lane
- clarify how the nonce subtree relates to:
  - order replay protection
  - auth/session or trading-key nonce semantics
  - later prover public inputs
- if any L1 placeholder notes move, keep them Foundry-only and metadata-only
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
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-010.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-010.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-010.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-010.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/rollup](/Users/zhangza/code/funnyoption/internal/rollup)
- [/Users/zhangza/code/funnyoption/internal/api](/Users/zhangza/code/funnyoption/internal/api)
- [/Users/zhangza/code/funnyoption/internal/shared/auth](/Users/zhangza/code/funnyoption/internal/shared/auth)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)

## Owned files

- `internal/rollup/**`
- `internal/api/**` only where needed to expose canonical nonce shadow inputs
- `internal/shared/auth/**` only where needed to define nonce truth
- `migrations/**`
- `docs/sql/**`
- `docs/architecture/**`
- `contracts/src/**` only if metadata-only notes/placeholders need a narrow
  update
- `contracts/test/**` only if metadata-only notes/placeholders need a narrow
  update
- `docs/harness/handshakes/HANDSHAKE-CHAIN-011.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-011.md`

## Acceptance criteria

- `orders_root.nonce_root` is no longer a hidden zero placeholder:
  - either truthful shadow state is implemented from durable inputs
  - or the remaining limitation is narrowed further with an explicit tested
    contract the prover lane can rely on
- the nonce/public-input lane is documented clearly enough that a later prover
  worker does not reopen replay-protection truth boundaries
- deterministic replay still uses only durable batch input
- docs remain explicit that production truth is unchanged

## Validation

- targeted Go tests for rollup replay and the touched API/auth nonce path
- `git diff --check`
- one deterministic replay proof showing the nonce subtree behavior

## Dependencies

- `TASK-CHAIN-010` completed

## Handoff

- return changed files, the nonce/public-input contract, validation commands,
  and the recommended prover/verifier follow-up
- state explicitly whether the nonce subtree is now truthful shadow or still
  carries any bounded placeholder semantics
