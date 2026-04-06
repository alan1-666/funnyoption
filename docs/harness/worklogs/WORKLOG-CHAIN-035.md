# WORKLOG-CHAIN-035

### 2026-04-07 02:10 CST

- thread:
  - commander+worker merged
- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/COMMANDER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-034.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-034.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-034.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/api/handler/order_handler.go`
  - `internal/account/service/event_processor.go`
  - `internal/oracle/service/worker.go`
  - `internal/settlement/service/processor.go`
  - `internal/chain/service/rollup_submitter.go`
- changed:
  - created `TASK-CHAIN-035`, `HANDSHAKE-CHAIN-035`, and this worklog for one
    broader frozen-mode mutable-truth guard tranche
  - added one shared API-side frozen rejection helper and applied it to:
    - `CreateMarket`
    - `CreateFirstLiquidity`
    - `CreateClaimPayout`
    - `ResolveMarket`
  - made oracle worker skip resolution writes while frozen
  - made settlement processor skip position/settlement writes while frozen
  - made account service skip order/trade/settlement balance mutations while
    frozen
  - made rollup submitter surface a stable `FROZEN` idle action instead of
    continuing to broadcast batch transactions that would revert onchain
- validated:
  - `go test ./internal/account/service ./internal/api ./internal/api/handler ./internal/matching/service ./internal/oracle/service ./internal/settlement/service ./internal/chain/service ./internal/rollup ./cmd/rollup`
  - `forge test --offline --match-path contracts/test/FunnyRollupCore.t.sol`
  - `git diff --check`
- next:
  - continue toward escape-hatch collateral claims and fuller production-truth
    switching without reopening this widened frozen runtime contract
