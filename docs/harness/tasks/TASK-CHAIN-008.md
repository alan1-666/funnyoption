# TASK-CHAIN-008

## Summary

Define the target Mode B architecture for FunnyOption as a `ZK-Rollup`
exchange with offchain operator execution, onchain custody, proof-verified
state transitions, and user exit guarantees.

## Scope

- stay architecture-first for this task; do not jump into prover, contract, or
  runtime implementation before the target contract is explicit
- compare the current FunnyOption architecture against the target Mode B shape:
  - current BSC vault + centralized offchain ledger
  - target rollup-style custody + state-root + batch-proof system
- define the canonical Mode B component boundary:
  - offchain operator services that remain CEX-like
  - new offchain proving / batching services
  - L1 contract surface
- define the canonical state model:
  - balances tree
  - orders / replay-protection tree or equivalent executed-order commitment
  - position / funding / insurance-fund state
  - withdrawal request state
- define the batch truth model:
  - sequencer journal / durable batch input
  - state transition boundary
  - what must be replayable after restart
- define the user-fund lifecycle:
  - deposit
  - slow withdrawal
  - fast withdrawal via LP reimbursement
  - forced withdrawal / freeze / escape hatch
- fix the DA choice explicitly:
  - `ZK-Rollup` only for this architecture lane
  - no validium or external-DA fallback in the first target contract
- define the L1 contract boundary at a design level:
  - verifier
  - state root / state update
  - deposit / withdrawal / claim
  - forced withdrawal / freeze / escape hatch
- define the minimum circuit / proof obligations conceptually:
  - valid state transition
  - balance conservation
  - nonce / replay protection
  - withdrawal / LP reimbursement correctness
- define the migration picture from the current repo:
  - what can stay centralized in a first transitional phase
  - what must be replaced before the system can honestly claim Mode B
- keep this task document- and contract-focused
- if examples of future contract placement are needed, keep them on the repo's
  existing Foundry layout only:
  - `contracts/src`
  - `contracts/test`
  - `contracts/script`
  - do not introduce another Solidity framework

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md](/Users/zhangza/code/funnyoption/docs/architecture/direct-deposit-session-key.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md](/Users/zhangza/code/funnyoption/docs/architecture/order-flow.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md](/Users/zhangza/code/funnyoption/docs/architecture/oracle-settled-crypto-markets.md)
- [/Users/zhangza/code/funnyoption/docs/sql/schema.md](/Users/zhangza/code/funnyoption/docs/sql/schema.md)
- [/Users/zhangza/code/funnyoption/foundry.toml](/Users/zhangza/code/funnyoption/foundry.toml)
- [/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol](/Users/zhangza/code/funnyoption/contracts/src/FunnyVault.sol)
- [/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go](/Users/zhangza/code/funnyoption/internal/api/handler/order_handler.go)
- [/Users/zhangza/code/funnyoption/internal/settlement/service/processor.go](/Users/zhangza/code/funnyoption/internal/settlement/service/processor.go)
- [/Users/zhangza/code/funnyoption/internal/oracle/service/worker.go](/Users/zhangza/code/funnyoption/internal/oracle/service/worker.go)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-008.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-008.md)

## Owned files

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-008.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-008.md`
- optional narrow Foundry-side contract boundary notes only if they directly
  support the architecture handoff
- no broad runtime or contract implementation in this task

## Acceptance criteria

- one canonical design document or design-section update covers:
  - Mode B component map
  - state model
  - batch truth / sequencer-journal model
  - `ZK-Rollup` DA assumptions
  - slow / fast / forced withdrawal state machines
  - L1 contract boundary
  - migration stages from current FunnyOption
- the design states clearly which current services can remain operator-run and
  which current truths must be replaced before the system can honestly claim
  proof-verified settlement
- the design is explicit that current FunnyOption is not yet Mode B
- the design is explicit about exit guarantees and the role of data
  availability
- any optional contract placeholder notes stay within the repo's Foundry layout

## Validation

- docs are internally consistent with the current FunnyOption flow and with the
  already-landed V2 trading-key / oracle-market architecture docs
- the design does not pretend current SQL balances / Kafka settlement already
  satisfy rollup truthfulness
- worklog records the chosen DA mode, withdrawal model, major rejected options,
  and the recommended first implementation tranche

## Dependencies

- current offchain MVP baseline is stable
- current trading-key auth and oracle-settled market baselines are complete

## Handoff

- return changed files, the final Mode B architecture contract, and the
  recommended first implementation tranche
- state explicitly what remains out of scope until after the architecture lane
  is accepted
