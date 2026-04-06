# WORKLOG-CHAIN-029

### 2026-04-06 21:06 CST

- thread:
  - commander+worker merged
- read:
  - `AGENTS.md`
  - `PLAN.md`
  - `docs/harness/README.md`
  - `docs/harness/roles/COMMANDER.md`
  - `docs/harness/roles/WORKER.md`
  - `docs/harness/PROJECT_MAP.md`
  - `docs/harness/THREAD_PROTOCOL.md`
  - `docs/harness/tasks/TASK-CHAIN-028.md`
  - `docs/harness/handshakes/HANDSHAKE-CHAIN-028.md`
  - `docs/harness/worklogs/WORKLOG-CHAIN-028.md`
  - `docs/architecture/mode-b-zk-rollup.md`
  - `docs/sql/schema.md`
  - `internal/rollup/**`
  - `internal/chain/service/**`
  - `contracts/src/FunnyVault.sol`
  - `scripts/local-full-flow.sh`
- changed:
  - created `TASK-CHAIN-029`, `HANDSHAKE-CHAIN-029`, and this worklog for one
    current-session tranche that turns accepted submissions into a truthful
    local slow-withdraw claim lane and proves one real local broadcast
- validated:
  - scope stays narrow:
    - no full settlement/account production truth switch yet
    - no forced-withdraw freeze / escape hatch yet
    - no proof/public-signal contract rewrite
- blockers:
  - local DB currently has `journal=0`, `batches=0`, `submissions=0`, so one
    real live-broadcast validation still requires an actual local lifecycle run
- next:
  - implement accepted-batch / accepted-withdrawal materialization
  - wire accepted withdrawals into one canonical `WITHDRAWAL_CLAIM` queue
  - run local full-flow -> prepare/submit -> accepted-state verification

### 2026-04-06 23:32 CST

- changed:
  - added accepted-lane materialization:
    - `internal/rollup/accepted.go`
    - `migrations/017_rollup_accepted_lane.sql`
    - `rollup_accepted_batches`
    - `rollup_accepted_withdrawals`
  - extended `internal/rollup/store.go` so accepted submissions now materialize
    durable accepted-batch / accepted-withdrawal records and derive canonical
    `WITHDRAWAL_CLAIM` queue rows only after an accepted withdrawal leaf exists
  - extended `internal/chain/service/sql_store.go`,
    `listener.go`, and `rollup_submitter.go` so:
    - accepted withdrawal claims move through
      `CLAIMABLE -> CLAIM_SUBMITTED -> CLAIMED`
    - claim confirmations are derived from `ClaimProcessed`
    - accepted submissions are re-materialized after acceptance
  - fixed two real local-runtime bugs found during validation:
    - `WithdrawalQueued` `bytes32` ids are now normalized before hitting
      `varchar(64)` storage
    - local anvil fresh starts now reset stale local cursor / rollup runtime
      state before services boot
  - switched `/api/v1/withdrawals` to effective accepted-claim truth, including
    `claim_status`, `claim_tx_hash`, `claim_submitted_at`, `claimed_at`, and
    `last_error`
- validated:
  - `GOCACHE=/tmp/funnyoption-gocache go test ./internal/rollup ./internal/chain/service ./internal/api/handler ./internal/api`
  - `cd web && npm run build`
  - `bash -n scripts/local-chain-up.sh scripts/dev-up.sh`
  - local dev stack boot now logs:
    - `resetting local anvil runtime state`
  - local real onchain evidence:
    - two accepted batches:
      - `rsub_1 -> batch 1`
      - `rsub_2 -> batch 2`
    - one accepted withdrawal:
      - `withdrawal_id=c53197e763f60b3bc077ee6b3b99eaa232e38152e51d0ca1758b24a19b222267`
      - `claim_status=CLAIMED`
      - `claim_tx_hash=b6545d1ae6da02edec05cd331386c04a1c2b793e056eff6e847918b8846bf408`
    - local API proof:
      - `GET /api/v1/withdrawals?user_id=1001&limit=20`
      - returned `status=CLAIMED`
- blockers:
  - full settlement/account production truth still has not switched
  - forced-withdraw / freeze / escape hatch runtime still does not exist
  - prover lane is still not a full state-transition circuit
- next:
  - continue from the now-proven accepted slow-withdraw lane into:
    - forced-withdraw request/runtime foundations
    - fuller state-transition prover inputs
    - broader production-truth switch decisions
