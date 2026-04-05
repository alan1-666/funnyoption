# TASK-CHAIN-015

## Summary

Add the smallest Foundry-only verifier/state-root acceptance tranche: consume
the stable verifier-gated batch contract from `TASK-CHAIN-014`, keep the
`shadow-batch-v1` public-input shape unchanged, and prepare `FunnyRollupCore`
to reject any batch whose auth proof is not fully `JOINED` without widening
into full prover/verifier or production withdrawal rewrite.

## Scope

- build directly on `TASK-CHAIN-014`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- land the narrowest verifier/state-root acceptance hook:
  - keep consuming `BuildVerifierGateBatchContract(history, batch)` outputs
  - keep `shadow-batch-v1` public inputs unchanged
  - keep `FunnyRollupCore` on the repo's existing Foundry toolchain
  - reject target batches whose auth proof contains
    `MISSING_TRADING_KEY_AUTHORIZED` or `NON_VERIFIER_ELIGIBLE`
  - prepare the state-root advancement boundary without implementing full
    proof verification
- keep verifier/auth scope narrow:
  - do not redesign the tranche-1 monotonic-floor nonce contract
  - do not widen auth witness shape unless strictly required to enforce the
    already-chosen `JOINED` gate
- do not implement:
  - full prover generation
  - full verifier logic
  - production withdrawal claim rewrite
  - forced-withdrawal runtime
  - full rollup contract system

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-014.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-014.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-014.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-014.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-014.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-014.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/internal/rollup](/Users/zhangza/code/funnyoption/internal/rollup)
- [/Users/zhangza/code/funnyoption/internal/shared/auth](/Users/zhangza/code/funnyoption/internal/shared/auth)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/test](/Users/zhangza/code/funnyoption/contracts/test)

## Owned files

- `internal/rollup/**`
- `internal/shared/auth/**` only if the verifier gate needs narrow metadata-aligned helpers
- `contracts/src/**`
- `contracts/test/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-015.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-015.md`

## Acceptance criteria

- repo has one minimal Foundry-only verifier/state-root acceptance hook that
  consumes the stable verifier-gated batch contract from `TASK-CHAIN-014`
- batches with any auth-proof row not equal to `JOINED` are rejected before
  state-root advancement
- `shadow-batch-v1` public-input shape remains unchanged
- docs stay explicit that production truth is unchanged and no full verifier is
  present yet
- the tranche does not widen into prover/verifier/runtime rewrite

## Validation

- targeted Go tests for touched rollup/auth paths
- narrow Foundry tests for `FunnyRollupCore` acceptance boundary
- `git diff --check`

## Dependencies

- `TASK-CHAIN-014` completed

## Handoff

- return changed files, the minimal verifier/state-root acceptance contract,
  validation commands, residual limitations, and the recommended next
  prover/verifier follow-up
