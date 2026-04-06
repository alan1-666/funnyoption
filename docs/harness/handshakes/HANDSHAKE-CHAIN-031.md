# HANDSHAKE-CHAIN-031

## Task

- [TASK-CHAIN-031.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-031.md)

## Thread owner

- commander+worker merged thread

## Reads before coding

- `AGENTS.md`
- `PLAN.md`
- `roles/COMMANDER.md`
- `roles/WORKER.md`
- `PROJECT_MAP.md`
- `THREAD_PROTOCOL.md`
- `docs/harness/tasks/TASK-CHAIN-030.md`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-030.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-030.md`
- `docs/architecture/mode-b-zk-rollup.md`
- `docs/operations/local-full-flow-acceptance.md`
- `cmd/local-lifecycle/**`
- `internal/chain/service/rollup_submitter.go`
- this handshake
- `WORKLOG-CHAIN-031.md`

## Files in scope

- `cmd/local-lifecycle/**`
- `scripts/local-lifecycle.sh`
- `scripts/local-full-flow.sh`
- `docs/operations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-031.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-031.md`

## Inputs from other threads

- `TASK-CHAIN-029` and `TASK-CHAIN-030` landed:
  - accepted submissions can be broadcast locally
  - accepted batches / withdrawals / balances / positions / payouts are now
    materialized into read-truth mirrors
- local full-flow still stops short of proving verifier-eligible rollup
  submission from the canonical trading-key flow by default

## Outputs back to commander

- changed files
- local full-flow rollup-submission evidence
- accepted readback evidence
- validation commands
- residual limitations
- recommended next forced-withdraw / freeze follow-up

## Handoff notes

- keep unchanged:
  - `VerifierContext`
  - `proofData-v1`
  - current Groth16 verifier lane
  - current accepted read-truth tables and query contract
- add only:
  - default verifier-eligible local lifecycle behavior
  - one truthful post-settlement rollup submission step
  - one accepted readback proof in the harness summary
- do not widen into:
  - forced-withdrawal runtime
  - mutable backend write-truth switch
  - a new proof/public-signal contract

## Blockers

- do not silently keep `/api/v1/sessions` as the default harness path
- do not claim local acceptance means production truth has switched

## Status

- completed

## Completion notes

- `cmd/local-lifecycle` now defaults to the verifier-eligible
  `trading-key-oracle` flow instead of legacy `/api/v1/sessions`
- the local full-flow now drives rollup submission after oracle settlement and
  confirms accepted-batch advancement plus accepted balances / positions /
  payouts readback through live API calls
