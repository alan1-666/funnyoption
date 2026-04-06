# TASK-CHAIN-031

## Summary

Promote the local full-flow harness into one verifier-eligible end-to-end proof
path: default the local lifecycle runner to the canonical trading-key oracle
flow, drive real rollup submission after settlement, and verify accepted read
truth through local API readbacks.

## Scope

- build directly on `TASK-CHAIN-030`
- keep the repo truth explicit:
  - accepted read surfaces now exist
  - mutable backend writes still remain on current SQL/Kafka services
  - do not claim the product is already full `Mode B`
- implement:
  - default `cmd/local-lifecycle` / `scripts/local-lifecycle.sh` behavior that
    runs the verifier-eligible trading-key oracle flow by default
  - post-settlement `submit-until-idle` execution inside the harness
  - local accepted-batch readback verification after successful rollup
    submission
  - summary output that records rollup submission actions plus accepted-batch
    evidence
- do not implement:
  - forced-withdrawal / freeze / escape hatch runtime
  - a mutable backend write-truth switch
  - a full state-transition prover rewrite

## Inputs to read

- [/Users/zhangza/code/funnyoption/AGENTS.md](/Users/zhangza/code/funnyoption/AGENTS.md)
- [/Users/zhangza/code/funnyoption/PLAN.md](/Users/zhangza/code/funnyoption/PLAN.md)
- [/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md](/Users/zhangza/code/funnyoption/docs/harness/roles/WORKER.md)
- [/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md](/Users/zhangza/code/funnyoption/docs/harness/PROJECT_MAP.md)
- [/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md](/Users/zhangza/code/funnyoption/docs/harness/THREAD_PROTOCOL.md)
- [/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-030.md](/Users/zhangza/code/funnyoption/docs/harness/tasks/TASK-CHAIN-030.md)
- [/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-030.md](/Users/zhangza/code/funnyoption/docs/harness/handshakes/HANDSHAKE-CHAIN-030.md)
- [/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-030.md](/Users/zhangza/code/funnyoption/docs/harness/worklogs/WORKLOG-CHAIN-030.md)
- [/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md](/Users/zhangza/code/funnyoption/docs/architecture/mode-b-zk-rollup.md)
- [/Users/zhangza/code/funnyoption/docs/operations/local-full-flow-acceptance.md](/Users/zhangza/code/funnyoption/docs/operations/local-full-flow-acceptance.md)
- [/Users/zhangza/code/funnyoption/cmd/local-lifecycle/main.go](/Users/zhangza/code/funnyoption/cmd/local-lifecycle/main.go)
- [/Users/zhangza/code/funnyoption/cmd/local-lifecycle/trading_key_oracle_flow.go](/Users/zhangza/code/funnyoption/cmd/local-lifecycle/trading_key_oracle_flow.go)
- [/Users/zhangza/code/funnyoption/internal/chain/service/rollup_submitter.go](/Users/zhangza/code/funnyoption/internal/chain/service/rollup_submitter.go)

## Owned files

- `cmd/local-lifecycle/**`
- `scripts/local-lifecycle.sh`
- `scripts/local-full-flow.sh`
- `docs/operations/**`
- `docs/architecture/**`
- `docs/sql/**`
- `docs/harness/handshakes/HANDSHAKE-CHAIN-031.md`
- `docs/harness/worklogs/WORKLOG-CHAIN-031.md`

## Acceptance criteria

- default local lifecycle behavior no longer depends on deprecated
  `/api/v1/sessions`
- the verifier-eligible local full-flow can:
  - register trading keys
  - trade and settle
  - run rollup submission until idle
  - prove at least one newly accepted batch exists after the flow
- the same flow verifies accepted balances / positions / payouts through live
  API readbacks
- docs explain that this is still a local acceptance harness, not a production
  truth switch

## Validation

- `go test ./cmd/local-lifecycle ./internal/rollup ./internal/chain/service ./internal/api/handler ./internal/api`
- `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
- `./scripts/dev-up.sh`
- `./scripts/local-full-flow.sh`
- `git diff --check`

## Dependencies

- `TASK-CHAIN-030` completed

## Handoff

- return changed files, local full-flow rollup submission evidence, accepted
  readback evidence, residual limitations, and the recommended next follow-up
  for forced-withdrawal / freeze foundations
