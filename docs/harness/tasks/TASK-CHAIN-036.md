# TASK-CHAIN-036

## Summary

Close the remaining Mode-B critical path in one merged commander+worker thread:
land escape-hatch collateral Merkle claims, push the repo's financial read/runtime
truth further onto the accepted/frozen lane, and replace the current
outer-signal-only proving lane with a state-transition-witness proving contract
that can anchor the escape collateral root used by frozen exits.

## Scope

- build directly on `TASK-CHAIN-035`
- implement:
  - accepted-batch escape collateral root derivation plus durable accepted leaf
    storage
  - one onchain root anchor tied to accepted batches before freeze
  - one frozen escape-claim runtime using Merkle proofs against the last
    anchored accepted collateral root
  - truthful backend/API read surfaces for escape-claimable collateral and
    escape-claim status
  - accepted/frozen financial read-truth follow-through for:
    - payout claim request creation
    - liability reporting
    - accepted balances after escape claims
  - one new proving lane that consumes state-transition witness material rather
    than only outer digest equality, while keeping the current outer
    proof/public-signal envelope stable where possible
- do not implement:
  - emergency resolution of unresolved open markets during freeze
  - multi-circuit aggregation
  - a second Solidity toolchain

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/COMMANDER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/COMMANDER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-035.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-035.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-035.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-035.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-035.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-035.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/backend/internal/rollup](/Users/zhangza/code/funnyoption/backend/internal/rollup)
- [/Users/zhangza/code/funnyoption/backend/internal/api/handler](/Users/zhangza/code/funnyoption/backend/internal/api/handler)
- [/Users/zhangza/code/funnyoption/backend/internal/chain/service](/Users/zhangza/code/funnyoption/backend/internal/chain/service)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupCore.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyRollupVerifier.sol)

## Owned files

- `internal/rollup/**`
- `internal/api/dto/**`
- `internal/api/handler/**`
- `internal/api/routes_reads.go`
- `internal/chain/model/**`
- `internal/chain/service/**`
- `contracts/src/**`
- `contracts/test/**`
- `cmd/rollup/**`
- `migrations/**`
- `scripts/local-chain-up.sh`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-036.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-036.md`

## Acceptance criteria

- accepted batches derive and persist one escape collateral root plus durable
  collateral leaves/proof inputs
- `FunnyRollupCore` can anchor that root before freeze and can execute a frozen
  escape collateral claim via Merkle proof without operator discretion
- API/read surfaces can expose claimable escape collateral and claimed status
- payout claim creation and liability reporting prefer accepted/frozen truth
  rather than legacy settlement/account tables once that truth is visible
- the new preferred proving lane consumes state-transition witness material and
  is no longer just an outer-digest-equality proof
- docs stay explicit about any residual first-cut limitations around unresolved
  open positions or emergency market resolution

## Validation

- `go test ./internal/rollup ./internal/chain/service ./internal/api/handler ./internal/api ./cmd/rollup`
- `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
- one local dev flow that:
  - reaches an accepted batch
  - anchors an escape collateral root
  - freezes the rollup
  - proves and executes one escape collateral claim
- `git diff --check`

## Dependencies

- `TASK-CHAIN-035` completed

## Handoff

- return changed files, escape-claim contract/runtime behavior, validation
  commands, local frozen exit evidence, accepted/frozen truth changes, proving
  lane changes, residual limitations, and whether the repo still carries any
  remaining non-Mode-B truth boundary
