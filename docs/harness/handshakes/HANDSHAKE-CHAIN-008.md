# HANDSHAKE-CHAIN-008

## Task

- [TASK-CHAIN-008.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-008.md)

## Thread owner

- chain/architecture design worker

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/architecture/direct-deposit-session-key.md`
- `docs/architecture/order-flow.md`
- `docs/architecture/oracle-settled-crypto-markets.md`
- `docs/sql/schema.md`
- `foundry.toml`
- `contracts/src/FunnyVault.sol`
- `internal/api/handler/order_handler.go`
- `internal/settlement/service/processor.go`
- `internal/oracle/service/worker.go`
- this handshake
- `WORKLOG-CHAIN-008.md`

## Files in scope

- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-008.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-008.md`
- optional narrow Foundry-side contract-boundary notes only if they directly
  support the architecture handoff

## Inputs from other threads

- current FunnyOption already has the target UX direction for one-time wallet
  authorization plus browser-local trading keys, but still keeps centralized
  matching, balances, and settlement
- current crypto markets can auto-resolve from oracle data, but final market
  settlement is still a centralized offchain truth
- commander has now fixed two target product decisions for the Mode B lane:
  - data availability mode must be `ZK-Rollup`
  - withdrawal model must include `slow`, `fast`, and `forced` lanes
- commander wants this lane to stay architecture-first so the team does not
  widen into prover or contract implementation before the state / exit /
  contract truth is explicit

## Outputs back to commander

- changed files
- final Mode B component / state / contract architecture
- recommended first implementation tranche
- residual risks and rejected options
- canonical design doc:
  - `docs/architecture/mode-b-zk-rollup.md`

## Handoff notes

- target architecture is explicitly:
  - offchain operator execution
  - onchain custody
  - proof-verified batch settlement
  - `ZK-Rollup` data availability
  - user exit via forced withdrawal / freeze / escape hatch
- withdrawal lanes to model:
  - slow withdrawal
  - fast withdrawal via LP reimbursement
  - forced withdrawal
- one required design artifact is a migration story from the current BSC-vault
  centralized ledger into the Mode B shape
- landed design decisions for this worker handoff:
  - current FunnyOption is explicitly documented as not yet `Mode B`
  - first truthful DA lane is `ZK-Rollup` with L1-native DA and `calldata`
  - canonical roots are `balances + orders + positions_funding + withdrawals`
  - withdrawal contract includes `slow + fast + forced`
  - `FunnyVault.processClaim()` is explicitly not sufficient for Mode B
  - recommended first implementation tranche is `shadow journal + shadow roots`,
    not prover-first

## Blockers

- do not pretend current SQL balances / Kafka events already satisfy Mode B
- do not widen into prover, verifier, or full contract implementation in this
  task
- do not re-open already-accepted V2 trading-key or oracle-market contracts
  unless the Mode B design proves they must change
- do not introduce another contract toolchain; reuse Foundry if placeholders
  are needed

## Status

- completed
