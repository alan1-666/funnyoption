# TASK-CHAIN-014

## Summary

Implement the first verifier-gated auth/proof tranche: consume canonical
`TRADING_KEY_AUTHORIZED` and `NONCE_ADVANCED.payload.order_authorization`
witness material without reopening the stable `shadow-batch-v1` public-input
shape, and prepare the path from batch metadata to future `FunnyRollupCore`
state-root acceptance.

## Scope

- build directly on `TASK-CHAIN-013`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- implement the first verifier-gated auth/proof lane boundary:
  - make canonical V2 auth witness material prover-consumable
  - bind `TRADING_KEY_AUTHORIZED.authorization_ref` to
    `NONCE_ADVANCED.payload.order_authorization`
  - keep the current monotonic-floor nonce contract from `TASK-CHAIN-012`
  - keep the current `shadow-batch-v1` public-input shape unchanged
- prepare the future `FunnyRollupCore` state-root advancement contract:
  - do not yet implement a full verifier
  - do not yet make production state-root advancement canonical truth
  - define or land the narrowest runtime/code boundary that the later
    verifier-gated acceptance worker can plug into
- migrate any remaining verifier-eligible proof docs/runbooks off deprecated
  `/api/v1/sessions`
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
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-013.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-013.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-013.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-013.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/shared/auth](/Users/zhangza/code/funnyoption/backend/internal/shared/auth)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/backend/internal/api](/Users/zhangza/code/funnyoption/backend/internal/api)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/docs/operations/local-full-flow-acceptance.md](/Users/zhangza/code/funnyoption/docs/operations/local-full-flow-acceptance.md)

## Owned files

- `internal/shared/auth/**`
- `internal/rollup/**`
- `internal/api/**` only where needed for verifier-gated auth/proof prep
- `contracts/src/**` and `contracts/test/**` only for narrow metadata-aligned
  verifier-gate prep if justified
- `docs/architecture/**`
- `docs/sql/**`
- `docs/operations/**` only where verifier-eligible proof tooling docs must be corrected
- `docs/harness/handshakes/HANDSHAKE-CHAIN-014.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-014.md`

## Acceptance criteria

- canonical V2 auth witness material is consumable by the future verifier lane
  without reopening `shadow-batch-v1` public inputs
- the join between `TRADING_KEY_AUTHORIZED` and `NONCE_ADVANCED` is explicit
  and tested
- verifier-eligible proof docs/runbooks no longer point users at deprecated
  `/api/v1/sessions`
- docs remain explicit that production truth is unchanged
- the tranche does not widen into full prover/verifier/runtime rewrite

## Validation

- targeted Go tests for touched auth/rollup/api paths
- any narrow Foundry validation if the contract boundary changes
- verifier-eligible proof-tooling docs/runbook checks
- `git diff --check`

## Dependencies

- `TASK-CHAIN-013` completed

## Handoff

- return changed files, the verifier-gated auth/proof contract, validation
  commands, docs/runbook updates, residual limitations, and the recommended
  next verifier/state-root acceptance tranche
