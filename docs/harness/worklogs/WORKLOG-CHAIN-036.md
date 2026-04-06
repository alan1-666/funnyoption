# WORKLOG-CHAIN-036

### 2026-04-07 03:05 CST

- thread:
  - commander+worker merged
- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/COMMANDER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-035.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-035.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-035.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `internal/api/handler/sql_store.go`
  - `internal/chain/service/**`
  - `contracts/src/FunnyRollupCore.sol`
  - `contracts/src/FunnyVault.sol`
  - `contracts/src/FunnyRollupVerifier.sol`
- changed:
  - created `TASK-CHAIN-036`, `HANDSHAKE-CHAIN-036`, and this worklog for the
    merged escape-claim / accepted-truth / proving-lane closeout tranche
- validated:
  - planning/docs only
- next:
  - implement accepted escape collateral roots and frozen Merkle-proof claims
  - then widen accepted/frozen truth
  - then upgrade the proving lane to consume state-transition witness material
