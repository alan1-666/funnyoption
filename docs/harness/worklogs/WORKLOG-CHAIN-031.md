# WORKLOG-CHAIN-031

### 2026-04-07 01:12 CST

- thread:
  - commander+worker merged
- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/COMMANDER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-030.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-030.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-030.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/operations/local-full-flow-acceptance.md`
  - `cmd/local-lifecycle/main.go`
  - `cmd/local-lifecycle/trading_key_oracle_flow.go`
  - `internal/chain/service/rollup_submitter.go`
- changed:
  - created `TASK-CHAIN-031`, `HANDSHAKE-CHAIN-031`, and this worklog for one
    current-session tranche that promotes the local full-flow harness into one
    default verifier-eligible trading-key + rollup-submission proof path
- validated:
  - scope stays narrow:
    - no forced-withdraw / freeze runtime yet
    - no mutable backend write-truth switch yet
    - no new proof/public-signal contract
- next:
  - default local lifecycle to the trading-key oracle flow
  - run rollup submission after oracle settlement
  - verify accepted balances / positions / payouts through live API readbacks

### 2026-04-07 01:59 CST

- changed:
  - defaulted `cmd/local-lifecycle` to `trading-key-oracle`
  - extended `trading_key_oracle_flow` so it now:
    - records baseline accepted-batch state
    - runs rollup submission until idle after payout/readback
    - verifies accepted balances / positions / payouts through live API reads
    - prints accepted-batch / submission evidence in the JSON summary
  - updated local full-flow docs to include rollup acceptance and accepted
    readback as part of the canonical local proof path
- validated:
  - `gofmt -w cmd/local-lifecycle/main.go cmd/local-lifecycle/trading_key_oracle_flow.go`
  - `GOCACHE=/tmp/funnyoption-gocache go test ./cmd/local-lifecycle ./internal/rollup ./internal/chain/service ./internal/api/handler ./internal/api`
  - `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
  - local dev runtime:
    - `./scripts/dev-up.sh`
    - `./scripts/local-full-flow.sh`
  - local full-flow result:
    - accepted batches advanced from `0` to `3`
    - latest submission status became `ACCEPTED`
    - the harness re-read accepted balances / positions / payouts after
      submission and passed
- blockers:
  - mutable backend write truth still has not switched to accepted-first
  - forced-withdraw / freeze / escape-hatch runtime still does not exist
  - the prover lane still does not carry a full state-transition circuit
- next:
  - continue into forced-withdraw / freeze foundations now that a verifier-
    eligible local proof path exists end-to-end
