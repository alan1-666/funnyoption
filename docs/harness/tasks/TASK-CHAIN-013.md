# TASK-CHAIN-013

## Summary

Add one narrow canonical V2 trading-key auth witness lane that binds
`NONCE_ADVANCED` to verifier-eligible order authorization without reopening the
stable `shadow-batch-v1` public-input shape, and migrate repo proof tooling
away from deprecated `/api/v1/sessions`.

## Scope

- build directly on `TASK-CHAIN-012`
- keep the current product truth unchanged:
  - SQL/Kafka settlement remains production truth
  - direct-vault claim remains production truth
  - do not claim FunnyOption is already Mode B
- add one canonical auth witness lane for future verifier/prover work:
  - bind `NONCE_ADVANCED` to canonical V2 trading-key order authorization
  - make the witness durable or otherwise prover-consumable
  - do not widen current public inputs
- migrate repo-local proof tooling off deprecated `/api/v1/sessions`:
  - lifecycle / harness / staging scripts should no longer depend on the blank-
    vault compatibility baseline for verifier-eligible paths
- keep the first-proof-lane nonce contract from `TASK-CHAIN-012`:
  - monotonic-floor nonce remains acceptable for tranche 1
  - do not rewrite runtime to gapless nonce
- do not implement:
  - prover generation
  - verifier logic
  - production withdrawal claim rewrite
  - forced-withdrawal runtime
  - full rollup contract system

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-012.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-012.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-012.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-012.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-012.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-012.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/backend/internal/api](/Users/zhangza/code/funnyoption/backend/internal/api)
- [/Users/zhangza/code/funnyoption/backend/internal/shared/auth](/Users/zhangza/code/funnyoption/backend/internal/shared/auth)
- [/Users/zhangza/code/funnyoption/backend/cmd/local-lifecycle](/Users/zhangza/code/funnyoption/backend/cmd/local-lifecycle)
- [/Users/zhangza/code/funnyoption/scripts](/Users/zhangza/code/funnyoption/scripts)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)

## Owned files

- `internal/rollup/**`
- `internal/api/**` only where needed for auth witness capture
- `internal/shared/auth/**` only where needed for canonical V2 witness shape
- `cmd/local-lifecycle/**` only if needed to migrate verifier-eligible proof tooling
- `scripts/**` only if needed to migrate verifier-eligible proof tooling
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-013.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-013.md`
- `contracts/src/**` and `contracts/test/**` only for narrow metadata-aligned
  comments/placeholders if justified

## Acceptance criteria

- repo has one narrow canonical V2 trading-key auth witness lane that a later
  prover/verifier worker can consume without reopening public-input shape
- verifier-eligible proof tooling no longer depends on deprecated
  `/api/v1/sessions`
- docs stay explicit that production truth is unchanged
- the tranche does not widen into prover/verifier/runtime rewrite

## Validation

- targeted Go tests for touched rollup/api/auth paths
- harness/proof-tooling validation for migrated verifier-eligible paths
- `git diff --check`

## Dependencies

- `TASK-CHAIN-012` completed

## Handoff

- return changed files, the auth witness contract, validation commands,
  migrated proof-tooling paths, residual limitations, and the recommended
  verifier-gated implementation follow-up
